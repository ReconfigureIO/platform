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
	Hostname     string
	Service      deployment.Service
	DB           *gorm.DB
	pollInterval time.Duration
}

var _ JobRunner = DeploymentRunner{}

// Run satisifies queue.JobRunner interface.
func (d DeploymentRunner) Run(j Job) {
	depID := j.ID

	//If deployment is already running, stop and log error
	var dep models.Deployment
	err := d.DB.Preload("Events", func(db *gorm.DB) *gorm.DB {
		return db.Order("timestamp")
	}).First(&dep, "id = ?", depID).Error

	if err != nil {
		log.Println(err)
	}

	if dep.HasStarted() {
		log.WithFields(log.Fields{
			"deployment": depID,
			"status":     dep.Status(),
			"spot":       dep.Spot,
			"instance":   dep.InstanceID,
		}).Error("Trying to start deployment that has already started")
		return
	}

	deployment := models.Deployment{}
	err = d.DB.Preload("Build").First(&deployment, "id = ?", depID).Error
	if err != nil {
		log.Error(err)
		return
	}

	//Can user still afford to run deployment?
	subscriptionDS := models.SubscriptionDataSource(d.DB)
	// Get the user's subscription info for this billing period.
	subscriptionInfo, err := subscriptionDS.CurrentSubscription(j.User)
	if err != nil {
		log.Errorf("Error while retrieving subscription info for user: %s", j.User.ID)
		log.Errorf("Error: %s", err)
		return
	}

	deploymentsDS := models.DeploymentDataSource(d.DB)
	// Get the user's used hours for this billing period
	usedHours, err := models.DeploymentHoursBtw(deploymentsDS, j.User.ID, subscriptionInfo.StartTime, time.Now())
	if err != nil {
		log.Errorf("Error while retrieving deployment hours used by user: %s", j.User.ID)
		log.Errorf("Error: %s", err)
		return
	}

	if usedHours >= subscriptionInfo.Hours {
		log.Errorf("User %s does not have enough hours remaining to deploy %s", j.User.ID, deployment.ID)
		return
	}

	callbackURL := fmt.Sprintf("https://%s/deployments/%s/events?token=%s", d.Hostname, deployment.ID, deployment.Token)

	instanceID, err := d.Service.RunDeployment(context.Background(), deployment, callbackURL)
	if err != nil {
		log.Error(err)
		return
	}

	err = d.DB.Model(&deployment).Update("InstanceID", instanceID).Error
	if err != nil {
		log.Error(err)
		return
	}

	newEvent := models.DeploymentEvent{Timestamp: time.Now(), Status: models.StatusQueued}
	err = d.DB.Model(&deployment).Association("Events").Append(newEvent).Error
	if err != nil {
		log.Error(err)
		return
	}
}

// Stop satisifies queue.JobRunner interface.
func (d DeploymentRunner) Stop(j Job) {
	depID := j.ID

	deployment := models.Deployment{}
	err := d.DB.First(&deployment, "id = ?", depID).Error
	if err != nil {
		log.Error(err)
	}

	err = d.Service.StopDeployment(context.Background(), deployment)
	if err != nil {
		log.Error(err)
	}
}
