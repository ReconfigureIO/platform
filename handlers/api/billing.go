package api

import (
	"fmt"

	"github.com/ReconfigureIO/platform/middleware"
	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/stripe"
	"github.com/ReconfigureIO/platform/sugar"
	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/customer"
)

// Billing handles requests for billing.
type Billing struct {
	Stripe stripe.Service
	Events events.EventService
}

// NewSimulation creates a new Simulation.
func NewSimulation(events events.EventService) Simulation {
	return Simulation{
		Stripe: stripeClient,
		Events: events,
	}
}

type BillingInterface interface {
	Get(c *gin.Context)
	Replace(c *gin.Context)
	FetchBillingHours(userID string) BillingHours
	RemainingHours(c *gin.Context)
}

// TokenUpdate is token update payload.
type TokenUpdate struct {
	Token string `json:"token"`
}

// Get the default card info for the customer for frontend display
func (b Billing) Get(c *gin.Context) {
	user := middleware.GetUser(c)
	if user.StripeToken == "" {
		sugar.ErrResponse(c, 404, nil)
		return
	}
	cust, err := customer.Get(user.StripeToken, nil)
	if err != nil {
		sugar.StripeError(c, err)
		return
	}
	sugar.SuccessResponse(c, 200, models.DefaultSource(cust))
}

// Replace updates the customer info for the current user, returning the card info
func (b Billing) Replace(c *gin.Context) {
	post := TokenUpdate{}
	err := c.BindJSON(&post)
	if err != nil {
		return
	}
	user := middleware.GetUser(c)

	cust, err := b.Stripe.CreateCustomer(post.Token, user)

	if err != nil {
		sugar.StripeError(c, err)
		return
	}

	err = db.Model(&user).Updates(models.User{StripeToken: cust.ID}).Error

	if err != nil {
		sugar.InternalError(c, err)
		return

	}
	sugar.SuccessResponse(c, 200, models.DefaultSource(cust))
}

func (b Billing) RemainingHours(c *gin.Context) {
	user := middleware.GetUser(c)
	billingHours := b.FetchBillingHours(user.ID)
	remaining, err := billingHours.Net()
	if err != nil {
		sugar.InternalError(c, err)
		return
	}
	sugar.SuccessResponse(c, 200, remaining)
}

// BillingHours returns information about billing hours for user.
// Hours are rounded up. i.e. 0 == 0, 1 hour == [1-60]minutes. e.t.c.
type BillingHours interface {
	// Available returns available number of hours.
	Available() (int, error)
	// Used returns total hours used by deployments.
	Used() (int, error)
	// Net returns hours after deducting used hours.
	// i.e. net = available - used.
	Net() (int, error)
}

// FetchBillingHours fetches and return billing hours for a user.
func (b Billing) FetchBillingHours(userID string) BillingHours {
	var user models.User
	err := db.Model(&models.User{}).Where("id = ?", userID).First(&user).Error
	if err != nil {
		return billingHours{err: err}
	}
	return billingHours{
		user:    user,
		depRepo: models.DeploymentDataSource(db),
		subRepo: models.SubscriptionDataSource(db),
	}
}

type billingHours struct {
	user    models.User
	depRepo models.DeploymentRepo
	subRepo models.SubscriptionRepo
	err     error
}

func (b billingHours) Available() (int, error) {
	if b.err != nil {
		return 0, b.err
	}
	sub, err := b.subRepo.CurrentSubscription(b.user)
	return sub.Hours, err
}

func (b billingHours) Used() (int, error) {
	if b.err != nil {
		return 0, b.err
	}
	sub, err := b.subRepo.CurrentSubscription(b.user)
	if err != nil {
		return 0, err
	}
	used, err := models.DeploymentHoursBtw(b.depRepo, b.user.ID, sub.StartTime, sub.EndTime)
	return used, err
}

func (b billingHours) Net() (int, error) {
	//If billingHours is invalid, stop
	if b.err != nil {
		return 0, b.err
	}
	sub, err := b.subRepo.CurrentSubscription(b.user)
	used, err := models.DeploymentHoursBtw(b.depRepo, b.user.ID, sub.StartTime, sub.EndTime)

	if err != nil {
		return 0, err
	}
	net := sub.Hours - used
	return net, nil
}
