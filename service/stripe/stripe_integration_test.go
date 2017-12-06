// +build integration

package stripe

import (
	"testing"

	"github.com/ReconfigureIO/platform/models"
	"github.com/caarlos0/env"
	"github.com/jinzhu/gorm"
)

func TestCreateCustomer(t *testing.T) {
	models.RunTransaction(func(db *gorm.DB) {
		u := models.User{
			ID:         "josh",
			Email:      "josh@joshbohde.com",
			GithubName: "joshbohde",
			// important part
			StripeToken: "cus_AgZQTeZbnY6AE4",
		}
		var config ServiceConfig
		err := env.Parse(&config)
		if err != nil {
			t.Fatal(err)
		}

		stripeService := New(config)

		cust, err := stripeService.CreateCustomer("", u)
		if err != nil {
			t.Fatal(err)
		}
		if cust.ID != u.StripeToken {
			t.Fatal(err)
		}

	})
}
