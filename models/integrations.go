// +build integration

package models

import (
	"fmt"
	"os"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres" // postgres driver
)

func init() {
	gormConnDets := os.Getenv("DATABASE_URL")
	db, err := gorm.Open("postgres", gormConnDets)
	if err != nil {
		fmt.Println(err)
		panic("failed to connect database")
	}
	MigrateAll(db)
	_db = db
}

var _db *gorm.DB

// Transaction runs a transaction, and then rolls back
func RunTransaction(ops func(db *gorm.DB)) {
	tx := _db.Begin()
	ops(tx)
	tx.Rollback()
}
