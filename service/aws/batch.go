package aws

import (
	"context"
	"log"
	"time"

	"github.com/ReconfigureIO/platform/models"
	"github.com/aws/aws-sdk-go/service/batch"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var (
	incompleteStatuses = []string{
		models.StatusTerminating,
		models.StatusCompleted,
		models.StatusErrored,
	}
)

type BatchJobs interface {
	UpdateBatchJobStatus(context.Context) error
	AddDeploymentEvent(context.Context, models.Deployment, models.DeploymentEvent) error
}

type batchJobs struct {
	Builds models.BuildRepo
	Aws    Service
}

func NewBatchJobs(builds models.BuildRepo, aws Service) BatchJobs {
	b := batchJobs{
		Builds: builds,
		Aws:    aws,
	}
	return &b
}

// Find all deployments that are not terminated, and update them
func (batchJobs *batchJobs) UpdateBatchJobStatus(ctx context.Context) error {
	terminatingJobs, err := batchJobs.Builds.GetWithStatus(incompleteStatuses, 100)
	log.Printf("Looking up %d deployments", len(terminatingdeployments))

	if len(terminatingdeployments) == 0 {
		return nil
	}

	//get the status of the associated EC2 instances
	statuses, err := instances.Deploy.DescribeInstanceStatus(ctx, terminatingdeployments)
	if err != nil {
		return err
	}

	log.Printf("statuses of %v", statuses)

	terminating := 0
	//for each deployment, if instance is terminated, send event
	for _, deployment := range terminatingdeployments {
		status, found := statuses[deployment.InstanceID]

		// if it's not found, it was terminated a long time ago, otherwise update
		if !found || status == ec2.InstanceStateNameTerminated {
			event := models.DeploymentEvent{
				Timestamp: time.Now(),
				Status:    models.StatusTerminated,
				Message:   models.StatusTerminated,
				Code:      0,
			}

			err = instances.AddDeploymentEvent(ctx, deployment, event)
			if err != nil {
				return err
			}
			terminating++
		} else if status != ec2.InstanceStateNameShuttingDown {
			// otherwise, if an instance isn't shutting down, something went wrong.
			// let's ask it to shut down in order to reconcile
			err = instances.Deploy.StopDeployment(ctx, deployment)
			if err != nil {
				return err
			}
		}
	}

// 	log.Printf("terminated %d deployments", terminating)
// 	return nil

// }
