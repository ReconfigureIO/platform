package deployment

import (
	"context"
	"time"

	"github.com/ReconfigureIO/platform/models"
	"github.com/aws/aws-sdk-go/service/ec2"
	log "github.com/sirupsen/logrus"
)

var (
	runningStatus = []string{
		models.StatusQueued,
		models.StatusStarted,
		models.StatusTerminating,
		models.StatusCompleted,
		models.StatusErrored,
	}

	incompleteStatuses = []string{
		models.StatusTerminating,
		models.StatusCompleted,
		models.StatusErrored,
	}
)

type Instances interface {
	UpdateInstanceStatus(context.Context) error
	AddDeploymentEvent(context.Context, models.Deployment, models.DeploymentEvent) error
	FindIPs(context.Context) error
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

func inSlice(slice []string, val string) bool {
	for _, v := range slice {
		if val == v {
			return true
		}
	}
	return false
}

// Find all deployments that are not terminated, and update them
func (instances *instances) UpdateInstanceStatus(ctx context.Context) error {
	runningdeployments, err := instances.Deployments.GetWithStatus(runningStatus, 100)

	log.WithFields(log.Fields{
		"count": len(runningdeployments),
	}).Info("Looking up deployments")

	if len(runningdeployments) == 0 {
		return nil
	}

	// get the status of the associated EC2 instances
	statuses, err := instances.Deploy.DescribeInstanceStatus(ctx, runningdeployments)
	if err != nil {
		return err
	}

	terminating := 0

	// for each deployment, if instance is terminated, send event
	for _, deployment := range runningdeployments {
		status, found := statuses[deployment.InstanceID]
		depStatus := deployment.Status()

		// if it's not found, it was terminated a long time ago, otherwise update
		if !found || status == ec2.InstanceStateNameTerminated {
			event := models.DeploymentEvent{
				Timestamp: time.Now(),
				Status:    models.StatusTerminated,
				Message:   "Instance has terminated",
				Code:      0,
			}

			err = instances.AddDeploymentEvent(ctx, deployment, event)
			if err != nil {
				return err
			}
			terminating++
		} else if status != ec2.InstanceStateNameShuttingDown && inSlice(incompleteStatuses, depStatus) {
			// otherwise, if an instance isn't shutting down, something went wrong.
			// let's ask it to shut down in order to reconcile
			err = instances.Deploy.StopDeployment(ctx, deployment)
			if err != nil {
				return err
			}
		}
	}

	log.WithFields(log.Fields{
		"count": terminating,
	}).Info("Terminated deployments")

	return nil

}

// For all deployments that do not have an IPv4 address, find their IPs
func (instances *instances) FindIPs(ctx context.Context) error {
	deploymentsWithoutIPs, err := instances.Deployments.GetWithoutIP()

	log.WithFields(log.Fields{
		"count": len(deploymentsWithoutIPs),
	}).Info("Getting IPs of Deployments")

	if len(deploymentsWithoutIPs) == 0 {
		return nil
	}

	// AWS Describe the associated EC2 instances to get their IPv4 addresses
	instanceIPs, err := instances.Deploy.DescribeInstanceIPs(ctx, deploymentsWithoutIPs)
	if err != nil {
		return err
	}

	updated := 0

	// for each deployment, if we have an IP, set IP
	for _, deployment := range deploymentsWithoutIPs {
		ip, found := instanceIPs[deployment.InstanceID]
		if found {
			err := instances.Deployments.SetIP(deployment, ip)
			if err != nil {
				log.Error(err)
				updated++
			}
		}
	}

	log.WithFields(log.Fields{
		"count": updated,
	}).Info("Found IPs for deployments")

	return nil

}
