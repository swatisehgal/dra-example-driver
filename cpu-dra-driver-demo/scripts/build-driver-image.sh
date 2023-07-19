#!/usr/bin/env bash

# Copyright 2023 The Kubernetes Authors.
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

# This scripts invokes `kind build image` so that the resulting
# image has a containerd with CDI support.
#
# Usage: kind-build-image.sh <tag of generated image>

# A reference to the current directory where this script is located
CURRENT_DIR="$(cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd)"

set -ex
set -o pipefail

source "${CURRENT_DIR}/common.sh"

# Create a temorary directory to hold all the artifacts we need for building the image
TMP_DIR="$(mktemp -d)"
cleanup() {
    rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

# Go back to the root directory of this repo
cd ${CURRENT_DIR}/../..

# Set build variables
export REGISTRY="${CPU_DRA_DRIVER_REGISTRY}"
export IMAGE="${CPU_DRA_DRIVER_IMAGE_NAME}"
export VERSION="${DRIVER_IMAGE_TAG}"

# Regenerate the CRDs and build the container image
CMD_TARGETS=cpu-dra-driver-controller,cpu-dra-driver-controller,cpu-nri-plugin IMAGE_NAME=quay.io/swsehgal/cpu-dra-driver  BUILDIMAGE=cpu-dra-driver-build:golang1.20.3 make docker-generate
CMD_TARGETS=cpu-dra-driver-controller,cpu-dra-driver-controller,cpu-nri-plugin IMAGE_NAME=quay.io/swsehgal/cpu-dra-driver  BUILDIMAGE=cpu-dra-driver-build:golang1.20.3 make -f deployments/container/Makefile "${DRIVER_IMAGE_PLATFORM}"
