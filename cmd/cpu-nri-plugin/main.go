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
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/containers/podman/v4/pkg/env"
	"github.com/sirupsen/logrus"
	"k8s.io/utils/cpuset"
	"sigs.k8s.io/yaml"

	"github.com/containerd/nri/pkg/api"
	"github.com/containerd/nri/pkg/stub"
)

const (
	milliCPUToCPU         = 1000
	CPUPluginEnvVarPrefix = "CPU_"
)

type config struct {
	CfgParam1 string `json:"cfgParam1"`
}

type plugin struct {
	stub stub.Stub
}

var (
	cfg config
	log *logrus.Logger
)

func (p *plugin) Configure(_ context.Context, config, runtime, version string) (stub.EventMask, error) {
	log.Infof("Connected to %s/%s...", runtime, version)

	if config == "" {
		return 0, nil
	}

	err := yaml.Unmarshal([]byte(config), &cfg)
	if err != nil {
		return 0, fmt.Errorf("failed to parse configuration: %w", err)
	}

	log.Infof("Got configuration data %+v...", cfg)

	return 0, nil
}

func (p *plugin) Synchronize(_ context.Context, pods []*api.PodSandbox, containers []*api.Container) ([]*api.ContainerUpdate, error) {
	log.Info("Synchronizing state with the runtime...")
	return nil, nil
}

func (p *plugin) Shutdown(_ context.Context) {
	log.Info("Runtime shutting down...")
}

func (p *plugin) RunPodSandbox(_ context.Context, pod *api.PodSandbox) error {
	log.Infof("Started pod %s/%s...", pod.GetNamespace(), pod.GetName())
	return nil
}

func (p *plugin) StopPodSandbox(_ context.Context, pod *api.PodSandbox) error {
	log.Infof("Stopped pod %s/%s...", pod.GetNamespace(), pod.GetName())
	return nil
}

func (p *plugin) RemovePodSandbox(_ context.Context, pod *api.PodSandbox) error {
	log.Infof("Removed pod %s/%s...", pod.GetNamespace(), pod.GetName())
	return nil
}

func (p *plugin) CreateContainer(_ context.Context, pod *api.PodSandbox, ctr *api.Container) (*api.ContainerAdjustment, []*api.ContainerUpdate, error) {
	log.Infof("Creating container %s/%s/%s...", pod.GetNamespace(), pod.GetName(), ctr.GetName())
	adjustment := &api.ContainerAdjustment{}
	updates := []*api.ContainerUpdate{}
	//
	// This is the container creation request handler. Because the container
	// has not been created yet, this is the lifecycle event which allows you
	// the largest set of changes to the container's configuration, including
	// some of the later immautable parameters. Take a look at the adjustment
	// functions in pkg/api/adjustment.go to see the available controls.
	//
	// In addition to reconfiguring the container being created, you are also
	// allowed to update other existing containers. Take a look at the update
	// functions in pkg/api/update.go to see the available controls.
	//

	if !AreCPUsBackedByDRARequested(ctr, pod) {
		return adjustment, updates, nil
	}

	uniqueName := getCtrUniqueName(pod, ctr)
	cpus := cpuset.New(2, 3)
	log.Infof("append mutual cpus to container %q", uniqueName)
	err := setCPUs(ctr, &cpus, uniqueName)
	if err != nil {
		return adjustment, updates, fmt.Errorf("CreateContainer: setting CPUs failed: %w", err)
	}

	//Adding mutual cpus without increasing cpuQuota,
	//might result with throttling the processes' threads
	//if the threads that are running under the mutual cpus
	//oversteps their boundaries, or the threads that are running
	//under the reserved cpus consumes the cpuQuota (pretty common in dpdk/latency sensitive applications).
	//Since we can't determine the cpuQuota for the mutual cpus
	//and avoid throttling the process is critical, increasing the cpuQuota to the maximum is the best option.
	// quota, err := calculateCFSQuota(ctr)
	// if err != nil {
	// 	return adjustment, updates, fmt.Errorf("failed to calculate CFS quota: %w", err)
	// }

	// parentCfsQuotaPath, err := cgroups.Adapter.GetCFSQuotaPath(pod.GetLinux().GetCgroupParent())
	// if err != nil {
	// 	return adjustment, updates, fmt.Errorf("failed to find parent cfs quota: %w", err)
	// }

	// ctrCfsQuotaPath, err := cgroups.Adapter.GetCrioContainerCFSQuotaPath(pod.GetLinux().GetCgroupParent(), ctr.GetId())
	// if err != nil {
	// 	return adjustment, updates, fmt.Errorf("failed to find parent cfs quota: %w", err)
	// }

	// log.Infof("inject hook to modify container's cgroups %q quota to: %d", ctrCfsQuotaPath, quota)
	// hook := &api.Hook{
	// 	Path: "/bin/bash",
	// 	// Args: []string{
	// 	// 	"/bin/bash",
	// 	// 	"-c",
	// 	// 	fmt.Sprintf("echo %d > %s && echo %d > %s", quota, parentCfsQuotaPath, quota, ctrCfsQuotaPath),
	// 	// },
	// }
	// adjustment.Hooks = &api.Hooks{
	// 	CreateRuntime: []*api.Hook{hook},
	// }

	adjustment.Linux = &api.LinuxContainerAdjustment{
		Resources: ctr.Linux.GetResources(),
	}

	log.Infof("sending adjustment to runtime: %+v", adjustment)

	return adjustment, updates, nil
}

