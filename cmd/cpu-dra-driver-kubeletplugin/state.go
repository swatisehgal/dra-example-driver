/*
 * Copyright 2023 The Kubernetes Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"fmt"
	"sync"

	nascrd "github.com/kubernetes-sigs/dra-example-driver/api/example.com/resource/cpu/nas/v1alpha1"
)

type AllocatableResources map[string]*AllocatableResourceInfo
type PreparedClaims map[string]*PreparedResources

type CPUInfo struct {
	uuid  string
	model string
}

type PreparedCPUs struct {
	Resources []*CPUInfo
}

type PreparedResources struct {
	CPU *PreparedCPUs
}

func (d PreparedResources) Type() string {
	if d.CPU != nil {
		return nascrd.CPUResourceType
	}
	return nascrd.UnknownCPUResourceType
}

type AllocatableResourceInfo struct {
	*CPUInfo
}

type ResourceState struct {
	sync.Mutex
	cdi         *CDIHandler
	allocatable AllocatableResources
	prepared    PreparedClaims
}

func NewResourceState(config *Config) (*ResourceState, error) {
	allocatable, err := enumerateAllPossibleCPUs()
	if err != nil {
		return nil, fmt.Errorf("error enumerating all possible devices: %v", err)
	}

	cdi, err := NewCDIHandler(config)
	if err != nil {
		return nil, fmt.Errorf("unable to create CDI handler: %v", err)
	}

	err = cdi.CreateCommonSpecFile()
	if err != nil {
		return nil, fmt.Errorf("unable to create CDI spec file for common edits: %v", err)
	}

	state := &ResourceState{
		cdi:         cdi,
		allocatable: allocatable,
		prepared:    make(PreparedClaims),
	}

	// err = state.syncPreparedDevicesFromCRDSpec(&config.nascrd.Spec)
	// if err != nil {
	// 	return nil, fmt.Errorf("unable to sync prepared devices from CRD: %v", err)
	// }

	return state, nil
}

func (s *ResourceState) Prepare(claimUID string, allocation nascrd.AllocatedResources) ([]string, error) {
	s.Lock()
	defer s.Unlock()

	if s.prepared[claimUID] != nil {
		cdiDevices, err := s.cdi.GetClaimDevices(claimUID, s.prepared[claimUID])
		if err != nil {
			return nil, fmt.Errorf("unable to get CDI devices names: %v", err)
		}
		return cdiDevices, nil
	}

	prepared := &PreparedResources{}

	var err error
	switch allocation.Type() {
	case nascrd.CPUResourceType:
		prepared.CPU, err = s.prepareCPUs(claimUID, allocation.CPUResource)
	default:
		err = fmt.Errorf("unknown device type: %v", allocation.Type())
	}
	if err != nil {
		return nil, fmt.Errorf("allocation failed: %v", err)
	}

	err = s.cdi.CreateClaimSpecFile(claimUID, prepared)
	if err != nil {
		return nil, fmt.Errorf("unable to create CDI spec file for claim: %v", err)
	}

	s.prepared[claimUID] = prepared

	cdiDevices, err := s.cdi.GetClaimDevices(claimUID, s.prepared[claimUID])
	if err != nil {
		return nil, fmt.Errorf("unable to get CDI devices names: %v", err)
	}
	return cdiDevices, nil
}

func (s *ResourceState) Unprepare(claimUID string) error {
	s.Lock()
	defer s.Unlock()

	if s.prepared[claimUID] == nil {
		return nil
	}

	switch s.prepared[claimUID].Type() {
	case nascrd.CPUResourceType:
		err := s.unprepareCPUs(claimUID, s.prepared[claimUID])
		if err != nil {
			return fmt.Errorf("unprepare failed: %v", err)
		}
	default:
		return fmt.Errorf("unknown device type: %v", s.prepared[claimUID].Type())
	}

	err := s.cdi.DeleteClaimSpecFile(claimUID)
	if err != nil {
		return fmt.Errorf("unable to delete CDI spec file for claim: %v", err)
	}

	delete(s.prepared, claimUID)

	return nil
}

func (s *ResourceState) GetUpdatedSpec(inspec *nascrd.NodeAllocationStateSpec) (*nascrd.NodeAllocationStateSpec, error) {
	s.Lock()
	defer s.Unlock()

	outspec := inspec.DeepCopy()
	err := s.syncAllocatableDevicesToCRDSpec(outspec)
	if err != nil {
		return nil, fmt.Errorf("synching allocatable devices to CR spec: %v", err)
	}

	err = s.syncPreparedDevicesToCRDSpec(outspec)
	if err != nil {
		return nil, fmt.Errorf("synching prepared devices to CR spec: %v", err)
	}

	return outspec, nil
}

func (s *ResourceState) prepareCPUs(claimUID string, allocated *nascrd.AllocatedCPUs) (*PreparedCPUs, error) {
	prepared := &PreparedCPUs{}

	for _, device := range allocated.Resources {
		cpuInfo := s.allocatable[device.UUID].CPUInfo

		if _, exists := s.allocatable[device.UUID]; !exists {
			return nil, fmt.Errorf("requested cpu does not exist: %v", device.UUID)
		}

		prepared.Resources = append(prepared.Resources, cpuInfo)
	}

	return prepared, nil
}

func (s *ResourceState) unprepareCPUs(claimUID string, devices *PreparedResources) error {
	return nil
}

func (s *ResourceState) syncAllocatableDevicesToCRDSpec(spec *nascrd.NodeAllocationStateSpec) error {
	cpus := make(map[string]nascrd.AllocatableResource)
	for _, device := range s.allocatable {
		cpus[device.uuid] = nascrd.AllocatableResource{
			CPUResource: &nascrd.AllocatableCPU{
				UUID:        device.uuid,
				ProductName: device.model,
			},
		}
	}

	var allocatable []nascrd.AllocatableResource
	for _, device := range cpus {
		allocatable = append(allocatable, device)
	}

	spec.AllocatableResources = allocatable

	return nil
}

func (s *ResourceState) syncPreparedDevicesFromCRDSpec(spec *nascrd.NodeAllocationStateSpec) error {
	cpus := s.allocatable

	prepared := make(PreparedClaims)
	for claim, devices := range spec.PreparedClaims {
		switch devices.Type() {
		case nascrd.CPUResourceType:
			prepared[claim] = &PreparedResources{}
			for _, d := range devices.CPUResource.Resources {
				prepared[claim].CPU.Resources = append(prepared[claim].CPU.Resources, cpus[d.UUID].CPUInfo)
			}
		default:
			return fmt.Errorf("unknown device type: %v", devices.Type())
		}

	}

	s.prepared = prepared

	return nil
}

func (s *ResourceState) syncPreparedDevicesToCRDSpec(spec *nascrd.NodeAllocationStateSpec) error {
	outcas := make(map[string]nascrd.PreparedResources)
	for claim, resources := range s.prepared {
		var prepared nascrd.PreparedResources
		switch resources.Type() {
		case nascrd.CPUResourceType:
			prepared.CPUResource = &nascrd.PreparedCPUs{}
			for _, device := range resources.CPU.Resources {
				outdevice := nascrd.PreparedCPU{
					UUID: device.uuid,
				}
				prepared.CPUResource.Resources = append(prepared.CPUResource.Resources, outdevice)
			}
		default:
			return fmt.Errorf("unknown device type: %v", resources.Type())
		}
		outcas[claim] = prepared
	}
	spec.PreparedClaims = outcas
	return nil
}
