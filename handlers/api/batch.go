package api

//go:generate mockgen -source=batch.go -package=api -destination=batch_mock.go

import (
	"time"

	"github.com/ReconfigureIO/platform/models"
)

// BatchService is aws batch job service.
type BatchService struct{}

type BatchInterface interface {
	AddEvent(batchJob *models.BatchJob, event models.PostBatchEvent) (models.BatchJobEvent, error)
}

// New creates a new batch job with its queued event.
func (b BatchService) New(batchID string) models.BatchJob {
	event := models.BatchJobEvent{Timestamp: time.Now(), Status: "QUEUED"}
	batchJob := models.BatchJob{BatchID: batchID, Events: []models.BatchJobEvent{event}}
	return batchJob
}

// AddEvent adds an event to the batch service.
func (b BatchService) AddEvent(batchJob *models.BatchJob, event models.PostBatchEvent) (models.BatchJobEvent, error) {
	timestamp := time.Time{}
	if event.Timestamp.IsZero() == false {
		timestamp = event.Timestamp
	} else {
		timestamp = time.Now()
	}

	newEvent := models.BatchJobEvent{
		Timestamp: timestamp,
		Status:    event.Status,
		Message:   event.Message,
		Code:      event.Code,
	}
	err := db.Model(batchJob).Association("Events").Append(newEvent).Error
	if err != nil {
		return models.BatchJobEvent{}, nil
	}

	if newEvent.Status == models.StatusTerminated {
		err = awsSession.HaltJob(batchJob.BatchID)
		if err != nil {
			return models.BatchJobEvent{}, nil
		}
	}
	return newEvent, nil
}
