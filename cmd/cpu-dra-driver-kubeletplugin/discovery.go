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
	"math/rand"
	"os"

	"github.com/google/uuid"
	"github.com/jaypipes/ghw"
	"k8s.io/klog/v2"
	"k8s.io/utils/cpuset"
)

func enumerateAllCPUs(topologyInfo *ghw.TopologyInfo, cpuInfo *ghw.CPUInfo, reservedCPUs *string) (AllocatableResources, error) {
	klog.Infof("enumerateAllCPUs called")
	reservedCpuset, err := cpuset.Parse(*reservedCPUs)
	if err != nil {
		return nil, err
	}
	klog.Infof("reservedCpuset %+v", reservedCpuset)

	numCPUs := getTotalCPUs(topologyInfo)
	seed := os.Getenv("NODE_NAME")
	uuids := generateUUIDs(seed, numCPUs)
	allCPUs := make(AllocatableResources)
	cpuNum := 0
	for _, node := range topologyInfo.Nodes {
		for _, core := range node.Cores {
			for _, cpu := range core.LogicalProcessors {
				if reservedCpuset.Contains(cpu) {
					klog.Infof("Skipping reserved CPU: %d", cpu)
					continue
				}
				cpuUUID := fmt.Sprintf("%s-%d-%d-%d", uuids[cpuNum], node.ID, core.ID, cpu)
				cpuInfo := &AllocatableResourceInfo{
					CPUInfo: &CPUInfo{
						uuid:   cpuUUID,
						cpuID:  cpu,
						model:  cpuInfo.Processors[0].Model,
						numaID: node.ID,
						coreID: core.ID,
					},
				}
				allCPUs[cpuUUID] = cpuInfo
				cpuNum++
			}
		}
	}
	return allCPUs, nil
}

func getTotalCPUs(topo *ghw.TopologyInfo) int {
	logicalCores := 0
	for _, node := range topo.Nodes {
		nodeSrc := findNodeByID(topo.Nodes, node.ID)
		for _, core := range nodeSrc.Cores {
			logicalCores += len(core.LogicalProcessors)
		}
	}
	return int(logicalCores)
}

func findNodeByID(nodes []*ghw.TopologyNode, nodeID int) *ghw.TopologyNode {
	for _, node := range nodes {
		if node.ID == nodeID {
			return node
		}
	}
	return nil
}

func generateUUIDs(seed string, count int) []string {
	rand := rand.New(rand.NewSource(hash(seed)))

	uuids := make([]string, count)
	for i := 0; i < count; i++ {
		charset := make([]byte, 16)
		rand.Read(charset)
		uuid, _ := uuid.FromBytes(charset)
		uuids[i] = "CPU-" + uuid.String()
	}

	return uuids
}

func hash(s string) int64 {
	h := int64(0)
	for _, c := range s {
		h = 31*h + int64(c)
	}
	return h
}
