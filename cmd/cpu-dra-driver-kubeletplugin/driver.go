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
	"net"
	"os"
	"path/filepath"
	"sync"

	cpusetupdaterpb "github.com/kubernetes-sigs/dra-example-driver/pkg/cpusetupdater"
	"google.golang.org/grpc"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	drapbv1 "k8s.io/kubelet/pkg/apis/dra/v1alpha2"
	"k8s.io/utils/cpuset"

	nascrd "github.com/kubernetes-sigs/dra-example-driver/api/example.com/resource/cpu/nas/v1alpha1"
	nasclient "github.com/kubernetes-sigs/dra-example-driver/api/example.com/resource/cpu/nas/v1alpha1/client"
)

var _ drapbv1.NodeServer = &driver{}

type driverCPUResourceData struct {
	info map[string]resourceInfo // namespace to podinfo mapping
}
type resourceInfo struct {
	podName       string
	containerInfo map[string]string // mapping between containers and cpusets allocated
}

type driver struct {
	nascrd    *nascrd.NodeAllocationState
	nasclient *nasclient.Client
	state     *ResourceState
	server    *grpc.Server
	wg        sync.WaitGroup
	cpuData   *driverCPUResourceData
}

func NewDriver(config *Config) (*driver, error) {
	klog.Infof("driver.go: NewDriver called")
	var d *driver
	client := nasclient.New(config.nascrd, config.exampleclient.NasV1alpha1())
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err := client.GetOrCreate()
		if err != nil {
			return err
		}

		err = client.UpdateStatus(nascrd.NodeAllocationStateStatusNotReady)
		if err != nil {
			return err
		}

		state, err := NewResourceState(config)
		if err != nil {
			return err
		}

		updatedSpec, err := state.GetUpdatedSpec(&config.nascrd.Spec)
		if err != nil {
			return fmt.Errorf("error getting updated CR spec: %v", err)
		}

		err = client.Update(updatedSpec)
		if err != nil {
			return err
		}

		err = client.UpdateStatus(nascrd.NodeAllocationStateStatusReady)
		if err != nil {
			return err
		}

		if err := os.Remove(sockAddr); err != nil && !os.IsNotExist(err) {
			klog.Errorf("failed to remove stalled socket file", err)
		}

		if _, err := os.Stat(sockAddr); !os.IsNotExist(err) {
			err := os.MkdirAll(filepath.Dir(sockAddr), 0750)
			if err != nil {
				return err
			}
			_, err = os.Create(sockAddr)
			if err != nil {
				return err
			}
		}

		pluginServer := grpc.NewServer()

		driverResourceInfo := &driverCPUResourceData{
			info: make(map[string]resourceInfo),
		}

		d = &driver{
			nascrd:    config.nascrd,
			nasclient: client,
			state:     state,
			server:    pluginServer,
			cpuData:   driverResourceInfo,
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return d, nil
}

func (d *driver) Shutdown() error {
	klog.Infof("driver.go: Shutdown called")
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err := d.nasclient.Get()
		if err != nil {
			return err
		}
		return d.nasclient.UpdateStatus(nascrd.NodeAllocationStateStatusNotReady)
	})
}

func (d *driver) NodePrepareResource(ctx context.Context, req *drapbv1.NodePrepareResourceRequest) (*drapbv1.NodePrepareResourceResponse, error) {
	klog.Infof("NodePrepareResource is called: request: %+v", req)

	var err error
	var prepared []string
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		prepared, err = d.Prepare(req.ClaimUid)
		if err != nil {
			return fmt.Errorf("error allocating devices for claim '%v': %v", req.ClaimUid, err)
		}

		updatedSpec, err := d.state.GetUpdatedSpec(&d.nascrd.Spec)
		if err != nil {
			return fmt.Errorf("error getting updated CR spec: %v", err)
		}

		err = d.nasclient.Update(updatedSpec)
		if err != nil {
			if err := d.state.Unprepare(req.ClaimUid); err != nil {
				klog.Errorf("Failed to unprepare after claim '%v' Update() error: %v", req.ClaimUid, err)
			}
			return err
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error preparing resource: %v", err)
	}

	klog.Infof("Prepared devices for claim '%v': %s", req.ClaimUid, prepared)
	return &drapbv1.NodePrepareResourceResponse{CdiDevices: prepared}, nil
}

