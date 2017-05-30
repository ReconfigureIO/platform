package profile

import (
	"testing"

	"github.com/ReconfigureIO/platform/models"
	validator "gopkg.in/validator.v2"
)

func TestProfileCanBeOpenSource(t *testing.T) {
	err := validator.Validate(ProfileData{
		BillingPlan: models.OpenSource,
	})
	if err != nil {
		t.Fail()
	}
}

func TestProfileCanBeSingleUser(t *testing.T) {
	err := validator.Validate(ProfileData{
		BillingPlan: models.SingleUser,
	})
	if err != nil {
		t.Fail()
	}
}

func TestProfileFailsWithNonexistantPlan(t *testing.T) {
	err := validator.Validate(ProfileData{
		BillingPlan: "nope",
	})
	if err == nil {
		t.Fail()
	}
}