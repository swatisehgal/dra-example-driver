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

type CpuInfo struct {
	uuid  string
	model string
}

type PreparedCpus struct {
	Resources []*CpuInfo
}

type PreparedResources struct {
	Cpu *PreparedCpus
}

func (d PreparedResources) Type() string {
	if d.Cpu != nil {
		return nascrd.CpuResourceType
	}
	return nascrd.UnknownCPUResourceType
}

type AllocatableResourceInfo struct {
	*CpuInfo
}

type ResourceState struct {
	sync.Mutex
	cdi         *CDIHandler
	allocatable AllocatableResources
	prepared    PreparedClaims
}

func NewResourceState(config *Config) (*ResourceState, error) {
	allocatable, err := enumerateAllPossibleDevices()
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

	err = state.syncPreparedDevicesFromCRDSpec(&config.nascrd.Spec)
	if err != nil {
		return nil, fmt.Errorf("unable to sync prepared devices from CRD: %v", err)
	}

	return state, nil
}

func (s *ResourceState) Prepare(claimUID string, allocation nascrd.AllocatedResources) ([]string, error) {
	s.Lock()
	defer s.Unlock()

	if s.prepared[claimUID] != nil {
		return s.cdi.GetClaimDevices(claimUID, s.prepared[claimUID]), nil
	}

	prepared := &PreparedResources{}

	var err error
	switch allocation.Type() {
	case nascrd.CpuResourceType:
		prepared.Cpu, err = s.prepareGpus(claimUID, allocation.CpuResource)
	}
	if err != nil {
		return nil, fmt.Errorf("allocation failed: %v", err)
	}

	err = s.cdi.CreateClaimSpecFile(claimUID, prepared)
	if err != nil {
		return nil, fmt.Errorf("unable to create CDI spec file for claim: %v", err)
	}

	s.prepared[claimUID] = prepared

	return s.cdi.GetClaimDevices(claimUID, s.prepared[claimUID]), nil
}

func (s *ResourceState) Unprepare(claimUID string) error {
	s.Lock()
	defer s.Unlock()

	if s.prepared[claimUID] == nil {
		return nil
	}

	switch s.prepared[claimUID].Type() {
	case nascrd.CpuResourceType:
		err := s.unprepareGpus(claimUID, s.prepared[claimUID])
		if err != nil {
			return fmt.Errorf("unprepare failed: %v", err)
		}
	}

	err := s.cdi.DeleteClaimSpecFile(claimUID)
	if err != nil {
		return fmt.Errorf("unable to delete CDI spec file for claim: %v", err)
	}

	delete(s.prepared, claimUID)

	return nil
}

func (s *ResourceState) GetUpdatedSpec(inspec *nascrd.NodeAllocationStateSpec) *nascrd.NodeAllocationStateSpec {
	s.Lock()
	defer s.Unlock()

	outspec := inspec.DeepCopy()
	s.syncAllocatableDevicesToCRDSpec(outspec)
	s.syncPreparedDevicesToCRDSpec(outspec)
	return outspec
}

func (s *ResourceState) prepareGpus(claimUID string, allocated *nascrd.AllocatedCpus) (*PreparedCpus, error) {
	prepared := &PreparedCpus{}

	for _, device := range allocated.Resources {
		gpuInfo := s.allocatable[device.UUID].CpuInfo

		if _, exists := s.allocatable[device.UUID]; !exists {
			return nil, fmt.Errorf("requested GPU does not exist: %v", device.UUID)
		}

		prepared.Resources = append(prepared.Resources, gpuInfo)
	}

	return prepared, nil
}

func (s *ResourceState) unprepareGpus(claimUID string, devices *PreparedResources) error {
	return nil
}

func (s *ResourceState) syncAllocatableDevicesToCRDSpec(spec *nascrd.NodeAllocationStateSpec) {
	gpus := make(map[string]nascrd.AllocatableResource)
	for _, device := range s.allocatable {
		gpus[device.uuid] = nascrd.AllocatableResource{
			CpuResource: &nascrd.AllocatableCpu{
				UUID:        device.uuid,
				ProductName: device.model,
			},
		}
	}

	var allocatable []nascrd.AllocatableResource
	for _, device := range gpus {
		allocatable = append(allocatable, device)
	}

	spec.AllocatableResources = allocatable
}

func (s *ResourceState) syncPreparedDevicesFromCRDSpec(spec *nascrd.NodeAllocationStateSpec) error {
	gpus := s.allocatable

	prepared := make(PreparedClaims)
	for claim, devices := range spec.PreparedClaims {
		switch devices.Type() {
		case nascrd.CpuResourceType:
			prepared[claim] = &PreparedResources{}
			for _, d := range devices.CpuResource.Resources {
				prepared[claim].Cpu.Resources = append(prepared[claim].Cpu.Resources, gpus[d.UUID].CpuInfo)
			}
		}
	}

	s.prepared = prepared
	return nil
}

func (s *ResourceState) syncPreparedDevicesToCRDSpec(spec *nascrd.NodeAllocationStateSpec) {
	outcas := make(map[string]nascrd.PreparedResources)
	for claim, resources := range s.prepared {
		var prepared nascrd.PreparedResources
		switch resources.Type() {
		case nascrd.CpuResourceType:
			prepared.CpuResource = &nascrd.PreparedCpus{}
			for _, device := range resources.Cpu.Resources {
				outdevice := nascrd.PreparedCpu{
					UUID: device.uuid,
				}
				prepared.CpuResource.Resources = append(prepared.CpuResource.Resources, outdevice)
			}
		}
		outcas[claim] = prepared
	}
	spec.PreparedClaims = outcas
}