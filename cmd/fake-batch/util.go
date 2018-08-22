package main

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/service/batch"
	"github.com/docker/docker/api/types"
)

type flushWriter struct {
	w interface {
		Write([]byte) (int, error)
		Flush()
	}
}

func (fw flushWriter) Write(p []byte) (int, error) {
	n, err := fw.w.Write(p)
	fw.w.Flush()
	return n, err
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
	// Note: we considered using 'State' field of the container struct but this
	// does not contain the exit status, so we use the human readable status
	// instead.
	//
	switch {
	case strings.HasPrefix(dockerStatus, "Creating "):
		return batch.JobStatusRunnable
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

func dockerContainerDetailToBatchContainerDetail(c types.Container) *batch.ContainerDetail {
	return &batch.ContainerDetail{
		LogStreamName: &c.ID,
	}
}
