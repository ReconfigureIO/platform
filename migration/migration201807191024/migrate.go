package migration201807191024

import (
	"errors"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"gopkg.in/gormigrate.v1"
)

var Migration = gormigrate.Migration{
	ID: "201807191024",
	Migrate: func(tx *gorm.DB) error {
		err := tx.Exec(sqlAddBuildComment).Error
		return err
	},
	Rollback: func(tx *gorm.DB) error {
		return errors.New("Migration failed. Hit rollback conditions while adding comment field to builds")
	},
}

const (
	sqlAddBuildComment = `
ALTER TABLE builds
ADD COLUMN comment text;
`
)
