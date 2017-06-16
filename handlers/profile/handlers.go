package profile

import (
	"github.com/ReconfigureIO/platform/middleware"
	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/sugar"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

// Profile handles requests for profile get & update
type Profile struct {
	DB *gorm.DB
	// subs models.SubscriptionDataSource I want to do this, but the cache makes it an issue cross request
}

func (p Profile) Get(c *gin.Context) {
	user := middleware.GetUser(c)

	sub, err := models.SubscriptionDataSource(p.DB).Current(user)
	if err != nil {
		sugar.InternalError(c, err)
		return
	}

	prof := ProfileData{}
	prof.FromUser(user, sub)

	sugar.SuccessResponse(c, 200, prof)
}

func (p Profile) Update(c *gin.Context) {
	user := middleware.GetUser(c)
	subs := models.SubscriptionDataSource(p.DB)

	sub, err := subs.Current(user)
	if err != nil {
		sugar.InternalError(c, err)
		return
	}

	prof := ProfileData{}
	prof.FromUser(user, sub)

	c.BindJSON(&prof)

	if !sugar.ValidateRequest(c, prof) {
		return
	}

	prof.Apply(&user)

	sub, err = subs.UpdatePlan(user, prof.BillingPlan)

	if err != nil {
		if _, ok := err.(*models.SubscriptionValidationError); ok {
			sugar.ErrResponse(c, 400, err)
			return
		}
		sugar.InternalError(c, err)
		return
	}

	err = p.DB.Save(&user).Error

	if err != nil {
		sugar.NotFoundOrError(c, err)
		return
	}

	prof.FromUser(user, sub)
	sugar.SuccessResponse(c, 200, prof)
}
