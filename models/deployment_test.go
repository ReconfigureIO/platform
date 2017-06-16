package models

import (
	"os"
	"testing"
	"time"

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

	d := deploymentRepo{db}

	_, err = d.GetWithStatus([]string{"COMPLETED"}, 10)
	if err != nil {
		t.Error(err)
		return
	}
}

func TestTimeToSQLStr(t *testing.T) {
	utcTime := time.Date(2010, 2, 11, 3, 20, 30, 0, time.UTC)
	expected := "2010-02-01 00:00:00"
	if ms := timeToSQLStr(monthStart(utcTime)); ms != expected {
		t.Errorf("Expected %v found %v", expected, ms)
	}
	expected = "2010-02-28 23:59:59"
	if ms := timeToSQLStr(monthEnd(utcTime)); ms != expected {
		t.Errorf("Expected %v found %v", expected, ms)
	}
}
