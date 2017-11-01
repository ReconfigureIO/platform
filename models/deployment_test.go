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
					Timestamp: time.Unix(20, 0),
					Status:    "COMPLETED",
				},
				DeploymentEvent{
					Timestamp: time.Unix(0, 0),
					Status:    "QUEUED",
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
			t.Fatalf("\nExpected: %+v\nGot:      %+v\n", expected, ids)
			return
		}

		if !reflect.DeepEqual(deps[0].Status(), "COMPLETED") {
			t.Fatalf("\nExpected dep to have status: %+v\nGot:      %+v\n", "COMPLETED", deps[0].Status())
			return
		}
	})
}

func TestDeploymentGetWithoutIP(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := deploymentRepo{db}

		dep := Deployment{
			Command:   "test",
			IPAddress: "",
			Events: []DeploymentEvent{
				DeploymentEvent{
					Timestamp: time.Unix(20, 0),
					Status:    "STARTED",
				},
				DeploymentEvent{
					Timestamp: time.Unix(0, 0),
					Status:    "QUEUED",
				},
			},
		}
		db.Create(&dep)

		deps, err := d.GetWithoutIP()
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
			t.Fatalf("\nExpected: %+v\nGot:      %+v\n", expected, ids)
			return
		}

		if !reflect.DeepEqual(deps[0].IPAddress, "") {
			t.Fatalf("\nExpected dep to have Null IP: %+v\nGot:      %+v\n", "COMPLETED", deps[0].IPAddress)
			return
		}
	})
}

func TestDeploymentHoursBtw(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := deploymentRepo{db}

		dep := genDeployment("Foo", time.Hour+5*time.Minute)

		db.Create(&dep)

		var zero time.Time
		now := time.Now()
		hours, err := DeploymentHoursBtw(&d, dep.Build.Project.UserID, zero, now)
		if err != nil {
			t.Error(err)
			return
		}
		if hours != 2 {
			t.Errorf("Expected %v found %v", 2, hours)
		}
	})
}

// genDeployment generates a mock deployment.
// if d > 0, the mock deployment will be in TERMINATED
// status and have a duration of d.
func genDeployment(userID string, d time.Duration) Deployment {
	var start = (time.Unix(0, 0)).Add(time.Hour)
	dep := Deployment{
		Build: Build{
			Project: Project{
				UserID: userID,
			},
		},
		Command: "test",
		Events: []DeploymentEvent{
			DeploymentEvent{
				Status:    "STARTED",
				Timestamp: start,
			},
		},
	}
	if d > 0 {
		dep.Events = append(dep.Events, DeploymentEvent{
			Status:    "TERMINATED",
			Timestamp: start.Add(d),
		})
	}
	return dep
}

func TestDeploymentActiveDeployments(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := deploymentRepo{db}

		userID := "user1"

		deps := []Deployment{
			genDeployment(userID, time.Hour),
			genDeployment(userID, 0),
			genDeployment(userID, 0),
			genDeployment(userID, time.Hour*2),
		}

		for i := range deps {
			db.Create(&(deps[i]))
		}

		ad, err := d.ActiveDeployments(userID)
		if err != nil {
			t.Error(err)
			return
		}
		if l := len(ad); l != 2 {
			t.Errorf("Expected %v found %v", 2, l)
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
