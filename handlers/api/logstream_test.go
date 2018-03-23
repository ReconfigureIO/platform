//+build integration

package api

import (
	"testing"
	"time"

	"github.com/ReconfigureIO/platform/models"
	"github.com/jinzhu/gorm"
)

func TestRefreshBatchJobEvents(t *testing.T) {
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

		err = refreshBatchJobEvents(&batchJob, db)
		if err != nil {
			t.Error(err)
			return
		}

		if len(batchJob.Events) == 0 {
			t.Fatalf("Expected 2 events, got 0")
			return
		}

		// Did the events come out in the logical order?
		if !(batchJob.Events[0].Status == "STARTED") {
			t.Fatalf("\nExpected: %+v\nGot:      %+v\n", "STARTED", batchJob.Events[0].Status)
			return
		}
	})
}

func TestRefreshBatchJobEventsReverseOrder(t *testing.T) {
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

		err = refreshBatchJobEvents(&batchJob, db)
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

func TestRefreshDeploymentEvents(t *testing.T) {
	models.RunTransaction(func(db *gorm.DB) {
		//create a Deployment in the DB
		timeNow := time.Now()
		timeLater := timeNow.Add(5 * time.Minute)
		deployment := models.Deployment{
			Events: []models.DeploymentEvent{
				models.DeploymentEvent{
					Status:    "STARTED",
					Timestamp: timeNow,
				},
				models.DeploymentEvent{
					Status:    "COMPLETED",
					Timestamp: timeLater,
				},
			},
		}
		err := db.Create(&deployment).Error
		if err != nil {
			t.Error(err)
			return
		}

		err = refreshDeploymentEvents(&deployment, db)
		if err != nil {
			t.Error(err)
			return
		}

		if len(deployment.Events) == 0 {
			t.Fatalf("Expected 2 events, got 0")
			return
		}

		// Did the events come out in the logical order?
		if !(deployment.Events[0].Status == "STARTED") {
			t.Fatalf("\nExpected: %+v\nGot:      %+v\n", "STARTED", deployment.Events[0].Status)
			return
		}
	})
}

func TestRefreshDeploymentEventsReverseOrder(t *testing.T) {
	models.RunTransaction(func(db *gorm.DB) {
		//create a Deployment in the DB
		timeNow := time.Now()
		timeLater := timeNow.Add(5 * time.Minute)
		deployment := models.Deployment{
			Events: []models.DeploymentEvent{
				models.DeploymentEvent{
					Status:    "COMPLETED",
					Timestamp: timeLater,
				},
				models.DeploymentEvent{
					Status:    "STARTED",
					Timestamp: timeNow,
				},
			},
		}
		err := db.Create(&deployment).Error
		if err != nil {
			t.Error(err)
			return
		}

		err = refreshDeploymentEvents(&deployment, db)
		if err != nil {
			t.Error(err)
			return
		}

		// Did the events come out in the logical order?
		if !(deployment.Events[0].Status == "STARTED") {
			t.Fatalf("\nExpected: %+v\nGot:      %+v\n", "STARTED", deployment.Events[0].Status)
			return
		}
	})
}

func TestRefreshBatchJobEventsAppended(t *testing.T) {
	models.RunTransaction(func(db *gorm.DB) {
		for i := 0; i < 100; i++ {
			//create a BatchJob in the DB
			timeNow := time.Now()
			timeLater := timeNow.Add(5 * time.Minute)
			batchJob := models.BatchJob{
				Events: []models.BatchJobEvent{
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

			err = refreshBatchJobEvents(&batchJob, db)
			if err != nil {
				t.Error(err)
				return
			}

			if !(len(batchJob.Events) == 1) {
				t.Fatalf("Expected 1 events, got %s", len(batchJob.Events))
				return
			}

			if !(batchJob.Events[0].Status == "STARTED") {
				t.Fatalf("\nExpected: %+v\nGot:      %+v\n", "STARTED", batchJob.Events[0].Status)
				return
			}

			completedEvent := models.BatchJobEvent{
				Status:    "COMPLETED",
				Timestamp: timeLater,
			}
			batchRepo := models.BatchDataSource(db)
			batchRepo.AddEvent(batchJob, completedEvent)

			err = refreshBatchJobEvents(&batchJob, db)
			if err != nil {
				t.Error(err)
				return
			}

			if !(len(batchJob.Events) == 2) {
				t.Fatalf("Expected 2 events, got %s", len(batchJob.Events))
				return
			}

			if !(batchJob.Events[0].Status == "STARTED") {
				t.Fatalf("\nExpected: %+v\nGot:      %+v\n", "STARTED", batchJob.Events[0].Status)
				return
			}

			if !(batchJob.Events[1].Status == "COMPLETED") {
				t.Fatalf("\nExpected: %+v\nGot:      %+v\n", "COMPLETED", batchJob.Events[0].Status)
				return
			}
		}
	})
}
