package api

import (
	"fmt"
	"time"

	"github.com/ReconfigureIO/platform/middleware"
	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/sugar"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	stripe "github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/customer"
)

// Billing handles requests for billing.
type Billing struct{}

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

// NetHours return the net instance hours for the user after
// deducting deployment time from available hours.
func NetHours(db *gorm.DB, userID string) (time.Duration, error) {
	hours, err := currentMonthHours(db, userID)
	if err != nil {
		return 0, err
	}
	used, err := currentMonthDeployments(db, userID)
	if err != nil {
		return 0, err
	}
	return hours - used, nil
}

// currentMonthHours returns the number of user time for the month.
// `time.Duration` is returned for ease of calculation.
// It can always be rounded up the nearest hour for frontend display.
func currentMonthHours(db *gorm.DB, userID string) (t time.Duration, err error) {
	var hours []models.Hours
	err = db.Model(&models.Hours{}).
		Where("user_id=?", userID).
		Where("year=?", time.Now().Year()).
		Where("month=?", time.Now().Month()).
		Find(&hours).Error
	if err != nil {
		return
	}
	for _, hour := range hours {
		t += hour.Hours
	}
	return
}

// currentMonthDeployments returns the duration of all deployments
// for the user. This is still a WIP.
func currentMonthDeployments(db *gorm.DB, userID string) (t time.Duration, err error) {
	var deployments []models.Deployment
	err = db.Model(&models.Deployment{}).
		Joins("left join builds on builds.id = deployments.build_id").
		Joins("left join projects on projects.id = builds.project_id").
		Where("projects.user_id=?", userID).
		Find(&deployments).Error
	if err != nil {
		return
	}
	for _, deployment := range deployments {
		// TODO this queries the db for each deployment.
		// there should be a better way of lazy loading
		// and filtering outside of database.
		err := db.Model(&models.DeploymentEvent{}).
			Where("DepID=?", deployment.ID).
			Order("timestamp").
			Find(&deployment.Events).Error
		if err != nil {
			// TODO decide if this should stop this calculation
			// or it should be ignored and calculation should continue
			// as it currently is.
			fmt.Println(err)
			continue
		}
		if deployment.HasFinished() {
			stopTime := deployment.Events[len(deployment.Events)-1].Timestamp
			duration := stopTime.Sub(deployment.StartTime())
			t += duration
		} else if deployment.HasStarted() {
			duration := time.Now().Sub(deployment.StartTime())
			t += duration
		}
	}
	return
}
