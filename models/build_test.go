// +build integration

package models

import (
	"reflect"
	"testing"

	"github.com/jinzhu/gorm"
)

func TestGetBuildsWithStatus(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := BuildDataSource(db)
		// create a build in the DB
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
		// run the get with status function
		builds, err := d.GetBuildsWithStatus([]string{"COMPLETED"}, 10)
		if err != nil {
			t.Error(err)
			return
		}

		ids := []string{}
		for _, returnedBuild := range builds {
			ids = append(ids, returnedBuild.ID)
		}
		// return from get with status should match the build we made at the start
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
		// create a build in the DB
		build := Build{}
		db.Create(&build)
		report := Report{}
		// run the get with status function
		err := d.StoreBuildReport(build, report)
		if err != nil {
			t.Error(err)
			return
		}
		return
	})
}

func TestBuildByIDForUser(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		users := []User{
			User{Email: "foo@bar.baz", GithubID: 1},
			User{Email: "baz@bar.foo", GithubID: 2},
		}
		for _, user := range users {
			db.Save(&user)
		}

		projects := []Project{
			Project{UserID: users[0].ID},
			Project{UserID: users[1].ID},
		}
		for _, project := range projects {
			db.Save(&project)
		}

		builds := []Build{
			Build{ProjectID: projects[0].ID},
			Build{ProjectID: projects[0].ID},
			Build{ProjectID: projects[1].ID},
			Build{ProjectID: projects[1].ID},
		}
		for _, build := range builds {
			db.Save(&build)
		}

		var testCombos = []struct {
			buildID string
			userID  string
			ret     Build
		}{
			{builds[0].ID, users[0].ID, builds[0]},
			{builds[0].ID, users[1].ID, Build{}},
			{builds[1].ID, users[0].ID, builds[1]},
			{builds[1].ID, users[1].ID, Build{}},
			{builds[2].ID, users[0].ID, Build{}},
			{builds[2].ID, users[1].ID, builds[2]},
			{builds[3].ID, users[0].ID, Build{}},
			{builds[3].ID, users[1].ID, builds[3]},
		}
		d := BuildDataSource(db)
		for _, testCombo := range testCombos {
			var build Build
			build, err := d.ByIDForUser(testCombo.buildID, testCombo.userID)
			if build.ID != testCombo.ret.ID {
				t.Errorf("Error during testcombo: %v, %v", testCombo, err)
			}
			if build.Project.UserID != testCombo.userID {
				t.Error("Preload error during testcombo")
			}
		}
	})
}

func TestBuildByIDForProject(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		projects := []Project{
			Project{UserID: "foo"},
			Project{UserID: "bar"},
		}
		for _, project := range projects {
			db.Save(&project)
		}

		builds := []Build{
			Build{ProjectID: projects[0].ID},
			Build{ProjectID: projects[1].ID},
		}
		for _, build := range builds {
			db.Save(&build)
		}

		var testCombos = []struct {
			buildID   string
			projectID string
			ret       Build
		}{
			{builds[0].ID, projects[0].ID, builds[0]},
			{builds[0].ID, projects[1].ID, Build{}},
			{builds[1].ID, projects[0].ID, Build{}},
			{builds[1].ID, projects[1].ID, builds[1]},
		}

		d := BuildDataSource(db)
		for _, testCombo := range testCombos {
			var build Build
			build, err := d.ByIDForProject(testCombo.buildID, testCombo.projectID)
			if build.ID != testCombo.ret.ID {
				t.Errorf("Error during testcombo: %v, %v", testCombo, err)
			}
		}
	})
}
