package credits

import (
	"fmt"
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
			invoiceStart := time.Unix(invoice.Lines.Data.Period.Start, 0)
			invoiceEnd := time.Unix(invoice.Lines.Data.Period.End, 0)
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

// func stripeSync(user models.User, ds models.UserBalanceRepo) error {
// 	cust, err := customer.Get(user.StripeToken, &stripe.CustomerParams{})
// 	if err != nil {
// 		return err
// 	}
// 	userBalance, err := ds.GetUserBalance(user)
// 	if err != nil {
// 		return err
// 	}
// 	// if credits in stripe are higher than we have on record there's a problem
// 	if string(userBalance.Credits.Hours) < cust.Meta["credit_hours"] {
// 		return fmt.Errorf("User %s has a credit mismatch in stripe", user.ID)
// 	}

// 	if string(userBalance.Debits.Hours) < cust.Meta["debit_hours"] {
// 		return fmt.Errorf("User %s has a debit mismatch in stripe", user.ID)
// 	}

// 	params := &stripe.CustomerParams{}
// 	params.AddMeta("debit_hours", string(userBalance.Debits.Hours))
// 	params.AddMeta("credit_hours", string(userBalance.Credits.Hours))

// 	_, err = customer.Update(user.StripeToken, params)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }
