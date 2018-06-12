package main

import (
	"context"
	"io"
	"log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
)

type dockerClient interface {
	ContainerCreate(
		ctx context.Context,
		config *container.Config,
		hostConfig *container.HostConfig,
		networkingConfig *network.NetworkingConfig,
		containerName string,
	) (
		container.ContainerCreateCreatedBody,
		error,
	)

	ContainerStart(
		ctx context.Context,
		containerID string,
		options types.ContainerStartOptions,
	) error

	ContainerWait(
		ctx context.Context,
		containerID string,
		condition container.WaitCondition,
	) (
		<-chan container.ContainerWaitOKBody,
		<-chan error,
	)

	ContainerList(
		ctx context.Context,
		options types.ContainerListOptions,
	) (
		[]types.Container,
		error,
	)

	ContainerLogs(
		ctx context.Context,
		container string,
		options types.ContainerLogsOptions,
	) (
		io.ReadCloser,
		error,
	)
}

type dockerHelper struct {
	client dockerClient
	id     string
}

func (dh dockerHelper) Wait() {
	exited, errored := dh.client.ContainerWait(
		context.Background(),
		dh.id,
		container.WaitConditionNotRunning,
	)

	select {
	case <-exited:
	case <-errored:
	}

	// TODO(pwaller): Grab log, shove in S3.
	// TODO(pwaller): Delete container.
}

func (dh dockerHelper) Start() {
	err := dh.client.ContainerStart(
		context.Background(),
		dh.id,
		types.ContainerStartOptions{},
	)
	if err != nil {
		// TODO(pwaller) IMPORTANT: Can we put this error somewhere it can be found by
		// ContainerList()?
		log.Printf("ContainerStart: %v", err)
	}
}

func (dh dockerHelper) Run() {
	dh.Start()
	dh.Wait()
}
