package profile

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/ReconfigureIO/platform/models"
	stripe "github.com/stripe/stripe-go"
	validator "gopkg.in/validator.v2"
)

func isBillingPlan(v interface{}, param string) error {
	st := reflect.ValueOf(v)
	if st.Kind() != reflect.String {
		return errors.New("isBillingPlan only validates strings")
	}
	if st.String() == models.PlanOpenSource {
		return nil
	}
	if st.String() == models.PlanSingleUser {
		return nil
	}
	return errors.New(fmt.Sprintf("value must be one of \"%s\" or \"%s\"", models.PlanOpenSource, models.PlanSingleUser))
}

func init() {
	validator.SetValidationFunc("is_billing_plan", isBillingPlan)
}

type ProfileData struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	BillingPlan string `json:"billing_plan" validate:"is_billing_plan"`
	Token       string `json:"token"` // read only
}

func UserSubscription(cust *stripe.Customer) string {
	if cust != nil {
		subs := cust.Subs.Values
		if len(subs) > 0 {
			return subs[0].Plan.ID
		}
	}
	return models.OpenSource
}

func (p *ProfileData) FromUser(user models.User, cust *stripe.Customer) {
	p.Name = user.Name
	p.Email = user.Email
	p.Token = user.LoginToken()
	p.BillingPlan = UserSubscription(cust)
}

func (p *ProfileData) Apply(user *models.User, cust *stripe.Customer) {
	user.Name = p.Name
	user.Email = p.Email
	
	// skip token, because it's read only
}