func (p *plugin) PostCreateContainer(_ context.Context, pod *api.PodSandbox, ctr *api.Container) error {
	log.Infof("Created container %s/%s/%s...", pod.GetNamespace(), pod.GetName(), ctr.GetName())
	return nil
}

func (p *plugin) StartContainer(_ context.Context, pod *api.PodSandbox, ctr *api.Container) error {
	log.Infof("Starting container %s/%s/%s...", pod.GetNamespace(), pod.GetName(), ctr.GetName())
	return nil
}

func (p *plugin) PostStartContainer(_ context.Context, pod *api.PodSandbox, ctr *api.Container) error {
	log.Infof("Started container %s/%s/%s...", pod.GetNamespace(), pod.GetName(), ctr.GetName())
	return nil
}

func (p *plugin) UpdateContainer(_ context.Context, pod *api.PodSandbox, ctr *api.Container, r *api.LinuxResources) ([]*api.ContainerUpdate, error) {
	log.Infof("Updating container %s/%s/%s...", pod.GetNamespace(), pod.GetName(), ctr.GetName())

	//
	// This is the container update request handler. You can make changes to
	// the container update before it is applied. Take a look at the functions
	// in pkg/api/update.go to see the available controls.
	//
	// In addition to altering the pending update itself, you are also allowed
	// to update other existing containers.
	//

	updates := []*api.ContainerUpdate{}

	return updates, nil
}

func (p *plugin) PostUpdateContainer(_ context.Context, pod *api.PodSandbox, ctr *api.Container) error {
	log.Infof("Updated container %s/%s/%s...", pod.GetNamespace(), pod.GetName(), ctr.GetName())
	return nil
}

func (p *plugin) StopContainer(_ context.Context, pod *api.PodSandbox, ctr *api.Container) ([]*api.ContainerUpdate, error) {
	log.Infof("Stopped container %s/%s/%s...", pod.GetNamespace(), pod.GetName(), ctr.GetName())

	//
	// This is the container (post-)stop request handler. You can update any
	// of the remaining running containers. Take a look at the functions in
	// pkg/api/update.go to see the available controls.
	//

	return []*api.ContainerUpdate{}, nil
}

func (p *plugin) RemoveContainer(_ context.Context, pod *api.PodSandbox, ctr *api.Container) error {
	log.Infof("Removed container %s/%s/%s...", pod.GetNamespace(), pod.GetName(), ctr.GetName())
	return nil
}

func (p *plugin) onClose() {
	log.Infof("Connection to the runtime lost, exiting...")
	os.Exit(0)
}

func main() {
	var (
		pluginName string
		pluginIdx  string
		err        error
	)

	log = logrus.StandardLogger()
	log.SetFormatter(&logrus.TextFormatter{
		PadLevelText: true,
	})

	flag.StringVar(&pluginName, "name", "", "plugin name to register to NRI")
	flag.StringVar(&pluginIdx, "idx", "", "plugin index to register to NRI")
	flag.Parse()

	p := &plugin{}
	opts := []stub.Option{
		stub.WithOnClose(p.onClose),
	}
	if pluginName != "" {
		opts = append(opts, stub.WithPluginName(pluginName))
	}
	if pluginIdx != "" {
		opts = append(opts, stub.WithPluginIdx(pluginIdx))
	}

	if p.stub, err = stub.New(p, opts...); err != nil {
		log.Fatalf("failed to create plugin stub: %v", err)
	}

	if err = p.stub.Run(context.Background()); err != nil {
		log.Errorf("plugin exited (%v)", err)
		os.Exit(1)
	}
}

func getCtrUniqueName(pod *api.PodSandbox, ctr *api.Container) string {
	return fmt.Sprintf("%s/%s/%s", pod.GetNamespace(), pod.GetName(), ctr.GetName())
}

func setCPUs(ctr *api.Container, cpus *cpuset.CPUSet, uniqueName string) error {
	lspec := ctr.GetLinux()

	ctrCpus := lspec.Resources.Cpu
	ctrCpus.Cpus = (*cpus).String()
	log.Infof("container %q cpus ids after applying cpus %q", uniqueName, ctrCpus.Cpus)
	return nil
}

// Requested checks whether a given container is requesting CPUs backed by DRA plugin
func AreCPUsBackedByDRARequested(ctr *api.Container, pod *api.PodSandbox) bool {
	log.Infof("CPUsBackedByDRARequested called: Pod ID: %q CPU Id: %q", pod.Name, ctr.Name)

	if strings.HasPrefix(pod.Namespace, "cpu-test") {

		log.Infof("Passed namespapace prefix test: Pod ID: %q CPU Id: %q", pod.Name, ctr.Name)
		log.Infof("pod: pod.Linux.PodResources %v", pod.Linux.PodResources)
		log.Infof("pod: pod.GetLinux().Resources.GetUnified() %v", pod.GetLinux().Resources.GetUnified())
		log.Infof("ctr: ctr.Linux.GetResources() %v", ctr.Linux.GetResources())

		envs, err := env.ParseSlice(ctr.Env)
		if err != nil {
			log.Errorf("failed to parse environment variables for container: %q; err: %v", ctr.Name, err)
			return false
		}

		for k, v := range envs {
			log.Infof("container :%q Environment variable: key %q value: %q", ctr.Name, k, v)
			return true
		}

		return true
	}
	return false
}
