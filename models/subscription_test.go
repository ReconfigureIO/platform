// +build integration

package models

import (
	"testing"

	"github.com/jinzhu/gorm"
	subscriptions "github.com/stripe/stripe-go/sub"
)

func TestShouldNotCreateDuplicateSubscriptions(t *testing.T) {
	u := User{
		ID:         "josh",
		Email:      "josh@joshbohde.com",
		GithubName: "joshbohde",
		// important part
		StripeToken: "cus_AgZQTeZbnY6AE4",
	}

	subs := repo(db)
	c, err := subs.cachedCustomer(u)
	if err != nil {
		t.Fatal(err)
	}

	// Cancel all subscriptions
	for _, val := range c.Subs.Values {
		_, err = subscriptions.Cancel(val.ID, nil)
		if err != nil {
			t.Fatal(err)
		}
	}

	subs = repo(db)

	_, err = subs.UpdatePlan(u, PlanSingleUser)
	if err != nil {
		t.Fatal(err)
	}

	subs = repo(db)

	_, err = subs.UpdatePlan(u, PlanSingleUser)
	if err != nil {
		t.Fatal(err)
	}

	subs = repo(db)
	c, err = subs.cachedCustomer(u)
	if err != nil {
		t.Fatal(err)
	}

	if len(c.Subs.Values) != 1 {
		t.Errorf("Expected 1 Subscription, but got %+v", c.Subs)
	}

}
