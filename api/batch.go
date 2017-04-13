package api

import (
	"github.com/ReconfigureIO/platform/models"
	"time"
)

type BatchService struct{}

// Create a new batch job with its queued event
func (b BatchService) New(batchId string) models.BatchJob {
	event := models.BatchJobEvent{Timestamp: time.Now(), Status: "QUEUED"}
	batchJob := models.BatchJob{BatchId: batchId, Events: []models.BatchJobEvent{event}}
	return batchJob
}

func (b BatchService) AddEvent(batchJob *models.BatchJob, event models.PostBatchEvent) (models.BatchJobEvent, error) {
	newEvent := models.BatchJobEvent{
		Timestamp: time.Now(),
		Status:    event.Status,
		Message:   event.Message,
		Code:      event.Code,
	}
	err := db.Model(batchJob).Association("Events").Append(newEvent).Error
	if err != nil {
		return models.BatchJobEvent{}, nil
	}

	if newEvent.Status == models.TERMINATED {
		err = awsSession.HaltJob(batchJob.BatchId)
		if err != nil {
			return models.BatchJobEvent{}, nil
		}
	}
	return newEvent, nil
}
