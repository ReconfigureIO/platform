package deployment

import (
	"context"
	"log"
	"time"

	"github.com/ReconfigureIO/platform/models"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var (
	incompleteStatuses = []string{
		models.StatusTerminating,
		models.StatusCompleted,
		models.StatusErrored,
	}
)

type Instances interface {
	UpdateInstanceStatus(context.Context) error
	AddDeploymentEvent(context.Context, models.Deployment, models.DeploymentEvent) error
}

type instances struct {
	Deployments models.DeploymentRepo
	Deploy      Service
}

func NewInstances(deployments models.DeploymentRepo, deploy Service) Instances {
	i := instances{
		Deployments: deployments,
		Deploy:      deploy,
	}
	return &i
}

// AddEvent adds a DeploymentEvent to the Deployment, Terminating the Deployment Instance if necessary.
func (instances *instances) AddDeploymentEvent(ctx context.Context, dep models.Deployment, event models.DeploymentEvent) error {
	err := instances.Deployments.AddEvent(dep, event)

	if err != nil {
		return err
	}

	if event.Status == "TERMINATING" {
		err = instances.Deploy.StopDeployment(ctx, dep)
	}

	return err
}

// Find all deployments that are not terminated, and update them
func (instances *instances) UpdateInstanceStatus(ctx context.Context) error {
	terminatingdeployments, err := instances.Deployments.GetWithStatus(incompleteStatuses, 100)
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

	log.Printf("terminated %d deployments", terminating)
	return nil

}
