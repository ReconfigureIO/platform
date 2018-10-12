package main

import (
	"log"
	"net/http"
	"os"

	"github.com/docker/docker/client"

	"github.com/ReconfigureIO/platform/service/storage/localfile"
)

func main() {
	os.Setenv("DOCKER_API_VERSION", "1.26") // 1.26 is in use on our ECS Vivado images.

	dockerClient, err := client.NewEnvClient()
	if err != nil {
		log.Fatalf("Unable to configure docker client: %v", err)
	}

	// Make a map of JobDefinitions to pass to handler
	jobDefinitions := map[string]JobDefinition{
		"fake-batch-job-definition": JobDefinition{
			Image: "ubuntu:latest",
		},
		"sdaccel-builder-build": JobDefinition{
			Image: "398048034572.dkr.ecr.us-east-1.amazonaws.com/reconfigureio/build-framework/sdaccel-builder:v0.17.5",
			MountPoints: []string{
				"/opt/Xilinx:/opt/Xilinx",
			},
		},
	}

	handler := &handler{
		dockerClient:   dockerClient,
		dockerState:    NewDockerState(),
		jobDefinitions: jobDefinitions,

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
