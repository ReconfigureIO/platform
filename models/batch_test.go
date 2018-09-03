// +build integration

package models

import (
	"reflect"
	"testing"
	"time"

	"github.com/jinzhu/gorm"
)

func TestBatchAddEvent(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := BatchDataSource(db)
		// create a build in the DB
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

func TestBatchSetLogName(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := BatchDataSource(db)
		batch := BatchJob{
			BatchID: "foo",
		}
		db.Create(&batch)
		d.SetLogName(batch.BatchID, "bar")

		db.First(&batch)
		if batch.LogName != "bar" {
			t.Fatal("Failed to set batch job's log name")
			return
		}
	})
}

func TestBatchActiveJobsWithoutLogs(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := BatchDataSource(db)

		batch := BatchJob{
			ID: 123456789,
			Events: []BatchJobEvent{
				BatchJobEvent{
					Timestamp: time.Unix(20, 0),
					Status:    "STARTED",
				},
				BatchJobEvent{
					Timestamp: time.Unix(0, 0),
					Status:    "QUEUED",
				},
			},
		}
		err := db.Create(&batch).Error
		if err != nil {
			t.Error(err)
			return
		}

		batchJobs, err := d.ActiveJobsWithoutLogs(time.Unix(0, 0))
		if err != nil {
			t.Error(err)
			return
		}

		ids := []int64{}
		for _, returnedBatchJob := range batchJobs {
			ids = append(ids, returnedBatchJob.ID)
		}

		expected := []int64{batch.ID}
		if !reflect.DeepEqual(expected, ids) {
			t.Fatalf("\nExpected: %+v\nGot:      %+v\n", expected, ids)
			return
		}

		if !reflect.DeepEqual(batchJobs[0].LogName, "") {
			t.Fatalf("\nExpected dep to have Null Log Name but got:      %+v\n", batchJobs[0].LogName)
			return
		}
	})
}

func TestBatchActiveJobsWithoutLogsWithLogs(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := BatchDataSource(db)

		batch := BatchJob{
			ID:      123456789,
			LogName: "foo",
			Events: []BatchJobEvent{
				BatchJobEvent{
					Timestamp: time.Unix(20, 0),
					Status:    "STARTED",
				},
				BatchJobEvent{
					Timestamp: time.Unix(0, 0),
					Status:    "QUEUED",
				},
			},
		}
		err := db.Create(&batch).Error
		if err != nil {
			t.Error(err)
			return
		}

		batchJobs, err := d.ActiveJobsWithoutLogs(time.Unix(0, 0))
		if err != nil {
			t.Error(err)
			return
		}

		if len(batchJobs) != 0 {
			t.Fatal("Expected 0 batch jobs, got %s", len(batchJobs))
			return
		}

	})
}

func TestGetBatchJobsWithStatus(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		batchRepo := BatchDataSource(db)
		// create a build in the DB
		batchJob := BatchJob{
			Events: []BatchJobEvent{
				BatchJobEvent{
					Status: "COMPLETED",
				},
			},
		}
		db.Create(&batchJob)
		// run the get with status function
		batchJobs, err := batchRepo.GetBatchJobsWithStatus([]string{"COMPLETED"}, 10)
		if err != nil {
			t.Error(err)
			return
		}

		ids := []string{}
		for _, returnedBatchJob := range batchJobs {
			ids = append(ids, string(returnedBatchJob.ID))
		}
		// return from get with status should match the build we made at the start
		expected := []string{string(batchJob.ID)}
		if !reflect.DeepEqual(expected, ids) {
			t.Fatalf("\nExpected: %+v\nGot:      %+v\n", expected, batchJobs)
			return
		}
	})
}
