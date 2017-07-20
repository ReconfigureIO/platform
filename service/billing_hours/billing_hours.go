package billing_hours

import (
	//	"log"

	"github.com/ReconfigureIO/platform/handlers/api"
	"github.com/ReconfigureIO/platform/models"
)

// CheckUserHours check running deployments and deduct a minute (cron interval) from
// instance hours of the user.
func CheckUserHours(ds models.SubscriptionRepo, billing api.BillingInterface) error {
	users, err := ds.ActiveUsers()
	if err != nil {
		return err
	}

	for _, user := range users {
		h, err := billing.FetchBillingHours(user.ID).Net()
		if err == nil && h <= 0 {
			// TODO terminate all deployments for user
		}
	}
	return nil
}
