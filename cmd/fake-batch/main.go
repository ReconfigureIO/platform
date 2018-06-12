package main

import (
	"context"
	"fmt"
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
	}

	log.Fatal(http.ListenAndServe(":9090", handler))
}

type handler struct {
	dockerClient          *client.Client
	defaultImage          string
	onlyJobDefinitionName string
	jobQueueSemaphores    jobQueueSemaphores
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[1:]
	left, path := lpartition(path, "/")

	if left != "v1" {
		msg := fmt.Sprintf("Only v1 endpoint supported: %q", left)
		http.Error(w, msg, http.StatusNotImplemented)
		return
	}

	switch path {
	case "submitjob":
		h.SubmitJob(w, r)

	case "describejobs":
		h.DescribeJobs(w, r)

	case "terminatejob":
		h.TerminateJob(w, r)

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
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	containerConfig := h.submitJobInputToContainerConfig(input)

	createOutput, err := h.dockerClient.ContainerCreate(
		ctx, &containerConfig, nil, nil, "",
	)
	if err != nil {
		log.Printf("ContainerCreate: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	jobID := createOutput.ID

	err = h.dockerClient.ContainerStart(ctx, jobID, types.ContainerStartOptions{})
	if err != nil {
		log.Printf("ContainerStart: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

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

func lpartition(s, sep string) (string, string) {
	pos := strings.Index(s, sep)
	if pos == -1 {
		return s, ""
	}
	return s[:pos], s[pos+1:]
}

func rpartition(s, sep string) (string, string) {
	pos := strings.LastIndex(s, sep)
	if pos == -1 {
		return "", s
	}
	return s[:pos], s[pos+1:]
}

func dockerStatusToBatchStatus(dockerStatus string) string {
	// AWS Batch Statuses:
	//
	//   batch.JobStatusSubmitted == "SUBMITTED"
	//   batch.JobStatusPending == "PENDING"
	//   batch.JobStatusRunnable == "RUNNABLE"
	//   batch.JobStatusStarting == "STARTING"
	//   batch.JobStatusRunning == "RUNNING"
	//   batch.JobStatusSucceeded == "SUCCEEDED"
	//   batch.JobStatusFailed == "FAILED"
	//
	// Docker Statuses:
	//
	//   Up 19 hours
	//   Exited (0) 16 minutes ago
	//   Exited (1) 1 minute ago
	//
	switch {
	case strings.HasPrefix(dockerStatus, "Up "):
		return batch.JobStatusRunning
	case strings.HasPrefix(dockerStatus, "Exited (0)"):
		return batch.JobStatusSucceeded
	case strings.HasPrefix(dockerStatus, "Exited "): // All other status codes
		return batch.JobStatusFailed
	default:
		return fmt.Sprintf("UNKNOWN STATUS: %q", dockerStatus)
	}
}
