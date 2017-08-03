// +build integration

package models

import (
	"reflect"
	"testing"
	"time"

	"github.com/jinzhu/gorm"
)

func TestDeploymentGetWithStatusForUser(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := deploymentRepo{db}

		dep := Deployment{
			Build: Build{
				Project: Project{
					UserID: "Foo",
				},
			},
			Command: "test",
			Events: []DeploymentEvent{
				DeploymentEvent{
					Status: "COMPLETED",
				},
			},
		}
		db.Create(&dep)

		deps, err := d.GetWithStatusForUser("Foo", []string{"COMPLETED"})
		if err != nil {
			t.Error(err)
			return
		}

		ids := []string{}
		for _, returnedDep := range deps {
			ids = append(ids, returnedDep.ID)
		}

		expected := []string{dep.ID}
		if !reflect.DeepEqual(expected, ids) {
			t.Fatalf("\nExpected: %+v\nGot:      %+v\n", expected, deps)
			return
		}
	})
}

func TestDeploymentGetWithStatus(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := deploymentRepo{db}

		dep := Deployment{
			Command: "test",
			Events: []DeploymentEvent{
				DeploymentEvent{
					Status: "COMPLETED",
				},
			},
		}
		db.Create(&dep)

		deps, err := d.GetWithStatus([]string{"COMPLETED"}, 10)
		if err != nil {
			t.Error(err)
			return
		}

		ids := []string{}
		for _, returnedDep := range deps {
			ids = append(ids, returnedDep.ID)
		}

		expected := []string{dep.ID}
		if !reflect.DeepEqual(expected, ids) {
			t.Fatalf("\nExpected: %+v\nGot:      %+v\n", expected, deps)
			return
		}
	})
}

func TestDeploymentHoursHours(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := deploymentRepo{db}

		zeroTime := time.Unix(0, 0)

		dep := Deployment{
			Build: Build{
				Project: Project{
					UserID: "Foo",
				},
			},
			Command: "test",
			Events: []DeploymentEvent{
				DeploymentEvent{
					Status:    "STARTED",
					Timestamp: zeroTime.Add(time.Hour),
				},
				DeploymentEvent{
					Status:    "TERMINATED",
					Timestamp: zeroTime.Add(2*time.Hour + (5 * time.Minute)),
				},
			},
		}
		db.Create(&dep)

		now := time.Now()
		hours, err := DeploymentHoursBtw(&d, dep.Build.Project.UserID, zeroTime, now)
		if err != nil {
			t.Error(err)
			return
		}
		if hours != 2 {
			t.Errorf("Expected %v found %v", 2, hours)
		}
	})
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
