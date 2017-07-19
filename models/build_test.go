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
