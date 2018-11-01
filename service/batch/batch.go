package batch

//go:generate mockgen -source=batch.go -package=batch -destination=batch_mock.go

import (

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/aws"
	"github.com/aws/aws-sdk-go/service/batch"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

// Service is a Batch service.
type Service interface {
	RunBuild(build models.Build, callbackURL string, reportsURL string) (string, error)
	RunGraph(graph models.Graph, callbackURL string) (string, error)
	RunSimulation(inputArtifactURL string, callbackURL string, command string) (string, error)
	RunDeployment(command string) (string, error)

	HaltJob(batchID string) error
	GetJobDetail(id string) (*batch.JobDetail, error)

	NewStream(stream cloudwatchlogs.LogStream) *aws.Stream
	GetJobStream(string) (*cloudwatchlogs.LogStream, error)

	Conf() *aws.ServiceConfig
}
