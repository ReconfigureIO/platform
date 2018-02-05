package migration

import (
	"errors"
	"fmt"
	"os"

	"github.com/ReconfigureIO/platform/service/leads"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	log "github.com/sirupsen/logrus"
	"gopkg.in/gormigrate.v1"
)

var migrations = []*gormigrate.Migration{
	{
		ID: "1",
		Migrate: func(tx *gorm.DB) error {
			err := tx.AutoMigrate(&InviteToken{}).Error
			if err != nil {
				return err
			}
			err = tx.AutoMigrate(&User{}).Error
			if err != nil {
				return err
			}
			err = tx.AutoMigrate(&Project{}).Error
			if err != nil {
				return err
			}
			err = tx.AutoMigrate(&Simulation{}).Error
			if err != nil {
				return err
			}
			err = tx.AutoMigrate(&Build{}).Error
			if err != nil {
				return err
			}
			err = tx.AutoMigrate(&BatchJob{}).Error
			if err != nil {
				return err
			}
			err = tx.AutoMigrate(&BatchJobEvent{}).Error
			if err != nil {
				return err
			}
			err = tx.AutoMigrate(&Deployment{}).Error
			if err != nil {
				return err
			}
			err = tx.AutoMigrate(&DeploymentEvent{}).Error
			if err != nil {
				return err
			}
			err = tx.AutoMigrate(&BuildReport{}).Error
			if err != nil {
				return err
			}
			err = tx.AutoMigrate(&Graph{}).Error
			if err != nil {
				return err
			}
			err = tx.AutoMigrate(&QueueEntry{}).Error
			return err
		},
		Rollback: func(tx *gorm.DB) error {
			log.Printf("Could not initialise tables")
			return nil
		},
	},
	{
		ID: "201711131234",
		Migrate: func(tx *gorm.DB) error {
			err := tx.Exec(sqlFillDeploymentUserID).Error
			if err != nil {
				return err
			}
			err = tx.Exec(sqlSetDeploymentUserIDNotNull).Error
			return err
		},
		Rollback: func(tx *gorm.DB) error {
			log.Printf("deployment.user_id migration rollback triggered")
			return nil
		},
	},
	{
		ID: "201801231441",
		Migrate: func(tx *gorm.DB) error {
			type User struct {
				Landing         string `json:"-"`
				MainGoal        string `json:"-"`
				Employees       string `json:"-"`
				MarketVerticals string `json:"-"`
				JobTitle        string `json:"-"`
			}
			err := tx.AutoMigrate(&User{}).Error
			return err
		},
		Rollback: func(tx *gorm.DB) error {
			return errors.New("Migration failed. Hit rollback conditions while adding marketing fields to users")
		},
	},
	{
		ID: "201801260952",
		Migrate: func(tx *gorm.DB) error {
			var userIDs []string
			tx.Select("id").Find(&User{})
			for _, id := range userIDs {
				user, err := userData.ImportIntercomData(id)
				if err != nil {
					log.WithError(err).WithFields(log.Fields{
						"user_id": id,
					}).Printf("Failed to import data from intercom for user")
				}
				err = tx.Model(&User{}).Update(&user)
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			return errors.New("Migration failed. Hit rollback conditions while importing marketing data from intercom")
		},
	},
}

var userData leads.Leads

const (
	sqlFillDeploymentUserID = `
UPDATE deployments
SET
user_id = users.id
FROM builds, projects, users
WHERE deployments.build_id = builds.id AND builds.project_id = projects.id AND projects.user_id = users.id AND deployments.user_id IS NULL
`

	sqlSetDeploymentUserIDNotNull = `
ALTER TABLE deployments
ALTER COLUMN user_id SET NOT NULL
`
)

// MigrateSchema performs database migration.
func MigrateSchema() {
	gormConnDets := os.Getenv("DATABASE_URL")
	db, err := gorm.Open("postgres", gormConnDets)
	if err != nil {
		fmt.Println(err)
		panic("failed to connect database")
	}
	db.LogMode(true)
	intercomKey := os.Getenv("RECO_INTERCOM_ACCESS_TOKEN")
	userData = leads.New(intercomKey, db)
	MigrateAll(db)
}

func MigrateAll(db *gorm.DB) {
	options := gormigrate.Options{
		TableName:      "migrations",
		IDColumnName:   "id",
		IDColumnSize:   255,
		UseTransaction: true,
	}
	m := gormigrate.New(db, &options, migrations)

	if err := m.Migrate(); err != nil {
		log.Fatalf("Could not migrate: %v", err)
	}
	log.Printf("Migration did run successfully")
}
