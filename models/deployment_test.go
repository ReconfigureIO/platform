package models

import (
	"os"
	"testing"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

func TestGetWithStatus(t *testing.T) {
	gormConnDets := os.Getenv("DATABASE_URL")
	db, err := gorm.Open("postgres", gormConnDets)
	if err != nil {
		t.Error(err)
		return
	}

	d := (*DB)(db)

	_, err = d.GetWithStatus([]string{"COMPLETED"}, 10)
	if err != nil {
		t.Error(err)
		return
	}
}
