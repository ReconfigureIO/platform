// +build integration

package fakebatch

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/batch"
)

func TestStream(t *testing.T) {
	fakebatchURL := string(os.Getenv("RECO_AWS_ENDPOINT"))
	if fakebatchURL == "" {
		t.Error("No endpoint for fakebatch found in environment")
	}
	// create a container with a job that'll make logs
	session := session.New(&aws.Config{
		Endpoint: &fakebatchURL,
	})
	batchSession := batch.New(session)
	params := &batch.SubmitJobInput{
		JobDefinition: aws.String("fake-batch-job-definition"), // Required, fake-batch-job-definition uses the ubuntu:latest image
		JobName:       aws.String("example"),                   // Required
		JobQueue:      aws.String("build"),                     // Required
		ContainerOverrides: &batch.ContainerOverrides{
			Command: aws.StringSlice([]string{"echo", "foobar"}),
		},
	}
	resp, err := batchSession.SubmitJob(params)
	if err != nil {
		t.Error(err)
	}

	batchService := Service{Endpoint: fakebatchURL}

	// Stream logs from that container
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	logs := batchService.Stream(ctx, *resp.JobId)
	defer logs.Close()
	bytes, err := ioutil.ReadAll(logs)
	if err != nil {
		t.Error(err)
	}
	if string(bytes) != "foobar" {
		t.Errorf("Expected 'foobar', got %v", string(bytes))
	}
}
