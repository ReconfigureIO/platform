// +build integration

package stripe

import (
	"testing"

	"github.com/ReconfigureIO/platform/models"
	"github.com/caarlos0/env"
	subscriptions "github.com/stripe/stripe-go/sub"
)

func TestCreateCustomer(t *testing.T) {
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
}

func TestGetCustomer(t *testing.T) {
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

	cust, err = stripeService.GetCustomer(u)
	if err != nil {
		t.Fatal(err)
	}
	if cust.ID != u.StripeToken {
		t.Fatal(err)
	}
}

func TestCreateSubscription(t *testing.T) {
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

	subInfo, err := stripeService.CreateSubscription(models.PlanOpenSource, u)
	if err != nil {
		t.Fatal(err)
	}
	if subInfo.UserID != u.ID || subInfo.Hours != 20 {
		t.Fatal("Created subscription did not match expectations")
	}
}

func TestCurrentSubscription(t *testing.T) {
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

	subInfo, err := stripeService.CreateSubscription(models.PlanOpenSource, u)
	if err != nil {
		t.Fatal(err)
	}
	if subInfo.UserID != u.ID || subInfo.Hours != 20 {
		t.Fatal("Created subscription did not match expectations")
	}

	subInfo, err = stripeService.CurrentSubscription(u)
	if err != nil {
		t.Fatal(err)
	}
	if subInfo.UserID != u.ID || subInfo.Hours != 20 {
		t.Fatal("Created subscription did not match expectations")
	}

}

func TestCurrentSubscriptionMissingSubscription(t *testing.T) {
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

	// Cancel all subscriptions
	for _, val := range cust.Subs.Values {
		_, err = subscriptions.Cancel(val.ID, nil)
		if err != nil {
			t.Fatal(err)
		}
	}

	subInfo, err := stripeService.CurrentSubscription(u)
	if err != nil {
		t.Fatal(err)
	}
	if subInfo.UserID != "" {
		t.Fatal("subscription is not null")
	}
}
