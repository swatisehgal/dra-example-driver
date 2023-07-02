# Copyright 2022 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

GOLANG_VERSION ?= 1.20.3

DRIVER_NAME := dra-example-driver
MODULE := github.com/kubernetes-sigs/$(DRIVER_NAME)
CPU_DRIVER_NAME := cpu-dra-driver

VERSION  ?= v0.1.0
vVERSION := v$(VERSION:v%=%)

VENDOR := example.com
APIS := gpu/nas/v1alpha1 gpu/v1alpha1
CPUAPIS := cpu/nas/v1alpha1 cpu/v1alpha1

PLURAL_EXCEPTIONS  = DeviceClassParameters:DeviceClassParameters
PLURAL_EXCEPTIONS += GpuClaimParameters:GpuClaimParameters
PLURAL_EXCEPTIONS += CPUResourceClassParameters:CPUResourceClassParameters
PLURAL_EXCEPTIONS += CPUClaimParameters:CPUClaimParameters

ifeq ($(IMAGE_NAME),)
REGISTRY ?= registry.example.com
IMAGE_NAME = $(REGISTRY)/$(DRIVER_NAME)
endif
