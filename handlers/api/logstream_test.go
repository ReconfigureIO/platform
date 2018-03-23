//+build integration

package api

import (
	"fmt"
	"testing"
	"time"

	"github.com/ReconfigureIO/platform/models"
	"github.com/jinzhu/gorm"
)

func TestRefreshEvents(t *testing.T) {
	models.RunTransaction(func(db *gorm.DB) {
		//create a BatchJob in the DB
		timeNow := time.Now()
		timeLater := timeNow.Add(5 * time.Minute)
		batchJob := models.BatchJob{
			Events: []models.BatchJobEvent{
				models.BatchJobEvent{
					Status:    "STARTED",
					Timestamp: timeNow,
				},
				models.BatchJobEvent{
					Status:    "COMPLETED",
					Timestamp: timeLater,
				},
			},
		}
		err := db.Create(&batchJob).Error
		if err != nil {
			t.Error(err)
			return
		}

		err = refreshEvents(&batchJob, db)
		if err != nil {
			t.Error(err)
			return
		}

		if len(batchJob.Events) == 0 {
			t.Fatalf("Expected 2 events, got 0")
			return
		}

		fmt.Println(batchJob.Events[0])

		// Did the events come out in the logical order?
		if !(batchJob.Events[0].Status == "STARTED") {
			t.Fatalf("\nExpected: %+v\nGot:      %+v\n", "STARTED", batchJob.Events[0].Status)
			return
		}
	})
}

func TestRefreshEventsReverseOrder(t *testing.T) {
	models.RunTransaction(func(db *gorm.DB) {
		//create a BatchJob in the DB
		timeNow := time.Now()
		timeLater := timeNow.Add(5 * time.Minute)
		batchJob := models.BatchJob{
			Events: []models.BatchJobEvent{
				models.BatchJobEvent{
					Status:    "COMPLETED",
					Timestamp: timeLater,
				},
				models.BatchJobEvent{
					Status:    "STARTED",
					Timestamp: timeNow,
				},
			},
		}
		err := db.Create(&batchJob).Error
		if err != nil {
			t.Error(err)
			return
		}

		err = refreshEvents(&batchJob, db)
		if err != nil {
			t.Error(err)
			return
		}

		// Did the events come out in the logical order?
		if !(batchJob.Events[0].Status == "STARTED") {
			t.Fatalf("\nExpected: %+v\nGot:      %+v\n", "STARTED", batchJob.Events[0].Status)
			return
		}
	})
}
