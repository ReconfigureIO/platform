package stripe

import (
	"time"

	"github.com/ReconfigureIO/platform/models"
	stripe "github.com/stripe/stripe-go"
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
