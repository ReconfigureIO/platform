// +build integration

package models

import (
	"fmt"
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
			UserID: "Foo",
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

		if len(ids) != 0 {
			t.Fatalf("\nExpected: %+v\nGot:      %+v\n", 0, len(ids))
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

		deps, err := d.GetWithUser("foobar")
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

func TestDeploymentQuery(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := deploymentRepo{db}

		dep := Deployment{
			Command: "test",
			Events:  []DeploymentEvent{},
			UserID:  "foobar",
		}
		db.Create(&dep)

		deps := []Deployment{}
		err := d.Query(dep.UserID).Find(&deps).Error
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

func TestDeploymentQueryNotOtherUsers(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := deploymentRepo{db}

		dep := Deployment{
			Command: "test",
			Events:  []DeploymentEvent{},
			UserID:  "foobar",
		}
		db.Create(&dep)

		deps := []Deployment{}
		err := d.Query("notfoobar").Find(&deps).Error
		if err != nil {
			t.Error(err)
			return
		}

		ids := []string{}
		for _, returnedDep := range deps {
			ids = append(ids, returnedDep.ID)
		}

		if len(ids) != 0 {
			t.Fatalf("\nExpected: %+v\nGot:      %+v\n", 0, len(ids))
			return
		}
	})
}

func TestDeploymentQueryPreloading(t *testing.T) {
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
		deps := []Deployment{}
		err := d.Query("foobar").Find(&deps).Error
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

func TestDeploymentPreload(t *testing.T) {
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
		deps := []Deployment{}
		err := d.Preload().Find(&deps).Error
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
		now := time.Now()

		deps := []Deployment{
			genDeployment(userID, now, time.Hour),               // 1 hour
			genDeployment(userID, now, 0),                       // 0 hours
			genDeployment(userID, now, 0),                       // 0 hours
			genDeployment(userID, now, time.Hour*2),             // 2 hours
			genDeployment(userID, now, time.Hour+5*time.Minute), // 1 hour 5 minutes
		} // total 4 hours 5 minutes, rounds to 5 hours

		for i := range deps {
			db.Create(&(deps[i]))
		}

		hours, err := DeploymentHoursBtw(&d, userID, now, now.AddDate(0, 0, 1))
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

func TestDeploymentHoursBtwRunning(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := deploymentRepo{db}
		userid := "foobar"
		var zero time.Time
		now := time.Now()

		dep := Deployment{
			Build: Build{
				Project: Project{
					UserID: userid,
				},
			},
			Command: "test",
			Events: []DeploymentEvent{
				DeploymentEvent{
					Status:    "STARTED",
					Timestamp: now.Add(-10 * time.Hour),
				},
			},
			UserID: userid,
		}

		db.Create(&dep)

		hours, err := DeploymentHoursBtw(&d, userid, zero, now)
		if err != nil {
			t.Error(err)
			return
		}
		if hours != 10 {
			t.Errorf("Expected %v found %v", 10, hours)
		}
	})
}

func TestDeploymentHoursBtwFinished(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := deploymentRepo{db}
		userid := "foobar"
		var zero time.Time
		now := time.Now()

		dep := Deployment{
			Build: Build{
				Project: Project{
					UserID: userid,
				},
			},
			Command: "test",
			Events: []DeploymentEvent{
				DeploymentEvent{
					Status:    "STARTED",
					Timestamp: now.Add(-10 * time.Hour),
				},
				DeploymentEvent{
					Status:    "TERMINATED",
					Timestamp: now.Add(-5 * time.Hour),
				},
			},
			UserID: userid,
		}

		db.Create(&dep)

		hours, err := DeploymentHoursBtw(&d, userid, zero, now)
		if err != nil {
			t.Error(err)
			return
		}
		if hours != 5 {
			t.Errorf("Expected %v found %v", 5, hours)
		}
	})
}

func TestDeploymentHoursBtwRunningFromBeforeStart(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := deploymentRepo{db}
		userid := "foobar"
		now := time.Now()

		dep := Deployment{
			Build: Build{
				Project: Project{
					UserID: userid,
				},
			},
			Command: "test",
			Events: []DeploymentEvent{
				DeploymentEvent{
					Status:    "STARTED",
					Timestamp: now.Add(-10 * time.Hour),
				},
			},
			UserID: userid,
		}

		db.Create(&dep)

		hours, err := DeploymentHoursBtw(&d, userid, now.Add(-5*time.Hour), now)
		if err != nil {
			t.Error(err)
			return
		}
		if hours != 5 {
			t.Errorf("Expected %v found %v", 5, hours)
		}
	})
}

func TestDeploymentHoursBtwFinishedBeforeStart(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := deploymentRepo{db}
		userid := "foobar"
		now := time.Now()

		dep := Deployment{
			Build: Build{
				Project: Project{
					UserID: userid,
				},
			},
			Command: "test",
			Events: []DeploymentEvent{
				DeploymentEvent{
					Status:    "STARTED",
					Timestamp: now.Add(-10 * time.Hour),
				},
				DeploymentEvent{
					Status:    "TERMINATED",
					Timestamp: now.Add(-5 * time.Hour),
				},
			},
			UserID: userid,
		}

		db.Create(&dep)

		hours, err := DeploymentHoursBtw(&d, userid, now.Add(-4*time.Hour), now)
		if err != nil {
			t.Error(err)
			return
		}
		if hours != 0 {
			t.Errorf("Expected %v found %v", 0, hours)
		}
	})
}

