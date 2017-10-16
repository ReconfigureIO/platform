package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/deployment"
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
)

// DeploymentRunner is queue job runner implementation for deployments.
type DeploymentRunner struct {
	Hostname string
	Service  deployment.Service
	DB       *gorm.DB
}

var _ JobRunner = DeploymentRunner{}

// Run satisifies queue.JobRunner interface.
func (d DeploymentRunner) Run(j Job) {
	depID := j.ID

	deployment := models.Deployment{}
	err := d.DB.First(&deployment, "id = ?", depID).Error
	if err != nil {
		log.Println(err)
		return
	}

	callbackURL := fmt.Sprintf("https://%s/deployments/%s/events?token=%s", d.Hostname, deployment.ID, deployment.Token)

	instanceID, err := d.Service.RunDeployment(context.Background(), deployment, callbackURL)
	if err != nil {
		log.Println(err)
		return
	}

	err = d.DB.Model(&deployment).Update("InstanceID", instanceID).Error

	if err != nil {
		log.Println(err)
		return
	}

	newEvent := models.DeploymentEvent{Timestamp: time.Now(), Status: "QUEUED"}
	err = d.DB.Model(&deployment).Association("Events").Append(newEvent).Error
	if err != nil {
		log.Println(err)
		return
	}

	// wait for deployment
	for {
		var dep models.Deployment
		err := d.DB.First(&dep, "id = ?", depID).Error
		if err != nil {
			log.Println(err)
		}
		if dep.HasFinished() {
			break
		}
		time.Sleep(time.Minute * 1)
	}
}

// Stop satisifies queue.JobRunner interface.
func (d DeploymentRunner) Stop(j Job) {
	depID := j.ID

	deployment := models.Deployment{}
	err := d.DB.First(&deployment, "id = ?", depID).Error
	if err != nil {
		log.Println(err)
	}

	err = d.Service.StopDeployment(context.Background(), deployment)
	if err != nil {
		log.Println(err)
	}
}
