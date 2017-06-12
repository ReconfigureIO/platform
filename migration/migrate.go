package migration

import (
	"fmt"
	"os"

	"github.com/ReconfigureIO/platform/models"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres" // postgres driver
)

// MigrateSchema performs database migration.
func MigrateSchema() {
	gormConnDets := os.Getenv("DATABASE_URL")
	db, err := gorm.Open("postgres", gormConnDets)
	if err != nil {
		fmt.Println(err)
		panic("failed to connect database")
	}
	db.AutoMigrate(&models.InviteToken{})
	db.AutoMigrate(&models.User{})
	db.AutoMigrate(&models.Project{})
	db.AutoMigrate(&models.Simulation{})
	db.AutoMigrate(&models.Build{})
	db.AutoMigrate(&models.BatchJob{})
	db.AutoMigrate(&models.BatchJobEvent{})
	db.AutoMigrate(&models.Deployment{})
	db.AutoMigrate(&models.DeploymentEvent{})
	db.AutoMigrate(&models.Hours{})
}
