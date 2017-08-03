// +build integration

package models

import (
	"testing"
	"time"

	"github.com/jinzhu/gorm"
)

func TestUserModelsHook(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		//create a user in the DB
		user := User{}
		err := db.Create(&user).Error
		if err != nil {
			t.Error(err)
			return
		}
		returnedUser := User{}
		err = db.Model(&User{}).Where("id = ?", user.ID).Last(&returnedUser).Error
		if err != nil {
			t.Error(err)
			return
		}
		// Validate that the returned user is the same as the in memory user
		if !(user.CreatedAt.Round(time.Second) == returnedUser.CreatedAt.Round(time.Second)) {
			t.Fatalf("\nExpected: %+v\nGot:      %+v\n", user, returnedUser)
			return
		}
	})
}
