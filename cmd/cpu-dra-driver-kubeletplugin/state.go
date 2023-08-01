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
	"context"
	"fmt"
	"sync"

	nascrd "github.com/kubernetes-sigs/dra-example-driver/api/example.com/resource/cpu/nas/v1alpha1"
	cpusetupdaterpb "github.com/kubernetes-sigs/dra-example-driver/pkg/cpusetupdater"
	"k8s.io/klog/v2"
	"k8s.io/utils/cpuset"

	"github.com/jaypipes/ghw"
)

type AllocatableResources map[string]*AllocatableResourceInfo
type PreparedClaims map[string]*PreparedResources

type CPUInfo struct {
	uuid   string // This is going to be CPUId
	cpuID  int
	model  string
	numaID int
	coreID int
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
	klog.Infof("NewResourceState called")
	topology, err := ghw.Topology(ghw.WithPathOverrides(ghw.PathOverrides{
		"/sys": *config.flags.sysFsRoot,
	}))
	if err != nil {
		return nil, err
	}

	cpuInfo, err := ghw.CPU()
	if err != nil {
		return nil, err
	}

	allocatable, err := enumerateAllCPUs(topology, cpuInfo, config.flags.reservedCPUs)
	if err != nil {
		return nil, fmt.Errorf("error enumerating all possible devices: %v", err)
	}

	klog.Infof("state.go: allocatable: %+v", allocatable)
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
	klog.Infof("state.go: Prepare called claimUID %+v", claimUID)
	s.Lock()
	defer s.Unlock()

	if s.prepared[claimUID] != nil {
		cdiDevices, err := s.cdi.GetClaimDevices(claimUID, s.prepared[claimUID])
		if err != nil {
			return nil, fmt.Errorf("unable to get CDI devices names: %v", err)
		}
		klog.Infof("state.go: Prepare: cdiDevices: %+v", cdiDevices)
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
	klog.Infof("state.go: Unprepare called claimUID %+v", claimUID)
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
	klog.Infof("state.go: GetUpdatedSpec called")
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
	klog.Infof("state.go: prepareCPUs called %+v", claimUID)

	klog.Infof("state.go: preparing resources for claim %+v", claimUID)
	prepared := &PreparedCPUs{}

	klog.Infof("state.go: prepareCPUs called allocated.Resources:%+v", allocated.Resources)
	for _, device := range allocated.Resources {
		klog.Infof("UPDATE: state.go: prepareCPUs device %+v", device)
		klog.Infof("UPDATE: state.go: prepareCPUs s.allocatable[%s] %+v", device.UUID, s.allocatable[device.UUID])

		cpuInfo := s.allocatable[device.UUID].CPUInfo

		if _, exists := s.allocatable[device.UUID]; !exists {
			return nil, fmt.Errorf("requested cpu does not exist: %v", device.UUID)
		}

		prepared.Resources = append(prepared.Resources, cpuInfo)
	}

	return prepared, nil
}

func (s *ResourceState) unprepareCPUs(claimUID string, devices *PreparedResources) error {
	klog.Infof("state.go: unprepareCpus called  %+v", claimUID)
	return nil
}

func (s *ResourceState) syncAllocatableDevicesToCRDSpec(spec *nascrd.NodeAllocationStateSpec) error {
	klog.Infof("state.go: syncAllocatableDevicesToCRDSpec called")
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
	klog.Infof("state.go: syncPreparedDevicesFromCRDSpec called")
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
	klog.Infof("state.go: syncPreparedDevicesToCRDSpec called")
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

func (s *ResourceState) UpdateCPUSet(c context.Context, r *cpusetupdaterpb.CpusetRequest) (*cpusetupdaterpb.CpusetResponse, error) {
	klog.Info("UpdateCPUSet Server: UpdateCPUSet called")

	klog.Infof("r.Version: %v", r.Version)
	klog.Infof("Add another CPU id %qto the list", 4)
	klog.Infof("r.ResourceName: %v", r.ResourceName)

	klog.Infof("r.NodeName: %v", r.GetNodeName())

	cpus, err := cpuset.Parse(r.Version + ",4")
	if err != nil {
		return nil, err
	}

	resp := &cpusetupdaterpb.CpusetResponse{
		Cpuset: cpus.String(),
	}

	return resp, nil
}
