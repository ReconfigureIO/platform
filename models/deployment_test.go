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

func TestDeploymentGetWithUser(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := deploymentRepo{db}

		dep := Deployment{
			Command: "test",
			Events:  []DeploymentEvent{},
			UserID:  "foobar",
		}
		db.Create(&dep)

		deps, err := d.GetWithUser(dep.UserID)
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

		if !reflect.DeepEqual(deps[0].UserID, dep.UserID) {
			t.Fatalf("\nExpected dep to have user: %+v\nGot:      %+v\n", dep.UserID, deps[0].UserID)
			return
		}
	})
}

func TestDeploymentGetWithUserNotOtherUsers(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := deploymentRepo{db}

		dep := Deployment{
			Command: "test",
			Events:  []DeploymentEvent{},
			UserID:  "foobar",
		}
		db.Create(&dep)

		deps, err := d.GetWithUser("notfoobar")
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

		if !reflect.DeepEqual(deps[0].UserID, "notfoobar") {
			t.Fatalf("\nExpected dep to have user: %+v\nGot:      %+v\n", "notfoobar", deps[0].UserID)
			return
		}
	})
}

func TestDeploymentGetWithUserPreloading(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := deploymentRepo{db}

		dep := Deployment{
			Command: "test",
			Events:  []DeploymentEvent{},
			UserID:  "foobar",
			Build: Build{
				Project: Project{
					User: User{
						Name: "Foo Bar",
					},
				},
			},
		}
		db.Create(&dep)

		deps, err := d.GetWithUser("notfoobar")
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

		if !reflect.DeepEqual(deps[0].Build.Project.User.Name, dep.Build.Project.User.Name) {
			t.Fatalf("\nExpected dep to have user: %+v\nGot:      %+v\n", dep.Build.Project.User.Name, deps[0].Build.Project.User.Name)
			return
		}
	})
}

func TestDeploymentGetWithoutIP(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := deploymentRepo{db}

		dep := Deployment{
			Command: "test",
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
		err := db.Create(&dep).Error
		if err != nil {
			t.Error(err)
			return
		}

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

		userID := "user1"
		var zero time.Time
		now := time.Now()

		deps := []Deployment{
			genDeployment(userID, zero, time.Hour),               // 1 hour
			genDeployment(userID, zero, 0),                       // 0 hours
			genDeployment(userID, zero, 0),                       // 0 hours
			genDeployment(userID, zero, time.Hour*2),             // 2 hours
			genDeployment(userID, zero, time.Hour+5*time.Minute), // 1 hour 5 minutes
		} // total 4 hours 5 minutes, rounds to 5 hours

		for i := range deps {
			db.Create(&(deps[i]))
		}

		hours, err := DeploymentHoursBtw(&d, userID, zero, now)
		if err != nil {
			t.Error(err)
			return
		}
		if hours != 5 {
			t.Errorf("Expected %v found %v", 5, hours)
		}
	})
}

func TestDeploymentHoursBtwWithNoEvents(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := deploymentRepo{db}

		dep := Deployment{
			Build: Build{
				Project: Project{
					UserID: "foobar",
				},
			},
			Command: "test",
			Events:  []DeploymentEvent{},
		}

		db.Create(&dep)

		var zero time.Time
		now := time.Now()
		hours, err := DeploymentHoursBtw(&d, dep.Build.Project.UserID, zero, now)
		if err != nil {
			t.Error(err)
			return
		}
		if hours != 0 {
			t.Errorf("Expected %v found %v", 0, hours)
		}
	})
}

// genDeployment generates a mock deployment.
// if d > 0, the mock deployment will be in TERMINATED
// status and have a duration of d.
func genDeployment(userID string, start time.Time, d time.Duration) Deployment {
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
		var zero time.Time
		userID := "user1"

		deps := []Deployment{
			genDeployment(userID, zero, time.Hour),
			genDeployment(userID, zero, 0),
			genDeployment(userID, zero, 0),
			genDeployment(userID, zero, time.Hour*2),
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
