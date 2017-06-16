package models

import (
	"strconv"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/stripe/stripe-go/customer"
)

// SubscriptionRepo handles user subscription details.
type SubscriptionRepo interface {
	// Current retrieves the current subscription of the user.
	Current(User) (SubscriptionInfo, error)
	// ActiveUsers returns a list of active users.
	ActiveUsers() ([]User, error)
}

// SubscriptionInfo holds information about a user subscription.
type SubscriptionInfo struct {
	UserID    string
	StartTime time.Time
	EndTime   time.Time
	Hours     int
}

// Empty returns if the subscription info is empty.
func (s SubscriptionInfo) Empty() bool {
	return s == (SubscriptionInfo{})
}

// SubscriptionDataSource returns data source for subscriptions using db.
func SubscriptionDataSource(db *gorm.DB) SubscriptionRepo {
	return &subscriptionRepo{
		db:    db,
		cache: make(map[string]SubscriptionInfo),
	}
}

type subscriptionRepo struct {
	db    *gorm.DB
	cache map[string]SubscriptionInfo
}

func (s subscriptionRepo) ActiveUsers() (u []User, err error) {
	// there is no clear way to determine active users yet.
	// let's return all users for now.
	err = s.db.Model(&User{}).Find(&u).Error
	return
}

func (s *subscriptionRepo) Current(user User) (sub SubscriptionInfo, err error) {
	// cache
	// this is not a worry because the cache is scoped
	// to single instance of subscriptionRepo.
	if sub, ok := s.cache[user.ID]; ok {
		return sub, err
	}
	sub = SubscriptionInfo{
		UserID:    user.ID,
		StartTime: monthStart(time.Now()),
		Hours:     DefaultHours,
	}
	if user.StripeToken == "" {
		return
	}
	stripeCustomer, err := customer.Get(user.StripeToken, nil)
	if err != nil {
		return sub, err
	}
	// this may not be necessary if we are guaranteed the user
	// is always gonna have at most one subscription. In which
	// case, we can just return Values[0].Plan.ID directly.
	for _, val := range stripeCustomer.Subs.Values {
		if val.Status != "active " {
			continue
		}

		hours, err := strconv.Atoi(val.Plan.Meta["HOURS"])
		if err != nil {
			return sub, err
		}
		sub = SubscriptionInfo{
			UserID:    user.ID,
			StartTime: time.Unix(val.PeriodStart, 0),
			EndTime:   time.Unix(val.PeriodEnd, 0),
			Hours:     hours,
		}
		// set value only if information is retrived from stripe.
		s.cache[user.ID] = sub
		return sub, nil
	}
	return
}
