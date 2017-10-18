package stripe

import (
	"time"

	"github.com/ReconfigureIO/platform/models"
	stripe "github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/charge"
	"github.com/stripe/stripe-go/customer"
	"github.com/stripe/stripe-go/invoice"
)

func GetUserInvoices(start time.Time, end time.Time, user models.User) []invoice.Invoice {
	invoiceListParams := &stripe.InvoiceListParams{
		Customer: user.Stripe_token,
		DateRange: &stripe.RangeQueryParams{
			GreaterThan: start.Unix(),
			LesserThan:  end.Unix(),
		},
	}

	invoices := invoice.List(invoiceListParams)
	return invoices
}

func ChargeUser(charge int, description string, user models.User) (charge.Charge, err) {
	chargeParams := &stripe.ChargeParams{
		Amount:   uint64(charge),
		Currency: "usd",
		Desc:     description,
		Customer: user.StripeToken,
	}
	newCharge, err := charge.New(chargeParams)
	return newCharge, err
}
