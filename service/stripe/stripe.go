package stripe

//go:generate mockgen -source=stripe.go -package=stripe -destination=stripe_mock.go

import (
	"fmt"
	"time"

	"github.com/ReconfigureIO/platform/models"
	stripe "github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/charge"
	"github.com/stripe/stripe-go/client"
	"github.com/stripe/stripe-go/customer"
	"github.com/stripe/stripe-go/invoice"
)

// Service is a Stripe service.
type Service interface {
	GetUserInvoices(time.Time, time.Time, models.User) []stripe.Invoice
	ChargeUser(int, string, models.User) (*stripe.Charge, error)
	CreateCustomer(string, models.User) (*stripe.Customer, error)
	Conf() *ServiceConfig
}

type service struct {
	conf   ServiceConfig
	client *client.API
}

// ServiceConfig holds configuration for service.
type ServiceConfig struct {
	StripeKey string `env:"RECO_STRIPE_KEY"`
}

// New creates a new service with conf.
func New(conf ServiceConfig) Service {
	s := service{conf: conf}
	s.client = client.New(conf.StripeKey, nil)
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

func (s *service) CreateCustomer(token string, user models.User) (*stripe.Customer, error) {
	customerParams := &stripe.CustomerParams{
		Desc:  fmt.Sprintf("%s (github: %d)", user.Name, user.GithubID),
		Email: user.Email,
	}
	if token != "" {
		customerParams.SetSource(token)
	}

	var cust *stripe.Customer
	var err error
	if user.StripeToken == "" {
		cust, err = customer.New(customerParams)
	} else {
		cust, err = customer.Update(user.StripeToken, customerParams)
	}
	return cust, err
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
