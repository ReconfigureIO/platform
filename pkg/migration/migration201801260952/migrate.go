package migration201801260952

import (
	"errors"
	"os"

	"github.com/ReconfigureIO/platform/pkg/service/events"
	"github.com/ReconfigureIO/platform/pkg/service/leads"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	log "github.com/sirupsen/logrus"
	"gopkg.in/gormigrate.v1"
)

var Migration = gormigrate.Migration{
	ID: "201801260952",
	Migrate: func(tx *gorm.DB) error {
		intercomConfig := events.IntercomConfig{
			AccessToken: os.Getenv("RECO_INTERCOM_ACCESS_TOKEN"),
		}
		userData := leads.New(intercomConfig, tx)
		log.Printf("beginning to find intercom users")
		repo := NewUserRepo(tx)
		userIDs, err := repo.ListUserIDs()
		if err != nil {
			log.WithError(err).Printf("Failed to build a list of users")
			return err
		}
		for _, id := range userIDs {
			log.Printf("trying to import data for user %s", id)
			user, err := userData.ImportIntercomData(id)
			if err != nil {
				log.WithError(err).WithFields(log.Fields{
					"user_id": id,
				}).Printf("Failed to import data from intercom for user")
			}
			migrationUser := modelsUserToMigrationsUser(user)
			_, err = repo.UpdateUser(migrationUser)
			if err != nil {
				log.WithError(err).WithFields(log.Fields{
					"user_id": id,
				}).Printf("Failed to update user")
			}
		}
		return nil
	},
	Rollback: func(tx *gorm.DB) error {
		return errors.New("Migration failed. Hit rollback conditions while importing marketing data from intercom")
	},
}
