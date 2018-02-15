package migration201801260952

import (
	"github.com/ReconfigureIO/platform/models"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	log "github.com/sirupsen/logrus"
)

// Service is an AWS service.
type UserRepo interface {
	ListUserIDs() ([]string, error)
	UpdateUser(User) (User, error)
}

type userRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) UserRepo {
	repo := userRepo{
		db: db,
	}
	return &repo
}

func (r *userRepo) ListUserIDs() ([]string, error) {
	rows, err := r.db.Table("users").Select("id").Rows()
	if err != nil {
		log.WithError(err).Printf("Failed to look up users in DB")
		return []string{}, err
	}
	var userIDs []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		userIDs = append(userIDs, id)
	}
	rows.Close()
	return userIDs, nil
}

func (r *userRepo) UpdateUser(user User) (User, error) {
	err := r.db.Update(&user).Error
	var retUser User
	err = r.db.Where("id = ?", user.ID).First(&retUser).Error
	return retUser, err
}

func modelsUserToMigrationsUser(mUser models.User) User {
	user := User{
		ID:                mUser.ID,
		GithubID:          mUser.GithubID,
		GithubName:        mUser.GithubName,
		Name:              mUser.Name,
		Email:             mUser.Email,
		CreatedAt:         mUser.CreatedAt,
		PhoneNumber:       mUser.PhoneNumber,
		Company:           mUser.Company,
		Landing:           mUser.Landing,
		MainGoal:          mUser.MainGoal,
		Employees:         mUser.Employees,
		MarketVerticals:   mUser.MarketVerticals,
		JobTitle:          mUser.JobTitle,
		GithubAccessToken: mUser.GithubAccessToken,
		Token:             mUser.Token,
		StripeToken:       mUser.StripeToken,
		BillingPlan:       mUser.BillingPlan,
	}
	return user
}
