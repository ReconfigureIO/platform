package main

import (
	"fmt"
	"os"

	"github.com/ReconfigureIO/platform/models"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

func setupDB() {
	gormConnDets := os.Getenv("DATABASE_URL")
	db, err := gorm.Open("postgres", gormConnDets)
	if err != nil {
		fmt.Println(err)
		panic("failed to connect database")
	}
	db.AutoMigrate(&models.Simulation{})
	db.AutoMigrate(&models.Build{})
	db.AutoMigrate(&models.Project{})
	db.AutoMigrate(&models.User{})
}

func main() {

	// setup components
	setupDB()
}
