// +build integration

package models

import (
	"context"
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

func TestBatchGetLogName(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := BatchDataSource(db)
		batch := BatchJob{
			BatchID: "foo",
			LogName: "foobarLogName",
		}
		db.Create(&batch)
		returned, err := d.GetLogName(batch.BatchID)
		if err != nil {
			t.Error(err)
		}
		if batch.LogName != returned {
			t.Fatalf("Failed to get batch job's log name. Expected: %v Got: %v \n", batch.LogName, returned)
			return
		}
	})
}

func TestBatchAwaitStarted(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := BatchDataSource(db)

		batch := BatchJob{
			BatchID: "foo",
		}
		db.Create(&batch)

		err := d.AddEvent(batch, BatchJobEvent{
			BatchJobID: batch.ID,
			Status:     StatusStarted,
		})
		if err != nil {
			t.Error(err)
		}

		ctxtimeout, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err = BatchAwaitStarted(ctxtimeout, d, batch.BatchID, 100*time.Microsecond)
		if err != nil {
			t.Error(err)
		}
	})
}

func TestBatchHasStarted(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := BatchDataSource(db)
		batch := BatchJob{
			BatchID: "foo",
			Events: []BatchJobEvent{
				BatchJobEvent{
					Status: StatusStarted,
				},
			},
		}
		db.Create(&batch)

		started, err := d.HasStarted(batch.BatchID)
		if err != nil {
			t.Error(err)
		}
		if !started {
			t.Error("BatchJob with started event is not considered started")
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
			t.Fatalf("Expected 0 batch jobs, got %v", len(batchJobs))
			return
		}

	})
}
