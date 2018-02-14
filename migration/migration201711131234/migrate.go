package migration201711131234

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	log "github.com/sirupsen/logrus"
	"gopkg.in/gormigrate.v1"
)

var Migration = gormigrate.Migration{
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
}

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
