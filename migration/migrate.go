package migration

import (
	"fmt"
	"log"
	"os"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"gopkg.in/gormigrate.v1"
)

var migrations = []*gormigrate.Migration{
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
			log.Printf("migration rollback triggered")
			return nil
		},
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

// MigrateSchema performs database migration.
func MigrateSchema() {
	gormConnDets := os.Getenv("DATABASE_URL")
	db, err := gorm.Open("postgres", gormConnDets)
	if err != nil {
		fmt.Println(err)
		panic("failed to connect database")
	}
	db.LogMode(true)
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

	m.InitSchema(func(tx *gorm.DB) error {
		err := tx.AutoMigrate(
			&InviteToken{},
			&User{},
			&Project{},
			&Simulation{},
			&Build{},
			&BatchJob{},
			&BatchJobEvent{},
			&Deployment{},
			&DeploymentEvent{},
			&BuildReport{},
			&Graph{},
			&QueueEntry{},
		).Error
		if err != nil {
			log.Print("Could not initialise tables")
			return err
		}
		return nil
	})

	if err := m.Migrate(); err != nil {
		log.Fatalf("Could not migrate: %v", err)
	}
	log.Printf("Migration did run successfully")
}
