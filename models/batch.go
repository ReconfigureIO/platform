package models

//go:generate mockgen -source=batch.go -package=models -destination=batch_mock.go

import (
	"time"
)

type BatchRepo interface {
	AddEvent(batchJob *BatchJob, event PostBatchEvent) (BatchJobEvent, error)
}

type batchRepo struct{ db *gorm.DB }

func BatchDataSource(db *gorm.DB) BatchRepo {
	return &batchRepo{db: db}
}

// New creates a new batch job with its queued event.
func (b BatchRepo) New(batchID string) BatchJob {
	event := BatchJobEvent{Timestamp: time.Now(), Status: "QUEUED"}
	batchJob := BatchJob{BatchID: batchID, Events: []BatchJobEvent{event}}
	return batchJob
}

// AddEvent adds an event to the batch service.
func (b BatchRepo) AddEvent(batchJob *BatchJob, event BatchJobEvent) error {
	db := repo.db
	err = db.Model(batchJob).Association("Events").Append(event).Error
	if err != nil {
		return err
	}

	if event.Status == StatusTerminated {
		err = awsSession.HaltJob(batchJob.BatchID)
		if err != nil {
			return err
		}
	}
	return nil
}
