// +build integration

package models

import (
	"reflect"
	"testing"

	"github.com/jinzhu/gorm"
)

func TestDeploymentGetWithStatus(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := PostgresRepo{db}

		dep := Deployment{
			Command: "test",
			DepJob: DepJob{
				DepID: "Bar",
				Events: []DepJobEvent{
					DepJobEvent{
						DepJobID: "Bar",
						Status:   "COMPLETED",
					},
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
