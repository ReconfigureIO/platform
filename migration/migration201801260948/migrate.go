package migration201801260948

import (
	"errors"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"gopkg.in/gormigrate.v1"
)

var Migration = gormigrate.Migration{
	ID: "201801260948",
	Migrate: func(tx *gorm.DB) error {
		err := tx.AutoMigrate(&User{}).Error
		if err != nil {
			return err
		}
		return nil
	},
	Rollback: func(tx *gorm.DB) error {
		return errors.New("Could not automigrate marketing fields into user table")
	},
}
