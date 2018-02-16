// +build integration

package leads

import (
	"os"
	"reflect"
	"testing"

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/events"
	"github.com/jinzhu/gorm"
	intercom "gopkg.in/intercom/intercom-go.v2"
)

func TestSyncIntercomCustomer(t *testing.T) {
	models.RunTransaction(func(db *gorm.DB) {
		icConfig := events.IntercomConfig{
			AccessToken: os.Getenv("RECO_INTERCOM_ACCESS_TOKEN"),
		}
		if icConfig.AccessToken == "" {
			t.Fatal("Intercom access token missing, cannot test intercom integration")
		}
		leads := New(icConfig, db)
		user := models.User{
			ID:              "24694b52-b7ea-49d5-8352-fd460fa46a2a",
			Name:            "Max Siegieda",
			Email:           "max.siegieda@reconfigure.io",
			PhoneNumber:     "0123456789",
			MainGoal:        "to test intercom sync",
			Employees:       "123456789",
			MarketVerticals: "Services",
			Company:         "Reconfigure.io",
		}

		err := leads.SyncIntercomCustomer(user)
		if err != nil {
			t.Error(err)
			return
		}
	})
}

func TestImportIntercomData(t *testing.T) {
	models.RunTransaction(func(db *gorm.DB) {
		icConfig := events.IntercomConfig{
			AccessToken: os.Getenv("RECO_INTERCOM_ACCESS_TOKEN"),
		}
		if icConfig.AccessToken == "" {
			t.Fatal("Intercom access token missing, cannot test intercom integration")
		}
		leads := New(icConfig, db)

		user := models.User{
			ID:              "24694b52-b7ea-49d5-8352-fd460fa46a2a",
			Name:            "Max Siegieda",
			Email:           "max.siegieda@reconfigure.io",
			PhoneNumber:     "0123456789",
			MainGoal:        "to test intercom sync",
			Employees:       "123456789",
			MarketVerticals: "Services",
			Company:         "Reconfigure.io",
			JobTitle:        "Operations Specialist",
		}
		//put user data in intercom
		err := leads.SyncIntercomCustomer(user)
		if err != nil {
			t.Error(err)
			return
		}
		//get user data from intercom
		returned, err := leads.ImportIntercomData(user.ID)
		if err != nil {
			t.Error(err)
			return
		}

		if !reflect.DeepEqual(user, returned) {
			t.Fatalf("\nExpected: %+v\nGot:      %+v\n", user, returned)
			return
		}

	})
}

func TestConvertIcUser(t *testing.T) {
	icUser := intercom.User{}
	_ = convertIcUser(icUser, "foobar")
	return
}
