package billing_hours

import (
	"context"
	"time"

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/deployment"
	"github.com/ReconfigureIO/platform/service/stripe"
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
			err = terminateUserDeployments(user, deployments, deploy)
			if err != nil {
				log.Printf("Error while terminating deployments of user: %s", user.ID)
				log.Printf("Error: %s", err)
			}
		}
	}
	return nil
}

func terminateUserDeployments(user models.User, deploymentsDB models.DeploymentRepo, deploy deployment.Service) error {
	deployments, err := deploymentsDB.GetWithStatusForUser(user.ID, []string{"started"})
	if err != nil {
		return err
	}
	for _, deployment := range deployments {
		err = deploy.StopDeployment(context.Background(), deployment)
		if err != nil {
			log.Printf("Error while terminating deployment: %+v", deployment)
		}
	}
	return nil
}

func UpdateDebits(ds models.UserBalanceRepo, deployments models.DeploymentRepo, now time.Time) error {
	// Get all the active users
	users, err := ds.ActiveUsers()
	if err != nil {
		return err
	}
	midnight := now.Truncate(24 * time.Hour)
	midnightYesterday := midnight.AddDate(0, 0, -1)

	// For each active user:
	for _, user := range users {
		// Get their recent invoices
		invoices := stripe.GetUserInvoices(midnightYesterday, midnight, user)
		if len(invoices) >= 1 {
			subscriptionInfo, err := ds.CurrentSubscription(user)
			if err != nil {
				log.Printf("Error while retrieving subscription info for user %s: %s", user.ID, err)
			}
			for _, invoice := range invoices {
				// get the used deployment hours during the invoice period
				usedHours, err := models.DeploymentHoursBtw(deployments, user.ID, time.Unix(invoice.Start, 0), time.Unix(invoice.End, 0))
				if err != nil {
					log.Printf("Error while retrieving deployment hours for user %s: %s", user.ID, err)
				}
				// has the user used credits this month?
				if usedHours > subscriptionInfo.Hours {
					debit := usedHours - subscriptionInfo.Hours
					err = ds.AddDebit(user, debit, invoice.ID)
					if err != nil {
						log.Printf("Error while adding %s hours debit to user: %s", debit, user.ID)
						log.Printf("Error: %s", err)
					}
				}
			}
		}
	}
	return nil

}
