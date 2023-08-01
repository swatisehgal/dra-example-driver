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
	"net"
	"os"
	"strings"
	"time"

	"github.com/containers/podman/v4/pkg/env"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"sigs.k8s.io/yaml"

	"github.com/containerd/nri/pkg/api"
	"github.com/containerd/nri/pkg/stub"
	cpusetupdaterpb "github.com/kubernetes-sigs/dra-example-driver/pkg/cpusetupdater"
)

type config struct {
	CfgParam1 string `json:"cfgParam1"`
}

type plugin struct {
	stub   stub.Stub
	client cpusetupdaterpb.AllocateClient
}

const (
	sockAddr = "/var/lib/kubelet/draplugin/cpudraplugin.sock"
)

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

func (p *plugin) CreateContainer(ctx context.Context, pod *api.PodSandbox, ctr *api.Container) (*api.ContainerAdjustment, []*api.ContainerUpdate, error) {
	log.Infof("Creating container %s/%s/%s...", pod.GetNamespace(), pod.GetName(), ctr.GetName())

	log.Infof("pod: pod.Linux.PodResources %v", pod.Linux.PodResources)
	log.Infof("ctr: ctr.Linux.GetResources() %v", ctr.Linux.GetResources())

	adjustment := &api.ContainerAdjustment{}
	updates := []*api.ContainerUpdate{}

	if !AreCPUsBackedByDRARequested(ctr, pod) {
		return adjustment, updates, nil
	}

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
	log.Infof("Create request")

	req := &cpusetupdaterpb.CpusetRequest{
		Version:      "2,3",
		NodeName:     os.Getenv("NODE_NAME"),
		ResourceName: "cpu",
	}

	log.Infof("NRI: client: About to call UpdateCPUSet")

	resp, err := p.client.UpdateCPUSet(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	log.Infof("Got UpdateNodeTopology: response%v", resp)

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

	log.Info("Before Stub creation")

	if p.stub, err = stub.New(p, opts...); err != nil {
		log.Fatalf("failed to create plugin stub: %v", err)
	}

	log.Info("Stub created")
	go func() {
		if err = p.stub.Run(context.Background()); err != nil {
			log.Errorf("plugin exited (%v)", err)
			os.Exit(1)
		}
	}()

	log.Info("Preparing for dial up")

	var dialOpts []grpc.DialOption
	dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	dialOpts = append(dialOpts, grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
		return net.Dial("unix", addr)
	}))

	log.Info("Dialling")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)

	conn, err := grpc.DialContext(ctx, sockAddr, dialOpts...)
	if err != nil {
		log.Fatalf("Could not dial gRPC: %s", err)
	}
	defer conn.Close()
	defer cancel()

	log.Info("creating client")

	p.client = cpusetupdaterpb.NewAllocateClient(conn)

	select {}
}

func AreCPUsBackedByDRARequested(ctr *api.Container, pod *api.PodSandbox) bool {
	log.Infof("CPUsBackedByDRARequested called: Pod ID: %q CPU Id: %q", pod.Name, ctr.Name)

	// TODO: This needs to check for either resource claim or an annotation populated by DRA controller
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