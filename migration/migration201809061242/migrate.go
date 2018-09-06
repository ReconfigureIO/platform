package migration201809061242

import (
	"errors"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

var Migration = gormigrate.Migration{
	ID: "201809061242",
	Migrate: func(tx *gorm.DB) error {
		err := tx.AutoMigrate(&SimulationReport{}).Error
		return err
	},
	Rollback: func(tx *gorm.DB) error {
		return errors.New("Migration failed. Hit rollback conditions while adding Simulation Reports table to DB")
	},
}
