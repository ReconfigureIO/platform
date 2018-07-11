package billing_hours

import (
	"context"
	"time"

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/deployment"
	log "github.com/sirupsen/logrus"
)

// Cancel deployments whenever the user has too many billable hours
func CheckUserHours(ds models.SubscriptionRepo, deployments models.DeploymentRepo, deploy deployment.Service) error {
	// Get all the active users
	users, err := ds.ActiveUsers()
	if err != nil {
		return err
	}

	// For each active user:
	for _, user := range users {
		// Get the user's subscription info for this billing period.
		subscriptionInfo, err := ds.CurrentSubscription(user)
		if err != nil {
			log.WithError(err).WithFields(log.Fields{
				"user": user.ID,
			}).Error("Error while retrieving subscription info for user")
		}

		// Get the user's used hours for this billing period
		usedHours, err := models.DeploymentHoursBtw(deployments, user.ID, subscriptionInfo.StartTime, time.Now())
		if err != nil {
			log.WithError(err).WithFields(log.Fields{
				"user": user.ID,
			}).Error("Error while finding user's consumed deployment hours")
		}

		if usedHours >= subscriptionInfo.Hours {
			log.WithFields(log.Fields{
				"user":                  user.ID,
				"subscription-hours":    subscriptionInfo.Hours,
				"consumed-hours":        usedHours,
				"terminating-instances": true,
			}).Info("User has consumed more hours than their subscription allows, terminating their instances")
			err = terminateUserDeployments(user, deployments, deploy)
			if err != nil {
				log.WithError(err).WithFields(log.Fields{
					"user": user.ID,
				}).Error("Error while terminating deploynments of user")
			}
		} else {
			log.WithFields(log.Fields{
				"user":                  user.ID,
				"subscription-hours":    subscriptionInfo.Hours,
				"consumed-hours":        usedHours,
				"terminating-instances": false,
			}).Info("User has consumed fewer hours than their subscription allows, taking no action")
		}
	}
	return nil
}

func terminateUserDeployments(user models.User, deploymentsDB models.DeploymentRepo, deploy deployment.Service) error {
	deployments, err := deploymentsDB.GetWithStatusForUser(user.ID, []string{models.StatusStarted, models.StatusQueued, models.StatusTerminating})
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"user":     user.ID,
			"function": "Billing_hours: terminateUserDeployments",
		}).Error("Couldn't get deployments for user")
		return err
	}

	log.WithFields(log.Fields{
		"user":                  user.ID,
		"number-of-deployments": len(deployments),
		"function":              "Billing_hours: terminateUserDeployments",
	}).Info("Stopping deployments")

	for _, deployment := range deployments {
		log.WithFields(log.Fields{
			"user":       user.ID,
			"deployment": deployment.ID,
			"function":   "Billing_hours: terminateUserDeployments",
		}).Info("Stopping deployment")
		err = deploy.StopDeployment(context.Background(), deployment)
		if err != nil {
			log.Printf("Error while terminating deployment: %+v", deployment)
		}
	}
	return nil
}
