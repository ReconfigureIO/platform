package profile

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/ReconfigureIO/platform/models"
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
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	PhoneNumber string    `json:"phone_number"`
	Company     string    `json:"company"`
	BillingPlan string    `json:"billing_plan" validate:"is_billing_plan"`
	Token       string    `json:"token"` // read only
	CreatedAt   time.Time `json:"created_at"`
}

func (p *ProfileData) FromUser(user models.User, sub models.SubscriptionInfo) {
	p.ID = user.ID
	p.Name = user.Name
	p.Email = user.Email
	p.PhoneNumber = user.PhoneNumber
	p.Company = user.Company
	p.Token = user.LoginToken()
	p.BillingPlan = sub.Identifier
	p.CreatedAt = user.CreatedAt
}

func (p *ProfileData) Apply(user *models.User) {
	user.Name = p.Name
	user.Email = p.Email
	user.PhoneNumber = p.PhoneNumber
	user.Company = p.Company

	// skip id & token, because they are read only
}
