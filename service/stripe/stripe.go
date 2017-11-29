package stripe

//go:generate mockgen -source=stripe.go -package=stripe -destination=stripe_mock.go

import (
	"time"

	"github.com/ReconfigureIO/platform/models"
	stripe "github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/charge"
	"github.com/stripe/stripe-go/invoice"
)

// Service is a Stripe service.
type Service interface {
	GetUserInvoices(time.Time, time.Time, models.User) []stripe.Invoice
	ChargeUser(int, string, models.User) (*stripe.Charge, error)
	Conf() *ServiceConfig
}

type service struct {
	conf   ServiceConfig
	client *stripe.API
}

//TODO add to the config loader so this service is set up on program start
// ServiceConfig holds configuration for service.
type ServiceConfig struct {
	StripeKey string `env:"RECO_STRIPE_KEY"`
}

// New creates a new service with conf.
func New(conf ServiceConfig) Service {
	s := service{conf: conf}
	s.client = &client.New{conf.StripeKey, nil}
	return &s
}

func (s *service) Conf() *ServiceConfig {
	return &s.conf
}

func (s *service) GetUserInvoices(start time.Time, end time.Time, user models.User) []stripe.Invoice {
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

func (s *service) ChargeUser(amount int, description string, user models.User) (*stripe.Charge, error) {
	chargeParams := &stripe.ChargeParams{
		Amount:   uint64(amount),
		Currency: "usd",
		Desc:     description,
		Customer: user.StripeToken,
	}
	newCharge, err := charge.New(chargeParams)
	return newCharge, err
}
