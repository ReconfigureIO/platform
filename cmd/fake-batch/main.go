package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/private/protocol/json/jsonutil"
	"github.com/aws/aws-sdk-go/service/batch"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/kr/pretty"
)

func main() {
	os.Setenv("DOCKER_API_VERSION", "1.37")
	dockerClient, err := client.NewEnvClient()
	if err != nil {
		log.Fatalf("Unable to configure docker client: %v", err)
	}

	handler := &handler{
		dockerClient: dockerClient,
	}

	log.Fatal(http.ListenAndServe(":9090", handler))
}

type handler struct {
	dockerClient *client.Client
}

const xAmzTarget = "X-Amz-Target"

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

func (h *handler) SubmitJob(w http.ResponseWriter, r *http.Request) {
	var input batch.SubmitJobInput
	err := jsonutil.UnmarshalJSON(&input, r.Body)
	if err != nil {
		log.Printf("Failed to unmarshal: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	pretty.Print(input)

	createOutput, err := h.dockerClient.ContainerCreate(ctx, &container.Config{
		Image: "ubuntu:latest",
		Cmd:   []string{"echo", "hello", "world"},
		// Image: jobDescription.ImageID,
		// Cmd:   []string{jobDescription.Command},
		Labels: map[string]string{
			"fake-batch": "true",
		},
	}, nil, nil, "")
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

	w.Write(output)
}

func (h *handler) DescribeJobs(w http.ResponseWriter, r *http.Request) {
	var input batch.DescribeJobsInput
	err := jsonutil.UnmarshalJSON(&input, r.Body)
	if err != nil {
		log.Printf("Failed to unmarshal: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	pretty.Print(input)
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
}
