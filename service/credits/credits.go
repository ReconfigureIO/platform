package credits

import (
	"log"
	"time"

	"github.com/ReconfigureIO/platform/models"
	stripe "github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/charge"
)

const (
	// Price of one hour of deployment time in US cents
	hourPrice = 250
)

func UpdateDebits(ds models.UserBalanceRepo, deployments models.DeploymentRepo) error {
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

		//if we're at the end of the billing period
		if subscriptionInfo.EndTime.After(time.Now()) {
			// Get the user's used hours for this billing period
			usedHours, err := models.DeploymentHoursBtw(deployments, user.ID, subscriptionInfo.StartTime, subscriptionInfo.EndTime)
			if err != nil {
				log.Printf("Error while retrieving deployment hours used by user: %s", user.ID)
				log.Printf("Error: %s", err)
			}

			//has the user used credits this month?
			if usedHours > subscriptionInfo.Hours {
				debit := usedHours - subscriptionInfo.Hours
				err = ds.AddDebit(user, debit)
				if err != nil {
					log.Printf("Error while adding %s hours debit to user: %s", debit, user.ID)
					log.Printf("Error: %s", err)
				}
			}
		}
	}
	return nil

}

func AddCredits(desiredCredits int, ds models.UserBalanceRepo, user models.User) error {
	totalCharge := hourPrice * desiredCredits

	chargeParams := &stripe.ChargeParams{
		Amount:   uint64(totalCharge),
		Currency: "usd",
		Desc:     "Charge for deployment time with Reconfigure.io",
		Customer: user.StripeToken,
	}
	_, err := charge.New(chargeParams)
	if err != nil {
		return err
	}

	err = ds.AddCredit(user, desiredCredits)
	if err != nil {
		return err
		// TODO revoke the charge if we fail here
	}
	return nil
}
