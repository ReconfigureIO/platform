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
	uuidHook
	ID           string `gorm:"primary_key" json:"id"`
	UserID       string `json:"-"`
	Subscription SubscriptionInfo
	Credits      Credits
	Debits       Debits
}

type Credits struct {
	uuidHook
	ID         string `gorm:"primary_key" json:"id"`
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

func (repo *userBalanceRepo) AvailableCredit(user User) (int, error) {
	db := repo.db
	userBalance := UserBalance{}
	err := db.Where("ID = ?", user.ID).First(&userBalance).Error
	if err != nil {
		return 0, err
	}

	available := userBalance.Subscription.Hours + (userBalance.Credits.Hours - userBalance.Debits.Hours)
	return available, nil
}

func (repo *userBalanceRepo) AddDebit(user User, hours int) error {
	db := repo.db
	userBalance := UserBalance{}
	err := db.Where("ID = ?", user.ID).First(&userBalance).Error
	if err != nil {
		return err
	}

	userBalance.Debits.Hours = userBalance.Debits.Hours + hours
	err = db.Save(&userBalance).Error
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
