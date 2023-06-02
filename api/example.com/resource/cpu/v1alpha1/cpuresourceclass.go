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

// CpuResourceSelector allows one to match on a specific type of Device as part of the class.
type CpuResourceSelector struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

// CpuResourceClassParametersSpec is the spec for the DeviceClassParametersSpec CRD.
type CpuResourceClassParametersSpec struct {
	CpuResourceSelector []CpuResourceSelector `json:"cpuResourceSelector,omitempty"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:resource:scope=Cluster

// CpuResourceClassParameters holds the set of parameters provided when creating a resource class for this driver.
type CpuResourceClassParameters struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec CpuResourceClassParametersSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CpuResourceClassParametersList represents the "plural" of a ResourceClassParameters CRD object.
type CpuResourceClassParametersList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []CpuResourceClassParameters `json:"items"`
}
