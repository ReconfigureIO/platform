package models

import (
	"github.com/jinzhu/gorm"
)

// UserBalanceRepo handles user balance details.
type UserBalanceRepo interface {
	AvailableCredit(user User) (int, error)
	AddDebit(user User, hours int) error
	ActiveUsers() ([]User, error)
	CurrentSubscription(User) (SubscriptionInfo, error)
	UpdatePlan(User, string) (SubscriptionInfo, error)
}

// UserBalance holds information about a user's subscription, credits and debits.
type UserBalance struct {
	Subscription SubscriptionInfo
	Credits      Credits
	Debits       Debits
}

type Credits struct {
	uuidHook
	ID         string `gorm:"primary_key" json:"id"`
	User       User   `json:"-" gorm:"ForeignKey:UserID"`
	UserID     string `json:"-"`
	StripeID   string `json:"-"`
	Identifier string `json:"id"`
	Hours      int
}

type Debits struct {
	uuidHook
	ID         string `gorm:"primary_key" json:"id"`
	UserID     string `json:"-"`
	StripeID   string `json:"-"`
	Identifier string `json:"id"`
	Hours      int
}

// UserBalanceDataSource returns data source for user balances using db.
func UserBalanceDataSource(db *gorm.DB) UserBalanceRepo {
	return newUserBalanceRepo(db)
}

func newUserBalanceRepo(db *gorm.DB) *userBalanceRepo {
	return &userBalanceRepo{
		db: db,
	}
}

type userBalanceRepo struct {
	db *gorm.DB
}

func (repo *userBalanceRepo) GetUserBalance(user User) (UserBalance, error) {
	db := repo.db
	userCredits := Credits{}
	err := db.Where("UserID = ?", user.ID).First(&userCredits).Error
	if err != nil {
		return UserBalance{}, err
	}
	userDebits := Debits{}
	err = db.Where("UserID = ?", user.ID).First(&userDebits).Error
	if err != nil {
		return UserBalance{}, err
	}
	subscriptionInfo, err := repo.CurrentSubscription(user)
	if err != nil {
		return UserBalance{}, err
	}

	userBalance := UserBalance{
		Subscription: subscriptionInfo,
		Credits:      userCredits,
		Debits:       userDebits,
	}

	return userBalance, nil
}

func (repo *userBalanceRepo) AvailableCredit(user User) (int, error) {
	balance, err := repo.GetUserBalance(user)
	if err != nil {
		return 0, err
	}

	available := balance.Subscription.Hours + (balance.Credits.Hours - balance.Debits.Hours)
	return available, nil
}

func (repo *userBalanceRepo) AddDebit(user User, hours int) error {
	balance, err := repo.GetUserBalance(user)
	if err != nil {
		return err
	}

	balance.Debits.Hours = balance.Debits.Hours + hours
	err = repo.db.Save(&balance.Debits).Error
	if err != nil {
		return err
	}
	return nil
}

func (repo *userBalanceRepo) ActiveUsers() ([]User, error) {
	return SubscriptionDataSource(repo.db).ActiveUsers()
}

func (repo *userBalanceRepo) CurrentSubscription(user User) (SubscriptionInfo, error) {
	return SubscriptionDataSource(repo.db).CurrentSubscription(user)
}

func (repo *userBalanceRepo) UpdatePlan(user User, plan string) (SubscriptionInfo, error) {
	return SubscriptionDataSource(repo.db).UpdatePlan(user, plan)
}
