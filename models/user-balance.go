package models

//go:generate mockgen -source=user-balance.go -package=models -destination=user-balance_mock.go

import (
	"github.com/jinzhu/gorm"
)

// UserBalanceRepo handles user balance details.
type UserBalanceRepo interface {
	AvailableCredit(User) (int, error)
	PurchasedCredit(User) (int, error)
	AddDebit(User, int, string) error
	AddCredit(User, int) error
	ActiveUsers() ([]User, error)
	CurrentSubscription(User) (SubscriptionInfo, error)
	UpdatePlan(User, string) (SubscriptionInfo, error)
	GetUserBalance(User) (UserBalance, error)
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
	Identifier string `json:"id"`
	Hours      int
}

type Debit struct {
	uuidHook
	ID        string `gorm:"primary_key" json:"id"`
	User      User   `json:"-" gorm:"ForeignKey:UserID"`
	UserID    string `json:"-"`
	InvoiceID string `json:"invoice_id"`
	Hours     int
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

	subscriptionInfo, err := repo.CurrentSubscription(user)
	if err != nil {
		return UserBalance{}, err
	}

	userCredits := Credits{}
	err = db.Where("user_id = ?", user.ID).First(&userCredits).Error
	if err != nil {
		//if missing add it
		userCredits = Credits{
			User:       user,
			Identifier: subscriptionInfo.Identifier,
			Hours:      0,
		}
		err = db.Create(&userCredits).Error
		if err != nil {
			return UserBalance{}, err
		}
	}
	userDebits := Debits{}
	err = db.Where("user_id = ?", user.ID).First(&userDebits).Error
	if err != nil {
		//if missing add it
		userDebits = Debits{
			User:       user,
			Identifier: subscriptionInfo.Identifier,
			Hours:      0,
		}
		err = db.Create(&userDebits).Error
		if err != nil {
			return UserBalance{}, err
		}
	}

	userBalance := UserBalance{
		Subscription: subscriptionInfo,
		Credits:      userCredits,
		Debits:       userDebits,
	}

	return userBalance, nil
}

// Returns the total number of hours available across subscription and purchased credits
func (repo *userBalanceRepo) AvailableCredit(user User) (int, error) {
	balance, err := repo.GetUserBalance(user)
	if err != nil {
		return 0, err
	}

	available := balance.Subscription.Hours + (balance.Credits.Hours - balance.Debits.Hours)
	return available, nil
}

// Returns the number of purchased credits
func (repo *userBalanceRepo) PurchasedCredit(user User) (int, error) {
	balance, err := repo.GetUserBalance(user)
	if err != nil {
		return 0, err
	}

	return balance.Credits.Hours, nil
}

func (repo *userBalanceRepo) AddCredit(user User, credit int) error {
	balance, err := repo.GetUserBalance(user)
	if err != nil {
		return err
	}
	balance.Credits.Hours = balance.Credits.Hours + credit
	err = repo.db.Save(&balance.Credits).Error
	if err != nil {
		return err
	}
	return nil
}

func (repo *userBalanceRepo) AddDebit(user User, hours int, invoiceID string) error {
	newDebit := Debit{
		User:      user,
		InvoiceID: invoiceID,
		Hours:     hours,
	}
	err := repo.db.Create(&newDebit).Error
	return err
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
