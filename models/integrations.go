// +build integration

package models

import (
	"log"
	"os"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres" // postgres driver
	stripe "github.com/stripe/stripe-go"
)

func init() {
	stripe.Key = os.Getenv("STRIPE_KEY")
	gormConnDets := os.Getenv("DATABASE_URL")
	db, err := gorm.Open("postgres", gormConnDets)
	if err != nil {
		log.Fatalf("failed to connect database. Error: %v", err)
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
