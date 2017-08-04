package models

//go:generate mockgen -source=batch.go -package=models -destination=batch_mock.go

import (
	"time"

	"github.com/jinzhu/gorm"
)

type BatchRepo interface {
	AddEvent(batchJob *BatchJob, event BatchJobEvent) error
	New(batchID string) BatchJob
}

type batchRepo struct{ db *gorm.DB }

func BatchDataSource(db *gorm.DB) BatchRepo {
	b := batchRepo{db: db}
	return &b
}

// New creates a new batch job with its queued event.
func (repo *batchRepo) New(batchID string) BatchJob {
	event := BatchJobEvent{Timestamp: time.Now(), Status: "QUEUED"}
	batchJob := BatchJob{BatchID: batchID, Events: []BatchJobEvent{event}}
	return batchJob
}

// AddEvent adds an event to the batch service.
func (repo *batchRepo) AddEvent(batchJob *BatchJob, event BatchJobEvent) error {
	db := repo.db
	err := db.Model(batchJob).Association("Events").Append(event).Error
	return err
}
