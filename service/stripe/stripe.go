package stripe

import (
	"time"

	"github.com/ReconfigureIO/platform/models"
	stripe "github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/charge"
	"github.com/stripe/stripe-go/invoice"
)

func GetUserInvoices(start time.Time, end time.Time, user models.User) []stripe.Invoice {
	invoiceListParams := &stripe.InvoiceListParams{
		Customer: user.StripeToken,
		DateRange: &stripe.RangeQueryParams{
			GreaterThan: start.Unix(),
			LesserThan:  end.Unix(),
		},
	}

	invoices := []stripe.Invoice{}
	invoiceIter := invoice.List(invoiceListParams)
	for invoiceIter.Next() {
		invoices = append(invoices, invoiceIter.Current().(stripe.Invoice))
	}
	return invoices
}

func ChargeUser(amount int, description string, user models.User) (*stripe.Charge, error) {
	chargeParams := &stripe.ChargeParams{
		Amount:   uint64(amount),
		Currency: "usd",
		Desc:     description,
		Customer: user.StripeToken,
	}
	newCharge, err := charge.New(chargeParams)
	return newCharge, err
}
