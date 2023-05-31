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

	corev1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/api/resource/v1alpha2"
	"k8s.io/dynamic-resource-allocation/controller"

	nascrd "github.com/kubernetes-sigs/dra-example-driver/api/example.com/resource/cpu/nas/v1alpha1"
	cpucrd "github.com/kubernetes-sigs/dra-example-driver/api/example.com/resource/cpu/v1alpha1"
)

type cpudriver struct {
	PendingAllocatedClaims *PerNodeAllocatedClaims
}

func NewCpuDriver() *cpudriver {
	return &cpudriver{
		PendingAllocatedClaims: NewPerNodeAllocatedClaims(),
	}
}

func (g *cpudriver) ValidateClaimParameters(claimParams *cpucrd.CpuClaimParametersSpec) error {
	if claimParams.Count < 1 {
		return fmt.Errorf("invalid number of GPUs requested: %v", claimParams.Count)
	}
	return nil
}

func (g *cpudriver) Allocate(crd *nascrd.NodeAllocationState, claim *resourcev1.ResourceClaim, claimParams *cpucrd.CpuClaimParametersSpec, class *resourcev1.ResourceClass, classParams *cpucrd.ResourceClassParametersSpec, selectedNode string) (OnSuccessCallback, error) {
	claimUID := string(claim.UID)

	if !g.PendingAllocatedClaims.Exists(claimUID, selectedNode) {
		return nil, fmt.Errorf("no allocations generated for claim '%v' on node '%v' yet", claim.UID, selectedNode)
	}

	crd.Spec.AllocatedClaims[claimUID] = g.PendingAllocatedClaims.Get(claimUID, selectedNode)
	onSuccess := func() {
		g.PendingAllocatedClaims.Remove(claimUID)
	}

	return onSuccess, nil
}

func (g *cpudriver) Deallocate(crd *nascrd.NodeAllocationState, claim *resourcev1.ResourceClaim) error {
	g.PendingAllocatedClaims.Remove(string(claim.UID))
	return nil
}

func (g *cpudriver) UnsuitableNode(crd *nascrd.NodeAllocationState, pod *corev1.Pod, gpucas []*controller.ClaimAllocation, allcas []*controller.ClaimAllocation, potentialNode string) error {
	g.PendingAllocatedClaims.VisitNode(potentialNode, func(claimUID string, allocation nascrd.AllocatedResources) {
		if _, exists := crd.Spec.AllocatedClaims[claimUID]; exists {
			g.PendingAllocatedClaims.Remove(claimUID)
		} else {
			crd.Spec.AllocatedClaims[claimUID] = allocation
		}
	})

	allocated := g.allocate(crd, pod, gpucas, allcas, potentialNode)
	for _, ca := range gpucas {
		claimUID := string(ca.Claim.UID)
		claimParams, _ := ca.ClaimParameters.(*cpucrd.CpuClaimParametersSpec)

		if claimParams.Count != len(allocated[claimUID]) {
			for _, ca := range allcas {
				ca.UnsuitableNodes = append(ca.UnsuitableNodes, potentialNode)
			}
			return nil
		}

		var resources []nascrd.AllocatedCpu
		for _, cpu := range allocated[claimUID] {
			resource := nascrd.AllocatedCpu{
				UUID: cpu,
			}
			resources = append(resources, resource)
		}

		allocatedDevices := nascrd.AllocatedResources{
			CpuResource: &nascrd.AllocatedCpus{
				Resources: resources,
			},
		}

		g.PendingAllocatedClaims.Set(claimUID, potentialNode, allocatedDevices)
		crd.Spec.AllocatedClaims[claimUID] = allocatedDevices
	}

	return nil
}

func (g *cpudriver) allocate(crd *nascrd.NodeAllocationState, pod *corev1.Pod, gpucas []*controller.ClaimAllocation, allcas []*controller.ClaimAllocation, node string) map[string][]string {
	available := make(map[string]*nascrd.AllocatableCpu)

	for _, resource := range crd.Spec.AllocatableResources {
		switch resource.Type() {
		case nascrd.CpuResourceType:
			available[resource.CpuResource.UUID] = resource.CpuResource
		}
	}

	for _, allocation := range crd.Spec.AllocatedClaims {
		switch allocation.Type() {
		case nascrd.CpuResourceType:
			for _, resource := range allocation.CpuResource.Resources {
				delete(available, resource.UUID)
			}
		}
	}

	allocated := make(map[string][]string)
	for _, ca := range gpucas {
		claimUID := string(ca.Claim.UID)
		if _, exists := crd.Spec.AllocatedClaims[claimUID]; exists {
			devices := crd.Spec.AllocatedClaims[claimUID].CpuResource.Resources
			for _, device := range devices {
				allocated[claimUID] = append(allocated[claimUID], device.UUID)
			}
			continue
		}

		claimParams, _ := ca.ClaimParameters.(*cpucrd.CpuClaimParametersSpec)
		var devices []string
		for i := 0; i < claimParams.Count; i++ {
			for _, device := range available {
				devices = append(devices, device.UUID)
				delete(available, device.UUID)
				break
			}
		}
		allocated[claimUID] = devices
	}

	return allocated
}
