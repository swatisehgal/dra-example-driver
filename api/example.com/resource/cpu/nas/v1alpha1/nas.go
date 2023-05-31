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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AllocatableCpu represents an allocatable CPU on a node.
type AllocatableCpu struct {
	UUID        string `json:"uuid"`
	ProductName string `json:"productName"`
}

// AllocatableResource represents an allocatable device on a node.
type AllocatableResource struct {
	CpuResource *AllocatableCpu `json:"cpuResource,omitempty"`
}

// Type returns the type of AllocatableResource this represents.
func (d AllocatableResource) Type() string {
	if d.CpuResource != nil {
		return CpuResourceType
	}
	return UnknownCPUResourceType
}

// AllocatedCpu represents an allocated GPU.
type AllocatedCpu struct {
	UUID string `json:"uuid,omitempty"`
}

// AllocatedCpus represents a set of allocated GPUs.
type AllocatedCpus struct {
	Resources []AllocatedCpu `json:"resources"`
}

// AllocatedDevices represents a set of allocated devices.
type AllocatedResources struct {
	CpuResource *AllocatedCpus `json:"cpuResource,omitempty"`
}

// Type returns the type of AllocatedDevices this represents.
func (r AllocatedResources) Type() string {
	if r.CpuResource != nil {
		return CpuResourceType
	}
	return UnknownCPUResourceType
}

// PreparedCpu represents a prepared GPU on a node.
type PreparedCpu struct {
	UUID string `json:"uuid"`
}

// PreparedCpus represents a set of prepared GPUs on a node.
type PreparedCpus struct {
	Resources []PreparedCpu `json:"resources"`
}

// PreparedDevices represents a set of prepared devices on a node.
type PreparedResources struct {
	CpuResource *PreparedCpus `json:"cpuResource,omitempty"`
}

// Type returns the type of PreparedDevices this represents.
func (d PreparedResources) Type() string {
	if d.CpuResource != nil {
		return CpuResourceType
	}
	return UnknownCPUResourceType
}

// NodeAllocationStateSpec is the spec for the NodeAllocationState CRD.
type NodeAllocationStateSpec struct {
	AllocatableResources []AllocatableResource         `json:"allocatableResources,omitempty"`
	AllocatedClaims      map[string]AllocatedResources `json:"allocatedClaims,omitempty"`
	PreparedClaims       map[string]PreparedResources  `json:"preparedClaims,omitempty"`
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:resource:singular=nas

// NodeAllocationState holds the state required for allocation on a node.
type NodeAllocationState struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeAllocationStateSpec `json:"spec,omitempty"`
	Status string                  `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeAllocationStateList represents the "plural" of a NodeAllocationState CRD object.
type NodeAllocationStateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []NodeAllocationState `json:"items"`
}
