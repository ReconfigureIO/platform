package migration201802231224

import (
	"errors"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"gopkg.in/gormigrate.v1"
)

var Migration = gormigrate.Migration{
	ID: "201802231224",
	Migrate: func(tx *gorm.DB) error {
		err := tx.Exec(sqlAddBatchJobsLogName).Error
		return err
	},
	Rollback: func(tx *gorm.DB) error {
		return errors.New("Migration failed. Hit rollback conditions while adding CwLogName field to BatchJobs")
	},
}

const (
	sqlAddBatchJobsLogName = `
ALTER TABLE batch_jobs
ADD COLUMN cw_log_name text;
`
)
