package api

import (
	"time"

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/batch/aws"
)

// BatchService is aws batch job service.
type BatchService struct {
	AWS aws.Service
}

// New creates a new batch job with its queued event.
func (b BatchService) New(batchID string) models.BatchJob {
	return models.BatchDataSource(db).New(batchID)
}

// AddEvent adds an event to the batch service.
func (b BatchService) AddEvent(batchJob *models.BatchJob, event models.PostBatchEvent) (models.BatchJobEvent, error) {
	newEvent := models.BatchJobEvent{
		Timestamp: time.Now(),
		Status:    event.Status,
		Message:   event.Message,
		Code:      event.Code,
	}

	repo := models.BatchDataSource(db)
	err := repo.AddEvent(*batchJob, newEvent)

	if err != nil {
		return models.BatchJobEvent{}, err
	}

	if newEvent.Status == models.StatusTerminating {
		err = b.AWS.HaltJob(batchJob.BatchID)
		if err != nil {
			return models.BatchJobEvent{}, err
		}

		terminated := models.BatchJobEvent{
			Timestamp: time.Now(),
			Status:    models.StatusTerminated,
			Message:   event.Message,
			Code:      event.Code,
		}
		_ = repo.AddEvent(*batchJob, terminated)
	}
	return newEvent, nil
}
