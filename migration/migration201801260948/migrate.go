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
		err := tx.Exec(sqlAddUserMarketingFields).Error
		if err != nil {
			return err
		}
		return nil
	},
	Rollback: func(tx *gorm.DB) error {
		return errors.New("Could not automigrate marketing fields into user table")
	},
}

const (
	sqlAddUserMarketingFields = `
ALTER TABLE users
ADD COLUMN landing text,
ADD COLUMN main_goal text,
ADD COLUMN employees text,
ADD COLUMN market_verticals text,
ADD COLUMN job_title text;
`
)
