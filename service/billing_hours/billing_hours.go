package billing_hours

import (
	"context"
	"log"
	"time"

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/mock_deployment"
)

// Cancel deployments whenever the user has too many billable hours
func CheckUserHours(ds models.SubscriptionRepo, deployments models.DeploymentRepo, mockDeploy mock_deployment.Service) error {
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
			log.Printf("Error while retrieving subscription info for user: %s", user.ID)
			log.Printf("Error: %s", err)
		}

		// Get the user's used hours for this billing period
		usedHours, err := models.DeploymentHoursBtw(deployments, user.ID, subscriptionInfo.StartTime, time.Now())
		if err != nil {
			log.Printf("Error while retrieving deployment hours used by user: %s", user.ID)
			log.Printf("Error: %s", err)
		}

		if usedHours >= subscriptionInfo.Hours {
			err = terminateUserDeployments(user, deployments, mockDeploy)
			if err != nil {
				log.Printf("Error while terminating deployments of user: %s", user.ID)
				log.Printf("Error: %s", err)
			}
		}
	}
	return nil
}

func terminateUserDeployments(user models.User, deploymentsDB models.DeploymentRepo, mockDeploy mock_deployment.Service) error {
	deployments, err := deploymentsDB.GetWithStatusForUser(user.ID, []string{"started"})
	if err != nil {
		return err
	}
	for _, deployment := range deployments {
		err = mockDeploy.StopDeployment(context.Background(), deployment)
		if err != nil {
			log.Printf("Error while terminating deployment: %+v", deployment)
		}
	}
	return nil
}