func (d *driver) NodeUnprepareResource(ctx context.Context, req *drapbv1.NodeUnprepareResourceRequest) (*drapbv1.NodeUnprepareResourceResponse, error) {
	klog.Infof("NodeUnprepareResource is called: request: %+v", req)

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err := d.Unprepare(req.ClaimUid)
		if err != nil {
			return fmt.Errorf("error unpreparing devices for claim '%v': %v", req.ClaimUid, err)
		}

		updatedSpec, err := d.state.GetUpdatedSpec(&d.nascrd.Spec)
		if err != nil {
			return fmt.Errorf("error getting updated CR spec: %v", err)
		}

		err = d.nasclient.Update(updatedSpec)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error unpreparing resource: %v", err)
	}

	klog.Infof("Unprepared devices for claim '%v'", req.ClaimUid)
	return &drapbv1.NodeUnprepareResourceResponse{}, nil
}

func (d *driver) Prepare(claimUID string) ([]string, error) {
	klog.Infof("driver.go: Prepare called claimUID %+v", claimUID)
	err := d.nasclient.Get()
	if err != nil {
		return nil, err
	}
	prepared, err := d.state.Prepare(claimUID, d.nascrd.Spec.AllocatedClaims[claimUID])
	if err != nil {
		return nil, err
	}
	return prepared, nil
}

func (d *driver) Unprepare(claimUID string) error {
	klog.Infof("driver.go: Unprepare called claimUID %+v", claimUID)
	err := d.nasclient.Get()
	if err != nil {
		return err
	}
	err = d.state.Unprepare(claimUID)
	if err != nil {
		return err
	}
	return nil
}

func (d *driver) UpdateCPUSet(c context.Context, r *cpusetupdaterpb.CpusetRequest) (*cpusetupdaterpb.CpusetResponse, error) {
	klog.Info("UpdateCPUSet Server: UpdateCPUSet called")

	klog.Infof("r.Version: %v", r.Version)
	klog.Infof("Add another CPU id %qto the list", 4)
	klog.Infof("r.ResourceName: %v", r.ResourceName)

	klog.Infof("r.PodName: %v", r.PodName)
	klog.Infof("r.PodNamespace: %v", r.PodNamespace)
	klog.Infof("r.ContainerName: %v", r.ContainerName)

	klog.Infof("r: %v", r.GetNodeName())

	cpus, err := cpuset.Parse(r.Version + ",4")
	if err != nil {
		return nil, err
	}

	klog.Infof("cpus.String(): %v", cpus.String())
	resp := &cpusetupdaterpb.CpusetResponse{
		Cpuset: cpus.String(),
	}

	ctrInfo := make(map[string]string)
	var resInfo resourceInfo
	var ok bool
	if resInfo, ok = d.cpuData.info[r.PodNamespace]; !ok {
		ctrInfo[r.ContainerName] = cpus.String()
		resInfo = resourceInfo{
			podName:       r.PodName,
			containerInfo: ctrInfo,
		}
	}

	driverInfo := make(map[string]resourceInfo)
	driverInfo[r.PodNamespace] = resInfo
	d.cpuData = &driverCPUResourceData{
		info: driverInfo,
	}

	klog.Infof("Server sending resp: %v", resp)
	return resp, nil
}

func (d *driver) Run(sock net.Listener) {
	klog.Info("driver Run:")

	cpusetupdaterpb.RegisterAllocateServer(d.server, d)

	d.server.Serve(sock)

	klog.InfoS("Starting to serve on socket", "socket", sockAddr)
}
