package models

import (
	"os"
	"testing"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

func TestDeploymentGetWithStatus(t *testing.T) {
	gormConnDets := os.Getenv("DATABASE_URL")
	if gormConnDets == "" {
		t.Skip()
		return
	}

	db, err := gorm.Open("postgres", gormConnDets)
	if err != nil {
		t.Error(err)
		return
	}

	d := PostgresRepo{db}

	_, err = d.GetWithStatus([]string{"COMPLETED"}, 10)
	if err != nil {
		t.Error(err)
		return
	}
}