func TestDeploymentHoursBtwStartedBeforeStartTerminatedAfterStart(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := deploymentRepo{db}
		userid := "foobar"
		now := time.Now()

		dep := Deployment{
			Build: Build{
				Project: Project{
					UserID: userid,
				},
			},
			Command: "test",
			Events: []DeploymentEvent{
				DeploymentEvent{
					Status:    "STARTED",
					Timestamp: now.Add(-10 * time.Hour),
				},
				DeploymentEvent{
					Status:    "TERMINATED",
					Timestamp: now.Add(-1 * time.Hour),
				},
			},
			UserID: userid,
		}

		db.Create(&dep)

		hours, err := DeploymentHoursBtw(&d, userid, now.Add(-2*time.Hour), now)
		if err != nil {
			t.Error(err)
			return
		}
		if hours != 1 {
			t.Errorf("Expected %v found %v", 1, hours)
		}
	})
}

func TestDeploymentHoursBtwWithSlowGoClock(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := deploymentRepo{db}
		userid := "foobar"
		var zero time.Time
		now := time.Now()

		dep := Deployment{
			Build: Build{
				Project: Project{
					UserID: userid,
				},
			},
			Command: "test",
			Events: []DeploymentEvent{
				DeploymentEvent{
					Status:    "STARTED",
					Timestamp: now.Add(-10 * time.Hour),
				},
			},
			UserID: userid,
		}

		db.Create(&dep)

		hours, err := DeploymentHoursBtw(&d, userid, zero, now.Add(-5*time.Second))
		if err != nil {
			t.Error(err)
			return
		}
		if hours != 10 {
			t.Errorf("Expected %v found %v", 10, hours)
		}
	})
}

func TestDeploymentHoursStartedNoTerminated(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := deploymentRepo{db}
		now := time.Now()

		dep := Deployment{
			UserID:  "foobar",
			Command: "test",
			Events: []DeploymentEvent{
				DeploymentEvent{
					Timestamp: now.AddDate(0, 0, -1),
					Status:    "STARTED",
				},
				DeploymentEvent{
					Timestamp: now.AddDate(0, 0, -2),
					Status:    "QUEUED",
				},
			},
		}

		db.Create(&dep)

		depHours, err := d.DeploymentHours(dep.UserID, now.AddDate(0, 0, -3), now)
		if err != nil {
			t.Error(err)
			return
		}
		if len(depHours) <= 0 {
			t.Error("expected more deployments")
		}
		for _, depHour := range depHours {
			if depHour.Started.After(depHour.Terminated) {
				fmt.Println("Starts at: ", depHour.Started)
				fmt.Println("Ends at: ", depHour.Terminated)
				t.Errorf("Deployment starts after it ends")

			}
		}
	})
}

func TestAggregateHoursBetween(t *testing.T) {
	now := time.Now()

	depHour := []DeploymentHours{
		DeploymentHours{
			Id:         "foobar",
			Started:    now.AddDate(0, 0, -1),
			Terminated: time.Time{},
		},
	}

	hours := AggregateHoursBetween(depHour, now.AddDate(0, 0, -3), now)
	if hours != 24 {
		t.Errorf("Expected: 24, Got: %s", hours)
	}
}

func TestAggregateHoursBetweenNoStarted(t *testing.T) {
	now := time.Now()

	depHour := []DeploymentHours{
		DeploymentHours{
			Id:         "foobar",
			Started:    time.Time{},
			Terminated: now.AddDate(0, 0, -1),
		},
	}

	hours := AggregateHoursBetween(depHour, now.AddDate(0, 0, -3), now)
	if hours != 0 {
		t.Errorf("Expected: 0, Got: %s", hours)
	}
}

func TestDeploymentHoursBtwWithRealTimes(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := deploymentRepo{db}
		userID := "user1"

		deps := []Deployment{
			genDeploymentWithTimestamps(userID, timeParser("2017-11-13T15:53:36.554449Z"), timeParser("2017-11-13T16:06:49.249005Z")),
			genDeploymentWithTimestamps(userID, timeParser("2017-11-06T12:19:54.604407Z"), timeParser("2017-11-06T12:21:35.229967Z")),
			Deployment{
				UserID: userID,
				Events: []DeploymentEvent{
					DeploymentEvent{
						Status:    "QUEUED",
						Timestamp: timeParser("2017-11-13T15:42:31.516901Z"),
					},
					DeploymentEvent{
						Status:    "TERMINATED",
						Timestamp: timeParser("2017-11-13T15:43:28.309Z"),
					},
				},
			},
		} // total 2 hours

		for i := range deps {
			db.Create(&(deps[i]))
		}

		utcLocation, _ := time.LoadLocation("UTC")
		firstOfMonth := time.Date(2017, 11, 1, 0, 0, 0, 0, utcLocation)
		lastOfMonth := firstOfMonth.AddDate(0, 1, -1)

		hours, err := DeploymentHoursBtw(&d, userID, firstOfMonth, lastOfMonth)
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
		UserID: userID,
	}
	if d > 0 {
		dep.Events = append(dep.Events, DeploymentEvent{
			Status:    "TERMINATED",
			Timestamp: start.Add(d),
		})
	}
	return dep
}

func genDeploymentWithTimestamps(userID string, start time.Time, end time.Time) Deployment {
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
			DeploymentEvent{
				Status:    "TERMINATED",
				Timestamp: end,
			},
		},
		UserID: userID,
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

func timeParser(stringTime string) time.Time {
	parsedTime, _ := time.Parse(time.RFC3339, stringTime)
	return parsedTime
}
