package credits

import (
	"log"
	"time"

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/stripe"
)

const (
	// Price of one hour of deployment time in US cents
	hourPrice = 250
)

func UpdateDebits(ds models.UserBalanceRepo, deployments models.DeploymentRepo, now time.Time) error {
	// Get all the active users
	users, err := ds.ActiveUsers()
	if err != nil {
		return err
	}

	// For each active user:
	for _, user := range users {
		//find invoices from yesterday
		midnight := now.Truncate(24 * time.Hour)
		previousMidnight := midnight.AddDate(0, 0, -1)
		invoices := stripe.GetUserInvoices(midnight, previousMidnight, user)
		//find ranges on invoice(s)
		for _, invoice := range invoices {
			invoiceStart := time.Unix(invoice.Lines.Values[0].Period.Start, 0)
			invoiceEnd := time.Unix(invoice.Lines.Values[0].Period.End, 0)
			//create debits for invoice period(s)
			subscriptionInfo, err := ds.CurrentSubscription(user)
			if err != nil {
				log.Printf("Error while retrieving user %s 's subscription info: %s", user.ID, err)
			}
			usedHours, err := models.DeploymentHoursBtw(deployments, user.ID, invoiceStart, invoiceEnd)
			if err != nil {
				log.Printf("Error while retrieving user %s 's deployment hours: %s", user.ID, err)
			}

			//has the user used credits this month?
			if usedHours > subscriptionInfo.Hours {
				debit := usedHours - subscriptionInfo.Hours
				err = ds.AddDebit(user, debit, invoice.ID)
				if err != nil {
					log.Printf("Error while adding %s hours debit to user %s : %s", debit, user.ID, err)
				}
			}
		}
	}
	return nil

}

func AddCredits(desiredCredits int, ds models.UserBalanceRepo, user models.User) error {
	totalCharge := hourPrice * desiredCredits
	chargeDescription := "Charge for deployment time with Reconfigure.io"

	_, err := stripe.ChargeUser(totalCharge, chargeDescription, user)
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
