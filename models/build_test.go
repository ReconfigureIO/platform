// +build integration

package models

import (
	"reflect"
	"testing"
	"time"

	"github.com/jinzhu/gorm"
)

func TestGetBuildsWithStatus(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := BuildDataSource(db)
		//create a build in the DB
		build := Build{
			BatchJob: BatchJob{
				Events: []BatchJobEvent{
					BatchJobEvent{
						Status: "COMPLETED",
					},
				},
			},
		}
		db.Create(&build)
		//run the get with status function
		builds, err := d.GetBuildsWithStatus([]string{"COMPLETED"}, 10)
		if err != nil {
			t.Error(err)
			return
		}

		ids := []string{}
		for _, returnedBuild := range builds {
			ids = append(ids, returnedBuild.ID)
		}
		//return from get with status should match the build we made at the start
		expected := []string{build.ID}
		if !reflect.DeepEqual(expected, ids) {
			t.Fatalf("\nExpected: %+v\nGot:      %+v\n", expected, builds)
			return
		}
	})
}

func TestCreateBuildReport(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := BuildDataSource(db)
		//create a build in the DB
		build := Build{}
		db.Create(&build)
		report := ReportV1{}
		//run the get with status function
		err := d.StoreBuildReport(build, report)
		if err != nil {
			t.Error(err)
			return
		}
		return
	})
}

// genBuild generates a mock build.
// if duration > 0, the mock build will be in TERMINATED
// status and have a duration of `duration`.
func genBuild(userID string, duration time.Duration) Build {
	var start = (time.Unix(0, 0)).Add(time.Hour)
	build := Build{
		Project: Project{
			UserID: userID,
		},
		BatchJob: BatchJob{
			Events: []BatchJobEvent{
				BatchJobEvent{
					Status:    "STARTED",
					Timestamp: start,
				},
			},
		},
	}
	if duration > 0 {
		build.BatchJob.Events = append(build.BatchJob.Events, BatchJobEvent{
			Status:    "TERMINATED",
			Timestamp: start.Add(duration),
		})
	}
	return build
}

func TestActiveBuilds(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		buildData := buildRepo{db}

		user := User{
			ID: "foobar",
		}

		builds := []Build{
			genBuild(user.ID, time.Hour),
			genBuild(user.ID, 0),
			genBuild(user.ID, 0),
			genBuild(user.ID, time.Hour*2),
		}

		for i := range builds {
			db.Create(&(builds[i]))
		}

		activeBuilds, err := buildData.ActiveBuilds(user)
		if err != nil {
			t.Error(err)
			return
		}
		if l := len(activeBuilds); l != 2 {
			t.Errorf("Expected %v found %v", 2, l)
		}
	})
}
