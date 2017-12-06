package stripe

//go:generate mockgen -source=stripe.go -package=stripe -destination=stripe_mock.go

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ReconfigureIO/platform/models"
	stripe "github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/charge"
	"github.com/stripe/stripe-go/client"
	"github.com/stripe/stripe-go/invoice"
)

// Service is a Stripe service.
type Service interface {
	GetUserInvoices(time.Time, time.Time, models.User) []stripe.Invoice
	ChargeUser(int, string, models.User) (*stripe.Charge, error)
	CreateCustomer(string, models.User) (*stripe.Customer, error)
	CreateSubscription(string, models.User) (models.SubscriptionInfo, error)
	CurrentSubscription(models.User) (models.SubscriptionInfo, error)
	GetCustomer(models.User) (*stripe.Customer, error)
	Conf() *ServiceConfig
}

type service struct {
	conf   ServiceConfig
	client *client.API
}

// ServiceConfig holds configuration for service.
type ServiceConfig struct {
	StripeKey string `env:"STRIPE_KEY"`
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
		cust, err = s.client.Customers.New(customerParams)
	} else {
		cust, err = s.client.Customers.Update(user.StripeToken, customerParams)
	}
	return cust, err
}

func (s *service) CreateSubscription(plan string, user models.User) (models.SubscriptionInfo, error) {
	subInfo, err := s.CurrentSubscription(user)
	if err != nil {
		return subInfo, err
	}

	var newSub *stripe.Sub
	if subInfo.StripeID == "" {
		newSub, err = s.client.Subs.New(&stripe.SubParams{
			Customer: user.StripeToken,
			Plan:     plan,
		})
	} else {
		newSub, err = s.client.Subs.Update(
			subInfo.StripeID,
			&stripe.SubParams{
				Plan:      plan,
				NoProrate: true,
			},
		)
	}
	return formatSubInfo(user, *newSub)
}

func (s *service) CurrentSubscription(user models.User) (models.SubscriptionInfo, error) {
	cust, err := s.GetCustomer(user)
	if err != nil {
		return models.SubscriptionInfo{}, err
	}
	subInfo, err := formatSubInfo(user, *cust.Subs.Values[0])
	return subInfo, err
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

func (s *service) GetCustomer(user models.User) (*stripe.Customer, error) {
	stripeCustomer, err := s.client.Customers.Get(user.StripeToken, nil)
	return stripeCustomer, err
}

func formatSubInfo(user models.User, stripeSub stripe.Sub) (models.SubscriptionInfo, error) {
	sub := models.SubscriptionInfo{}
	hours, err := strconv.Atoi(stripeSub.Plan.Meta["HOURS"])
	if err != nil {
		return sub, err
	}
	sub = models.SubscriptionInfo{
		UserID:     user.ID,
		StartTime:  time.Unix(stripeSub.PeriodStart, 0),
		EndTime:    time.Unix(stripeSub.PeriodEnd, 0),
		Hours:      hours,
		StripeID:   stripeSub.ID,
		Identifier: stripeSub.Plan.ID,
	}
	return sub, nil
}
