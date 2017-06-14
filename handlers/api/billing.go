package api

import (
	"fmt"
	"strconv"
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

// stripeSub holds info about a user
// stripe subscription.
type stripeSub struct {
	UserID    string
	StartDate string
	Hours     int
}

// subscriptionInfo returns information about the
// user subscription.
// If the user is without an active subscription, default
// open source subscription info is returned.
func subscriptionInfo(userID string) (sub stripeSub) {
	sub = stripeSub{
		UserID:    userID,
		StartDate: timeToSQLStr(monthStart(time.Now())),
		Hours:     models.DefaultHours,
	}
	var user models.User
	err := db.Model(&models.User{}).Where("id=?", userID).First(&user).Error
	switch {
	case err != nil:
		fmt.Println(err)
		fallthrough
	case user.StripeToken == "":
		return
	}
	stripeCustomer, err := customer.Get(user.StripeToken, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	// this may not be necessary if we are guaranteed the user
	// is always gonna have at most one subscription. In which
	// case, we can just return Values[0].Plan.ID directly.
	for _, val := range stripeCustomer.Subs.Values {
		if val.Status != "active " {
			continue
		}
		// check if one of the subscriptions is a valid subscription.
		switch val.Plan.ID {
		case models.PlanSingleUser, models.PlanOpenSource:
			hours, err := strconv.Atoi(val.Plan.Meta["HOURS"])
			if err != nil {
				fmt.Println(err)
				return
			}
			return stripeSub{
				UserID:    userID,
				StartDate: timeToSQLStr(time.Unix(val.PeriodStart, 0)),
				Hours:     hours,
			}
		}
	}
	return
}

// NetHours return the net instance hours for the user after
// deducting deployment time from available hours.
func NetHours(db *gorm.DB, userID string) (time.Duration, error) {
	sub := subscriptionInfo(userID)
	return netHours(db, sub)
}

func netHours(db *gorm.DB, sub stripeSub) (t time.Duration, err error) {
	var deployments []models.Deployment
	err = db.Model(&models.Deployment{}).
		Joins("left join builds on builds.id = deployments.build_id").
		Joins("left join projects on projects.id = builds.project_id").
		Where("projects.user_id=?", sub.UserID).
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
			Where("timestamp>=?", sub.StartDate).
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
	// substract total deployment time from available hours.
	return time.Hour*time.Duration(sub.Hours) - t, err
}

// monthStart changes t to the beginning of the month.
func monthStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// timeToSQLStr formats t in sql format YYYY-MM-DD HH:MM:SS.
func timeToSQLStr(t time.Time) string {
	return fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d",
		t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
}
