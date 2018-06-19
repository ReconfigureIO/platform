package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/private/protocol/json/jsonutil"
	"github.com/aws/aws-sdk-go/service/batch"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/kr/pretty"

	"github.com/ReconfigureIO/platform/service/storage"
	"github.com/ReconfigureIO/platform/service/storage/localfile"
)

func main() {
	os.Setenv("DOCKER_API_VERSION", "1.37") // Hmm.

	dockerClient, err := client.NewEnvClient()
	if err != nil {
		log.Fatalf("Unable to configure docker client: %v", err)
	}

	handler := &handler{
		dockerClient:          dockerClient,
		defaultImage:          "ubuntu:latest",             // TODO(pwaller): Configurability?
		onlyJobDefinitionName: "fake-batch-job-definition", // TODO(pwaller): Support for job defs?

		jobQueueSemaphores: newJobQueueSemaphores(map[Q]int{
			"build": 1,
			"graph": 2,
			"sim":   2,
		}),

		storage: localfile.Service("./logs/"),
	}

	// We just started, but we should deal with the case that docker was running
	// before we got here (e.g, our process crashed and restarted).
	// Fill queue slots with whatever is currently running in docker.
	handler.enqueuePreexistingContainers()

	log.Fatal(http.ListenAndServe(":9090", handler))
}

type handler struct {
	dockerClient          dockerClient
	defaultImage          string
	onlyJobDefinitionName string
	jobQueueSemaphores    jobQueueSemaphores

	storage storage.Service
}

// enqueuePreexistingContainers discovers previously submitted work, ensuring
// that running work takes up slots in the queue, and submitted but not started
// work is eventually started.
func (h *handler) enqueuePreexistingContainers() {
	// Before everything else, do cleanup of old containers. These have exited,
	// we just want to go through our wait procedure for the container (which
	// will also move the logs to long term storage and delete the container).
	for _, c := range h.listStatus("exited") {
		h.jobQueueSemaphores.Enqueue(
			Q(c.Labels["job-queue"]),
			// Wait until done.
			dockerHelper{h.dockerClient, c.ID, h.storage}.Wait,
		)
	}

	// First, anything which is already running needs to take up slots in the
	// queue.
	for _, c := range h.listStatus("running") {
		h.jobQueueSemaphores.Enqueue(
			Q(c.Labels["job-queue"]),
			// Wait until done.
			dockerHelper{h.dockerClient, c.ID, h.storage}.Wait,
		)
	}
	// Second, anything hanging around in the created state has been submitted.
	// Those should be started.
	for _, c := range h.listStatus("created") {
		h.jobQueueSemaphores.Enqueue(
			Q(c.Labels["job-queue"]),
			// Start and then wait.
			dockerHelper{h.dockerClient, c.ID, h.storage}.Run,
		)
	}
}

func (h *handler) listStatus(status string) []types.Container {
	containers, err := h.dockerClient.ContainerList(
		context.Background(),
		types.ContainerListOptions{
			// Select containers started by fake-batch.
			Filters: filters.NewArgs(
				filters.Arg("label", "responsible=fake-batch"),
				filters.Arg("status", status),
			),
		},
	)
	if err != nil {
		// TODO(pwaller): Hm, propagate instead?
		log.Panicln("ContainerList failed. Is docker running?", err)
	}
	return containers
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[1:]
	left, path := lpartition(path, "/")

	if left != "v1" {
		msg := fmt.Sprintf("Only v1 endpoint supported: %q", left)
		http.Error(w, msg, http.StatusNotImplemented)
		return

	}

	left, path = lpartition(path, "/")

	switch left {
	case "submitjob":
		h.SubmitJob(w, r)

	case "describejobs":
		h.DescribeJobs(w, r)

	case "terminatejob":
		h.TerminateJob(w, r)

	case "logs":
		h.Logs(w, r)

	default:
		log.Printf("Unsupported request: %q", path)
		r.Write(os.Stderr)
		msg := fmt.Sprintf("Unsupported path: %q", path)
		http.Error(w, msg, http.StatusNotImplemented)
		return
	}
}

func awsEnvToDockerEnv(in []string, kvs []*batch.KeyValuePair) (out []string) {
	out = in
	for _, kv := range kvs {
		out = append(
			out,
			fmt.Sprintf("%s=%s", *kv.Name, *kv.Value),
		)
	}
	return out
}

func (h *handler) submitJobInputToContainerConfig(
	input batch.SubmitJobInput,
) container.Config {
	var (
		cmd []string
		env []string
	)

	co := input.ContainerOverrides
	if co != nil {
		if co.Command != nil {
			cmd = aws.StringValueSlice(co.Command)
		}
		env = awsEnvToDockerEnv(env, co.Environment)
	}

	return container.Config{
		Image: h.defaultImage,
		Cmd:   cmd,
		Env:   env,
		Labels: map[string]string{
			"responsible":    "fake-batch",
			"job-name":       *input.JobName,
			"job-definition": *input.JobDefinition,
			"job-queue":      *input.JobQueue,
		},
	}
}

