package models

import (
	"github.com/jinzhu/gorm"
)

func MigrateAll(db *gorm.DB) {
	db.AutoMigrate(&InviteToken{})
	db.AutoMigrate(&User{})
	db.AutoMigrate(&Project{})
	db.AutoMigrate(&Simulation{})
	db.AutoMigrate(&Build{})
	db.AutoMigrate(&BatchJob{})
	db.AutoMigrate(&BatchJobEvent{})
	db.AutoMigrate(&Deployment{})
	db.AutoMigrate(&DeploymentEvent{})
	db.AutoMigrate(&BuildReport{})
	db.AutoMigrate(&Graph{})
	db.AutoMigrate(&Credit{})
	db.AutoMigrate(&Debit{})
}
