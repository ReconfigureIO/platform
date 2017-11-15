package migration

import (
	"fmt"
	"log"
	"os"

	"github.com/jinzhu/gorm"
	"gopkg.in/gormigrate.v1"
)

// MigrateSchema performs database migration.
func MigrateSchema() {
	gormConnDets := os.Getenv("DATABASE_URL")
	db, err := gorm.Open("postgres", gormConnDets)
	if err != nil {
		fmt.Println(err)
		panic("failed to connect database")
	}
	MigrateAll(db)
}

const (
	sqlDeploymentStatusForUser = `
UPDATE deployments
SET
user_id = users.id
FROM builds, projects, users
WHERE deployments.build_id = builds.id AND builds.project_id = projects.id AND projects.user_id = users.id
`
)

func MigrateAll(db *gorm.DB) {
	m := gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "201711131228",
			Migrate: func(tx *gorm.DB) error {
				err := tx.AutoMigrate(&Deployment{}).Error
				if err != nil {
					return err
				}
				err = tx.Raw(sqlDeploymentStatusForUser).Error
				if err != nil {
					return err
				}
				type Deployment struct {
					UserID string `gorm:"NOT_NULL"`
				}
				err = tx.AutoMigrate(&Deployment{}).Error
				return err
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Table("deployments").DropColumn("user_id").Error
			},
		},
		{
			ID: "201711131225",
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
				return nil
			},
		},
	})

	if err := m.Migrate(); err != nil {
		log.Fatalf("Could not migrate: %v", err)
	}
	log.Printf("Migration did run successfully")
}