func (h *handler) SubmitJob(w http.ResponseWriter, r *http.Request) {
	var input batch.SubmitJobInput
	err := jsonutil.UnmarshalJSON(&input, r.Body)
	if err != nil {
		log.Printf("Failed to unmarshal: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if *input.JobDefinition != h.onlyJobDefinitionName {
		msg := fmt.Sprintf(
			"Bad Request, only %q supported as job definition",
			h.onlyJobDefinitionName,
		)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	containerConfig := h.submitJobInputToContainerConfig(input)

	createOutput, err := h.dockerClient.ContainerCreate(
		ctx, &containerConfig, nil, nil, "",
	)
	if err != nil {
		if client.IsErrNotFound(err) {
			resp, err := h.dockerClient.ImagePull(ctx, containerConfig.Image, types.ImagePullOptions{
				All:           false,
				RegistryAuth:  "",
				PrivilegeFunc: nil,
				Platform:      ""},
			)
			if err != nil {
				log.Printf("ContainerCreate: PullImage: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				if resp.Close() != nil {
					log.Printf("ContainerCreate: PullImage: Close: %v", err)
				}
			}
			io.Copy(ioutil.Discard, resp)
			if resp.Close() != nil {
				log.Printf("ContainerCreate: PullImage: Close: %v", err)
			}
			log.Printf("I just finished pulling an image")
			createOutput, err = h.dockerClient.ContainerCreate(
				ctx, &containerConfig, nil, nil, "",
			)
			if err != nil {
				log.Printf("ContainerCreate: Attempt 2: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		}
	}

	jobID := createOutput.ID

	h.jobQueueSemaphores.Enqueue(
		Q(*input.JobQueue),
		dockerHelper{h.dockerClient, jobID, h.storage}.Run,
	)

	output, err := jsonutil.BuildJSON(
		(&batch.SubmitJobOutput{}).
			SetJobId(jobID).
			SetJobName(*input.JobName),
	)
	if err != nil {
		log.Printf("jsonutil.BuildJSON: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	_, err = w.Write(output)
	if err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

func (h *handler) DescribeJobs(w http.ResponseWriter, r *http.Request) {
	var input batch.DescribeJobsInput
	err := jsonutil.UnmarshalJSON(&input, r.Body)
	if err != nil {
		log.Printf("Failed to unmarshal: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	containers, err := h.dockerClient.ContainerList(
		context.Background(),
		types.ContainerListOptions{
			// Include stopped containers.
			All: true,
			// Select containers started by fake-batch.
			Filters: filters.NewArgs(
				filters.Arg("label", "responsible=fake-batch"),
			),
		},
	)
	if err != nil {
		log.Printf("h.dockerClient.ContainerList: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var output batch.DescribeJobsOutput
	for _, c := range containers {
		output.Jobs = append(
			output.Jobs,
			(&batch.JobDetail{}).
				SetJobId(c.ID).
				SetJobName(c.Labels["job-name"]).
				SetStatus(dockerStatusToBatchStatus(c.Status)),
		)
	}

	outputBytes, err := jsonutil.BuildJSON(output)
	if err != nil {
		log.Printf("jsonutil.BuildJSON: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	_, err = w.Write(outputBytes)
	if err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

func (h *handler) TerminateJob(w http.ResponseWriter, r *http.Request) {
	var input batch.TerminateJobInput
	err := jsonutil.UnmarshalJSON(&input, r.Body)
	if err != nil {
		log.Printf("Failed to unmarshal: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	pretty.Print(input)

	panic("TODO(pwaller): Implement this.")
}

func (h *handler) Logs(w http.ResponseWriter, r *http.Request) {
	jobID := strings.TrimPrefix(r.URL.Path, "/v1/logs/")

	// Check to see if the log is in long term storage, grab it from there if
	// possible.
	if reader, err := h.storage.Download(jobID); err == nil {
		_, err := io.Copy(w, reader)
		if err != nil {
			log.Printf("Logs: io.Copy(w, r): %v", err)
			return
		}
		return
	}

	rc, err := dockerHelper{
		client: h.dockerClient,
		id:     jobID,
	}.Logs(context.Background())
	if err != nil {
		log.Printf("Logs: ContainerLogs: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer func() {
		closeErr := rc.Close()
		if closeErr != nil {
			log.Printf("Logs: rc.Close: %v", closeErr)
		}
	}()

	fw := newFlushWriter(w)
	_, err = io.Copy(fw, rc)
	if err != nil {
		log.Printf("Logs: io.Copy: %v", err)
	}
}

func newFlushWriter(w http.ResponseWriter) io.Writer {
	var (
		fw flushWriter
		ok bool
	)

	fw.w, ok = w.(interface {
		Write([]byte) (int, error)
		Flush()
	})
	if !ok {
		return w // Flushing not available.
	}
	return fw
}
