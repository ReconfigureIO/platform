// +build integration

package leads

import (
	//	"context"
	"testing"
	//	"time"

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/events"
	//	"github.com/ReconfigureIO/platform/service/aws"
	//	"github.com/golang/mock/gomock"
	"github.com/jinzhu/gorm"
)

func TestSyncIntercomCustomer(t *testing.T) {
	models.RunTransaction(func(db *gorm.DB) {
		icConfig := events.IntercomConfig{
			AccessToken: "",
		}
		leads := New(icConfig, db)
		user := models.User{
			ID:   "24694b52-b7ea-49d5-8352-fd460fa46a2a",
			Name: "Max Siegieda",
		}

		err := leads.SyncIntercomCustomer(user)
		if err != nil {
			t.Error(err)
			return
		}
	})
}
