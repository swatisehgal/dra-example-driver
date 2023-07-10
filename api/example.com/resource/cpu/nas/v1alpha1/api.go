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
	"k8s.io/klog/v2"
)

const (
	GroupName = "nas.cpu.resource.example.com"
	Version   = "v1alpha1"

	CPUResourceType        = "cpu"
	UnknownCPUResourceType = "unknown"

	NodeAllocationStateStatusReady    = "Ready"
	NodeAllocationStateStatusNotReady = "NotReady"
)

type NodeAllocationStateConfig struct {
	Name      string
	Namespace string
	Owner     *metav1.OwnerReference
}

func NewNodeAllocationState(config *NodeAllocationStateConfig) *NodeAllocationState {
	klog.Infof("nas/v1alpha1/api.go: NewNodeAllocationState called :%+v", config)
	nascrd := &NodeAllocationState{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.Name,
			Namespace: config.Namespace,
		},
	}

	if config.Owner != nil {
		nascrd.OwnerReferences = []metav1.OwnerReference{*config.Owner}
	}

	return nascrd
}
