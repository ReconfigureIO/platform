package api

import (
	"fmt"

	"github.com/ReconfigureIO/platform/auth"
	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/sugar"
	"github.com/gin-gonic/gin"
	stripe "github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/customer"
)

// Billing handles requests for billing.
type Billing struct{}

type TokenUpdate struct {
	Token string `json:"token"`
}

// DefaultSource doesn't actually include the card info
func (b Billing) DefaultSource(cust *stripe.Customer) *stripe.Card {
	def := cust.DefaultSource.ID
	for _, source := range cust.Sources.Values {
		if source.ID == def {
			return source.Card
		}
	}
	return nil
}

func (b Billing) Get(c *gin.Context) {
	user := auth.GetUser(c)
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

// Update the customer info for the current user
func (b Billing) Replace(c *gin.Context) {
	post := TokenUpdate{}
	err := c.BindJSON(&post)
	if err != nil {
		return
	}
	user := auth.GetUser(c)

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
