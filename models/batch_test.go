// +build integration

package models

import (
	"testing"

	"github.com/jinzhu/gorm"
)

func TestBatchAddEvent(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := BatchDataSource(db)
		//create a build in the DB
		batch := BatchJob{
			Events: []BatchJobEvent{
				BatchJobEvent{
					Status: "Started",
				},
			},
		}
		db.Create(&batch)
		d.AddEvent(batch, BatchJobEvent{
			Status: "Completed",
		})

		newBatch := BatchJob{}
		db.Preload("Events").Where("id = ?", batch.ID).First(&newBatch)

		if len(newBatch.Events) != 2 {
			t.Fatalf("\nExpected: %+v\nGot:      %+v\n", 2, len(newBatch.Events))
			return
		}
	})
}
