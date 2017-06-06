package profile

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/ReconfigureIO/platform/models"
	validator "gopkg.in/validator.v2"
)

func isBillingPlan(v interface{}, param string) error {
	st := reflect.ValueOf(v)
	if st.Kind() != reflect.String {
		return errors.New("isBillingPlan only validates strings")
	}
	if st.String() == models.OpenSource {
		return nil
	}
	if st.String() == models.SingleUser {
		return nil
	}
	return errors.New(fmt.Sprintf("value must be one of \"%s\" or \"%s\"", models.OpenSource, models.SingleUser))
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

func (p *ProfileData) FromUser(user models.User) {
	p.Name = user.Name
	p.Email = user.Email
	p.BillingPlan = models.OpenSource
	p.Token = user.LoginToken()
}

func (p *ProfileData) Apply(user *models.User) {
	user.Name = p.Name
	user.Email = p.Email
	user.BillingPlan = p.BillingPlan
	// skip token, because it's read only
}
