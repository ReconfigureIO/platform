// +build integration

package migration201801260952

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"reflect"
	"testing"
)

func TestUpdateUser(t *testing.T) {
	models.RunTransaction(func(db *gorm.DB) {
		//make a basic user and save to DB
		user := User{
			Name: "foobar",
		}
		db.Create(&user)
		//pull the UUID back

		db.Where("name = ?", user.Name).First(&user)
		//now use the UUID in a fancy user
		fancyUser := User{
			ID:              user.ID,
			Name:            "not foobar",
			Email:           "foo@bar.com",
			PhoneNumber:     "0123456789",
			Company:         "foobar",
			Landing:         "foobar",
			MainGoal:        "foobar",
			Employees:       "foobar",
			MarketVerticals: "foobar",
			JobTitle:        "foobar",
		}
		testRepo := NewUserRepo(db)
		returned, err := testRepo.UpdateUser(fancyUser)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(fancyUser, returned) {
			t.Fail()
		}
	})
}
