package api

import (
	"fmt"
	"time"

	"github.com/ReconfigureIO/platform/middleware"
	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/sugar"
	"github.com/gin-gonic/gin"
	stripe "github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/customer"
)

// Billing handles requests for billing.
type Billing struct{}

// TokenUpdate is token update payload.
type TokenUpdate struct {
	Token string `json:"token"`
}

// DefaultSource doesn't actually include the card info, so search the
// sources on the customer for the card info
func (b Billing) DefaultSource(cust *stripe.Customer) *stripe.Card {
	def := cust.DefaultSource.ID
	for _, source := range cust.Sources.Values {
		if source.ID == def {
			return source.Card
		}
	}
	return nil
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
		sugar.InternalError(c, err)
		return
	}
	sugar.SuccessResponse(c, 200, b.DefaultSource(cust))
}

// Replace updates the customer info for the current user, returning the card info
func (b Billing) Replace(c *gin.Context) {
	post := TokenUpdate{}
	err := c.BindJSON(&post)
	if err != nil {
		return
	}
	user := middleware.GetUser(c)

	customerParams := &stripe.CustomerParams{
		Desc:  fmt.Sprintf("%s (github: %d)", user.Name, user.GithubID),
		Email: user.Email,
	}
	customerParams.SetSource(post.Token)

	var cust *stripe.Customer
	if user.StripeToken == "" {
		cust, err = customer.New(customerParams)
	} else {
		cust, err = customer.Update(user.StripeToken, customerParams)
	}

	if err != nil {
		sugar.InternalError(c, err)
		return
	}

	err = db.Model(&user).Updates(models.User{StripeToken: cust.ID}).Error

	if err != nil {
		sugar.InternalError(c, err)
		return

	}
	sugar.SuccessResponse(c, 200, b.DefaultSource(cust))
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
func FetchBillingHours(userID string) BillingHours {
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
	sub, err := b.subRepo.Current(b.user)
	return sub.Hours, err
}

func (b billingHours) Used() (int, error) {
	if b.err != nil {
		return 0, b.err
	}
	sub, err := b.subRepo.Current(b.user)
	if err != nil {
		return 0, err
	}
	used, err := b.depRepo.DeploymentHoursSince(b.user.ID, sub.StartTime)
	return int(used.Hours()), err
}

func (b billingHours) Net() (int, error) {
	if b.err != nil {
		return 0, b.err
	}
	sub, err := b.subRepo.Current(b.user)
	used, err := b.depRepo.DeploymentHoursSince(b.user.ID, sub.StartTime)
	if err != nil {
		return 0, err
	}
	net := time.Duration(sub.Hours)*time.Hour - used
	// round up the hour
	if net%time.Hour > 0 {
		return int(net.Hours()) + 1, nil
	}
	return int(net.Hours()), nil
}
