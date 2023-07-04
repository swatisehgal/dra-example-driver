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
	"os"

	cdiapi "github.com/container-orchestrated-devices/container-device-interface/pkg/cdi"
	cdispec "github.com/container-orchestrated-devices/container-device-interface/specs-go"
	nascrd "github.com/kubernetes-sigs/dra-example-driver/api/example.com/resource/cpu/nas/v1alpha1"
)

const (
	cdiVendor = "k8s." + DriverName
	cdiClass  = "cpu"
	cdiKind   = cdiVendor + "/" + cdiClass

	cdiCommonCpuName = "common-cpus"
)

type CDIHandler struct {
	registry cdiapi.Registry
}

func NewCDIHandler(config *Config) (*CDIHandler, error) {
	registry := cdiapi.GetRegistry(
		cdiapi.WithSpecDirs(*config.flags.cdiRoot),
	)

	err := registry.Refresh()
	if err != nil {
		return nil, fmt.Errorf("unable to refresh the CDI registry: %v", err)
	}

	handler := &CDIHandler{
		registry: registry,
	}

	return handler, nil
}

func (cdi *CDIHandler) GetDevice(device string) *cdiapi.Device {
	return cdi.registry.DeviceDB().GetDevice(device)
}

func (cdi *CDIHandler) CreateCommonSpecFile() error {
	spec := &cdispec.Spec{
		Kind: cdiKind,
		Cpus: []cdispec.Cpu{
			{
				Name: cdiCommonCpuName,
				ContainerEdits: cdispec.CPUContainerEdits{
					Env: []string{
						fmt.Sprintf("CPU_NODE_NAME=%s", os.Getenv("NODE_NAME")),
						fmt.Sprintf("DRA_RESOURCE_DRIVER_NAME=%s", DriverName),
					},
				},
			},
		},
	}

	minVersion, err := cdiapi.MinimumRequiredVersion(spec)
	if err != nil {
		return fmt.Errorf("failed to get minimum required CDI spec version: %v", err)
	}
	spec.Version = minVersion

	specName, err := cdiapi.GenerateNameForTransientSpec(spec, cdiCommonCpuName)
	if err != nil {
		return fmt.Errorf("failed to generate Spec name: %w", err)
	}

	return cdi.registry.SpecDB().WriteSpec(spec, specName)
}

func (cdi *CDIHandler) CreateClaimSpecFile(claimUID string, devices *PreparedResources) error {
	specName := cdiapi.GenerateTransientSpecName(cdiVendor, cdiClass, claimUID)

	spec := &cdispec.Spec{
		Kind:    cdiKind,
		Devices: []cdispec.Device{},
	}

	gpuIdx := 0
	switch devices.Type() {
	case nascrd.CPUResourceType:
		for _, device := range devices.CPU.Resources {
			cdiDevice := cdispec.Device{
				Name: device.uuid,
				ContainerEdits: cdispec.ContainerEdits{
					Env: []string{
						fmt.Sprintf("GPU_DEVICE_%d=%s", gpuIdx, device.uuid),
					},
				},
			}
			spec.Devices = append(spec.Devices, cdiDevice)
			gpuIdx++
		}
	default:
		return fmt.Errorf("unknown device type: %v", devices.Type())
	}

	minVersion, err := cdiapi.MinimumRequiredVersion(spec)
	if err != nil {
		return fmt.Errorf("failed to get minimum required CDI spec version: %v", err)
	}
	spec.Version = minVersion

	return cdi.registry.SpecDB().WriteSpec(spec, specName)
}

func (cdi *CDIHandler) DeleteClaimSpecFile(claimUID string) error {
	specName := cdiapi.GenerateTransientSpecName(cdiVendor, cdiClass, claimUID)
	return cdi.registry.SpecDB().RemoveSpec(specName)
}

func (cdi *CDIHandler) GetClaimDevices(claimUID string, devices *PreparedResources) ([]string, error) {
	cdiDevices := []string{
		cdiapi.QualifiedName(cdiVendor, cdiClass, cdiCommonCpuName),
	}

	switch devices.Type() {
	case nascrd.CPUResourceType:
		for _, device := range devices.CPU.Resources {
			cdiDevice := cdiapi.QualifiedName(cdiVendor, cdiClass, device.uuid)
			cdiDevices = append(cdiDevices, cdiDevice)
		}
	default:
		return nil, fmt.Errorf("unknown device type: %v", devices.Type())
	}

	return cdiDevices, nil
}
