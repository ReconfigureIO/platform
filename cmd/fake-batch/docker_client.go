package main

import (
	"bytes"
	"context"
	"io"
	"log"
	"strings"

	"github.com/docker/docker/pkg/stdcopy"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"

	"github.com/ReconfigureIO/platform/service/storage"
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

	ContainerRemove(
		ctx context.Context,
		containerID string,
		options types.ContainerRemoveOptions,
	) error

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

	ImagePull(
		ctx context.Context,
		refStr string,
		options types.ImagePullOptions,
	) (
		io.ReadCloser,
		error,
	)
}

type dockerHelper struct {
	client  dockerClient
	id      string
	storage storage.Service
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

	dh.ArchiveLogAndRemoveContainer(nil)
}

// ArchiveLogAndRemoveContainer grabs logs out of Docker and puts them into long
// term storage. You would think it would be better for this to be two
// functions, but it is important that a container is only deleted if the log
// archival succeeds. If it does not succeed, it is better for the container to
// hang around since it is the only place the logs exist.
func (dh dockerHelper) ArchiveLogAndRemoveContainer(extra io.Reader) {
	// Grab log, shove in S3.
	rc, err := dh.Logs(context.Background())
	if err != nil {
		log.Printf("dockerHelper.Wait: dh.Logs: %v", err)
		return
	}
	defer func() {
		closeErr := rc.Close()
		if closeErr != nil {
			log.Printf("dockerHelper.Wait: rc.Close: %v", err)
		}
	}()

	if extra == nil {
		extra = bytes.NewReader(nil)
	}

	_, err = dh.storage.Upload(dh.id, io.MultiReader(rc, extra))
	if err != nil {
		log.Printf("dockerHelper.Wait: dh.storage.Upload: %v", err)
		return
	}

	// Now that the log is in long term storage, the container can be deleted.
	err = dh.client.ContainerRemove(
		context.Background(),
		dh.id,
		types.ContainerRemoveOptions{},
	)
	if err != nil {
		log.Printf("dockerHelper.Wait: ContainerRemove: %v", err)
	}
}

func (dh dockerHelper) Start() {
	err := dh.client.ContainerStart(
		context.Background(),
		dh.id,
		types.ContainerStartOptions{},
	)
	if err != nil {
		log.Printf("ContainerStart: %v", err)
		dh.ArchiveLogAndRemoveContainer(
			strings.NewReader(err.Error()),
		)
	}
}

func (dh dockerHelper) Run() {
	dh.Start()
	dh.Wait()
}

func (dh dockerHelper) Logs(ctx context.Context) (io.ReadCloser, error) {
	rawLogs, err := dh.client.ContainerLogs(
		ctx,
		dh.id,
		types.ContainerLogsOptions{
			Follow:     true,
			ShowStderr: true,
			ShowStdout: true,
		},
	)
	if err != nil {
		return nil, err
	}
	combinedLogs, w := io.Pipe()
	go stripDockerLogEncapsulation(rawLogs, w)
	return combinedLogs, nil
}

func stripDockerLogEncapsulation(r io.Reader, w io.WriteCloser) {
	defer w.Close()

	_, err := stdcopy.StdCopy(w, w, r)
	if err != nil {
		log.Printf("stripDockerLogEncapsulation: %v", err)
	}
}
