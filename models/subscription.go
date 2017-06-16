package models

import (
	"strconv"
	"time"

	"github.com/jinzhu/gorm"
	stripe "github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/customer"
)

// SubscriptionRepo handles user subscription details.
type SubscriptionRepo interface {
	// Current retrieves the current subscription of the user.
	Current(User) (SubscriptionInfo, error)
	// ActiveUsers returns a list of active users.
	ActiveUsers() ([]User, error)
	// CanUpdatePlan returns an error if the user can't update to a plan.
	CanUpdatePlan(User, string) error
	// UpdatePlan sets the user's plan
	UpdatePlan(User, string) (SubscriptionInfo, error)
}

// SubscriptionInfo holds information about a user subscription.
type SubscriptionInfo struct {
	UserID     string    `json:-`
	Identifier string    `json:id`
	StartTime  time.Time `json:start`
	EndTime    time.Time `json:end`
	Hours      int       `json:hours`
}

// Empty returns if the subscription info is empty.
func (s SubscriptionInfo) Empty() bool {
	return s == (SubscriptionInfo{})
}

// SubscriptionDataSource returns data source for subscriptions using db.
func SubscriptionDataSource(db *gorm.DB) SubscriptionRepo {
	return &subscriptionRepo{
		db:            db,
		customerCache: make(map[string]stripe.Customer),
		cache:         make(map[string]SubscriptionInfo),
	}
}

type subscriptionRepo struct {
	db            *gorm.DB
	customerCache map[string]stripe.Customer
	cache         map[string]SubscriptionInfo
}

func (s subscriptionRepo) ActiveUsers() (u []User, err error) {
	// there is no clear way to determine active users yet.
	// let's return all users for now.
	err = s.db.Model(&User{}).Find(&u).Error
	return
}

func (s *subscriptionRepo) cachedCustomer(user User) (cust *stripe.Customer, err error) {
	if cust, ok := s.customerCache[user.ID]; ok {
		return &cust, err
	}
	if user.StripeToken == "" {
		return nil, nil
	}

	stripeCustomer, err := customer.Get(user.StripeToken, nil)
	s.customerCache[user.ID] = *stripeCustomer
	return stripeCustomer, err
}

func (s *subscriptionRepo) Current(user User) (sub SubscriptionInfo, err error) {
	// cache
	// this is not a worry because the cache is scoped
	// to single instance of subscriptionRepo.
	if sub, ok := s.cache[user.ID]; ok {
		return sub, err
	}
	sub = SubscriptionInfo{
		UserID:     user.ID,
		StartTime:  monthStart(time.Now()),
		EndTime:    monthEnd(time.Now()),
		Hours:      DefaultHours,
		Identifier: PlanOpenSource,
	}

	stripeCustomer, err := s.cachedCustomer(user)
	if err != nil {
		return sub, err
	}

	if stripeCustomer == nil {
		return sub, nil
	}

	// this may not be necessary if we are guaranteed the user
	// is always gonna have at most one subscription. In which
	// case, we can just return Values[0].Plan.ID directly.
	for _, val := range stripeCustomer.Subs.Values {
		if val.Status != "active" {
			continue
		}

		hours, err := strconv.Atoi(val.Plan.Meta["HOURS"])
		if err != nil {
			return sub, err
		}
		sub = SubscriptionInfo{
			UserID:     user.ID,
			StartTime:  time.Unix(val.PeriodStart, 0),
			EndTime:    time.Unix(val.PeriodEnd, 0),
			Hours:      hours,
			Identifier: val.Plan.ID,
		}
		// set value only if information is retrived from stripe.
		s.cache[user.ID] = sub
		return sub, nil
	}
	return
}

func (s *subscriptionRepo) CanUpdatePlan(user User, plan string) (err error) {
	return nil
}

func (s *subscriptionRepo) UpdatePlan(user User, plan string) (sub SubscriptionInfo, err error) {
	return SubscriptionInfo{}, nil
}
