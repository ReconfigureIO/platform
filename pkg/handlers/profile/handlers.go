package profile

import (
	"github.com/ReconfigureIO/platform/pkg/middleware"
	"github.com/ReconfigureIO/platform/pkg/models"
	"github.com/ReconfigureIO/platform/pkg/sugar"
	"github.com/ReconfigureIO/platform/service/leads"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
)

// Profile handles requests for profile get & update
type Profile struct {
	DB    *gorm.DB
	Leads leads.Leads
	// subs models.SubscriptionDataSource I want to do this, but the cache makes it an issue cross request
}

func (p Profile) Get(c *gin.Context) {
	user := middleware.GetUser(c)

	sub, err := models.SubscriptionDataSource(p.DB).CurrentSubscription(user)
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

	sub, err := subs.CurrentSubscription(user)
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

	err = p.Leads.SyncIntercomCustomer(user)
	if err != nil {
		log.WithError(err).Error("Error while syncing user to intercom")
	}

	prof.FromUser(user, sub)
	sugar.SuccessResponse(c, 200, prof)
}
