package migration

import (
	"fmt"
	"os"

	"github.com/ReconfigureIO/platform/migration/migration1"
	"github.com/ReconfigureIO/platform/migration/migration201711131234"
	//"github.com/ReconfigureIO/platform/migration/migration201801260948"
	"github.com/ReconfigureIO/platform/migration/migration201801260952"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	log "github.com/sirupsen/logrus"
	"gopkg.in/gormigrate.v1"
)

var migrations = []*gormigrate.Migration{
	&migration1.Migration,
	&migration201711131234.Migration,
	//&migration201801260948.Migration,
	&migration201801260952.Migration,
}

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

	if err := m.Migrate(); err != nil {
		log.Fatalf("Could not migrate: %v", err)
	}
	log.Printf("Migration did run successfully")
}
