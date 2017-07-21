package billing_hours

import (
	"testing"

	"github.com/ReconfigureIO/platform/handlers/api"
	"github.com/ReconfigureIO/platform/models"
	"github.com/gin-gonic/gin"
)

type fake_SubscriptionRepo struct{}

type fake_Billing struct{}

// provide a bunch of users who are active
func (repo fake_SubscriptionRepo) ActiveUsers() ([]models.User, error) {
	user := models.User{}
	return []models.User{user}, nil
}

func (billing fake_Billing) FetchBillingHours(userID string) api.BillingHours {
	return billingHours{}
}

func (b billingHours) Net() (int, error) {
	return 30, nil
}

func TestCheckUserHours(t *testing.T) {
	d := fake_SubscriptionRepo{}
	b := fake_Billing{}

	err := CheckUserHours(d, b)
	if err != nil {
		t.Fatalf("Error in TestCheckUserHours function: %s", err)
	}

}

type billingHours struct {
}

func (b billingHours) Available() (int, error) {
	return 40, nil
}

func (b billingHours) Used() (int, error) {
	return 40, nil
}

func (s fake_SubscriptionRepo) Current(user models.User) (sub models.SubscriptionInfo, err error) {

	sub = models.SubscriptionInfo{}
	return sub, nil
}

func (s fake_SubscriptionRepo) UpdatePlan(user models.User, plan string) (sub models.SubscriptionInfo, err error) {
	sub = models.SubscriptionInfo{}
	return sub, nil
}

func (b fake_Billing) Get(c *gin.Context) {}

func (b fake_Billing) Replace(c *gin.Context) {}
