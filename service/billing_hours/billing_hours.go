package billing_hours

import (
	"fmt"
	"log"

	"github.com/ReconfigureIO/platform/models"
)

// Cancel deployments whenever the user has too many billable hours
func CheckUserHours(ds models.SubscriptionRepo, deployments models.DeploymentRepo) error {
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
			log.Printf("Error while retrieving subscription info for user: %s", user)
			log.Printf("Error: %s", err)
		}

		// Get the user's used hours for this billing period
		usedHours, err := deployments.HoursUsedSince(user.ID, subscriptionInfo.StartTime)
		if err != nil {
			log.Printf("Error while retrieving deployment hours used by user: %s", user)
			log.Printf("Error: %s", err)
		}

		if usedHours >= subscriptionInfo.Hours {
			// err = deployments.TerminateUserDeployments(user)
			// Check err
		}
	}
}
