package migration1

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	log "github.com/sirupsen/logrus"
	"gopkg.in/gormigrate.v1"
)

var Migration = gormigrate.Migration{
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
}
