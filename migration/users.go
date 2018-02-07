package migration

import (
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
