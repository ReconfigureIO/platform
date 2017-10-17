package models

//go:generate mockgen -source=batch.go -package=models -destination=batch_mock.go

import (
	"time"

	"github.com/jinzhu/gorm"
)

type BatchRepo interface {
	AddEvent(batchJob BatchJob, event BatchJobEvent) error
	New(batchID string) BatchJob
}

type batchRepo struct{ db *gorm.DB }

func BatchDataSource(db *gorm.DB) BatchRepo {
	b := batchRepo{db: db}
	return &b
}

const (
	SQL_BATCH_STATUS = `SELECT j.id
FROM batch_jobs j
LEFT join batch_job_events e
ON j.id = e.batch_job_id
    AND e.timestamp = (
        SELECT max(timestamp)
        FROM batch_job_events e1
        WHERE j.id = e1.batch_job_id
    )
WHERE (e.status in (?))
LIMIT ?
`
)

// New creates a new batch job with its queued event.
func (repo *batchRepo) New(batchID string) BatchJob {
	event := BatchJobEvent{Timestamp: time.Now(), Status: "QUEUED"}
	batchJob := BatchJob{BatchID: batchID, Events: []BatchJobEvent{event}}
	return batchJob
}

// AddEvent adds an event to the batch service.
func (repo *batchRepo) AddEvent(batchJob BatchJob, event BatchJobEvent) error {
	db := repo.db
	err := db.Model(&batchJob).Association("Events").Append(event).Error
	return err
}

func (repo *batchRepo) GetJobsWithStatus(statuses []string, limit int) ([]BatchJob, error) {
	db := repo.db
	rows, err := db.Raw(SQL_BATCH_STATUS, statuses, limit).Rows()
	if err != nil {
		return nil, err
	}

	ids := []string{}
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}
	rows.Close()

	var jobs []BatchJob
	err = db.Preload("BatchJob").Preload("BatchJob.Events").Where("id in (?)", ids).Find(&jobs).Error
	if err != nil {
		return nil, err
	}

	return jobs, nil
}
