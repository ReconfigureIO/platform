package profile

import (
	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/middleware"
	"github.com/ReconfigureIO/platform/sugar"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	stripe "github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/customer"
)

// Profile handles requests for profile get & update
type Profile struct {
	DB *gorm.DB
}

func CustInfo(user models.User) (*stripe.Customer, error) {
	if user.StripeToken == "" {
		return nil, nil
	}
	c, err := customer.Get(user.StripeToken, nil)
	return c, err
}

func (p Profile) Get(c *gin.Context) {
	user := middleware.GetUser(c)

	cust, err := CustInfo(user)
	if err != nil {
		sugar.InternalError(c, err)
		return
	}
	
	prof := ProfileData{}
	prof.FromUser(user, cust)

	sugar.SuccessResponse(c, 200, prof)
}

func (p Profile) Update(c *gin.Context) {
	user := middleware.GetUser(c)

	cust, err := CustInfo(user)
	if err != nil {
		sugar.InternalError(c, err)
		return
	}

	prof := ProfileData{}
	prof.FromUser(user, cust)

	c.BindJSON(&prof)

	if !sugar.ValidateRequest(c, prof) {
		return
	}

	prof.Apply(&user, cust)

	err = p.DB.Save(&user).Error

	if err != nil {
		sugar.NotFoundOrError(c, err)
		return
	}

	prof.FromUser(user, cust)
	sugar.SuccessResponse(c, 200, prof)
}
