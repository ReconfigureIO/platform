package deployment

import (
	"context"
	"testing"

	"github.com/ReconfigureIO/platform/models"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/golang/mock/gomock"
)

func TestUpdateInstanceStatusShouldUpdateTerminatedInstances(t *testing.T) {
	ctx := context.Background()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	deployments := []models.Deployment{
		models.Deployment{InstanceID: "foo"},
	}
	statuses := map[string]string{"foo": ec2.InstanceStateNameTerminated}

	deploymentRepo := models.NewMockDeploymentRepo(mockCtrl)
	deploymentService := NewMockService(mockCtrl)

	// We don't care about the limit here
	deploymentRepo.EXPECT().GetWithStatus(incompleteStatuses, gomock.Any()).Return(deployments, nil)
	deploymentService.EXPECT().DescribeInstanceStatus(ctx, deployments).Return(statuses, nil)
	deploymentRepo.EXPECT().AddEvent(deployments[0], gomock.Any()).Return(nil)

	err := NewInstances(deploymentRepo, deploymentService).UpdateInstanceStatus(ctx)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateInstanceStatusSetMissingToTerminated(t *testing.T) {
	ctx := context.Background()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	deployments := []models.Deployment{
		models.Deployment{InstanceID: "foo"},
	}
	// test the not found case
	statuses := map[string]string{}

	deploymentRepo := models.NewMockDeploymentRepo(mockCtrl)
	deploymentService := NewMockService(mockCtrl)

	// We don't care about the limit here
	deploymentRepo.EXPECT().GetWithStatus(incompleteStatuses, gomock.Any()).Return(deployments, nil)
	deploymentService.EXPECT().DescribeInstanceStatus(ctx, deployments).Return(statuses, nil)
	deploymentRepo.EXPECT().AddEvent(deployments[0], gomock.Any()).Return(nil)

	err := NewInstances(deploymentRepo, deploymentService).UpdateInstanceStatus(ctx)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateInstanceStatusTerminateRunning(t *testing.T) {
	ctx := context.Background()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	deployments := []models.Deployment{
		models.Deployment{InstanceID: "foo"},
	}
	// test the not found case
	statuses := map[string]string{"foo": ec2.InstanceStateNameRunning}

	deploymentRepo := models.NewMockDeploymentRepo(mockCtrl)
	deploymentService := NewMockService(mockCtrl)

	// We don't care about the limit here
	deploymentRepo.EXPECT().GetWithStatus(incompleteStatuses, gomock.Any()).Return(deployments, nil)
	deploymentService.EXPECT().DescribeInstanceStatus(ctx, deployments).Return(statuses, nil)
	deploymentService.EXPECT().StopDeployment(ctx, deployments[0]).Return(nil)

	err := NewInstances(deploymentRepo, deploymentService).UpdateInstanceStatus(ctx)
	if err != nil {
		t.Error(err)
	}
}
