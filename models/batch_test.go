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

func TestBatchSetCwLogName(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := BatchDataSource(db)
		batch := BatchJob{
			BatchID: "foo",
		}
		db.Create(&batch)
		d.SetCwLogName(batch.BatchID, "bar")

		db.First(&batch)
		if batch.CwLogName != "bar" {
			t.Fatal("Failed to set batch job's log name")
			return
		}
	})
}
