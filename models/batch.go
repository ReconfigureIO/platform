package models

//go:generate mockgen -source=batch.go -package=models -destination=batch_mock.go

import (
	"context"
	"time"

	"github.com/jinzhu/gorm"
)

type BatchRepo interface {
	AddEvent(batchJob BatchJob, event BatchJobEvent) error
	New(batchID string) BatchJob
	GetLogName(batchID string) (logName string, err error)
	SetLogName(id string, logName string) error
	ActiveJobsWithoutLogs(time.Time) ([]BatchJob, error)
	HasStarted(batchID string) (started bool, err error)
}

const (
	sqlBatchJobsWithoutLogs = `
select j.id as id
from batch_jobs j
left join batch_job_events started
on j.id = started.batch_job_id
    and started.id = (
        select e1.id
        from batch_job_events e1
        where j.id = e1.batch_job_id and e1.status = 'STARTED'
        limit 1
    )
where (log_name = '' and started.timestamp > ?)
`
)

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

// AwaitStarted polls the BatchRepo's DB for the state of the batch job
// associated with a given ID. It blocks until the batch job has started, unless
// an error occurs.
func BatchAwaitStarted(ctx context.Context, repo BatchRepo, batchID string, pollPeriod time.Duration) error {
	for {
		select {
		case <-time.After(pollPeriod):
			started, err := repo.HasStarted(batchID)
			if err != nil {
				return err
			}
			if started {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// HasStarted returns if the build has started.
func (repo *batchRepo) HasStarted(batchID string) (bool, error) {
	var batchJob BatchJob
	err := repo.db.Preload("Events").Where("batch_id = ?", batchID).First(&batchJob).Error
	if err != nil {
		return false, err
	}
	return hasStarted(batchJob.Status()), nil
}

// AddEvent adds an event to the batch service.
func (repo *batchRepo) AddEvent(batchJob BatchJob, event BatchJobEvent) error {
	db := repo.db
	err := db.Model(&batchJob).Association("Events").Append(event).Error
	return err
}

// GetLogName takes a BatchJob ID and returns that BatchJob's logname if present
func (repo *batchRepo) GetLogName(id string) (string, error) {
	var batchJob BatchJob
	err := repo.db.Where("batch_id = ?", id).First(&batchJob).Error
	if err != nil {
		return "", err
	}
	return batchJob.LogName, nil
}

func (repo *batchRepo) SetLogName(id string, logName string) error {
	batchJob := BatchJob{}
	err := repo.db.Where("batch_id = ?", id).First(&batchJob).Error
	if err != nil {
		return err
	}
	err = repo.db.Model(&batchJob).Update("log_name", logName).Error
	return err
}

func (repo *batchRepo) ActiveJobsWithoutLogs(sinceTime time.Time) ([]BatchJob, error) {
	db := repo.db
	rows, err := db.Raw(sqlBatchJobsWithoutLogs, sinceTime).Rows()
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

	var batchJobs []BatchJob
	err = db.Where("id in (?)", ids).Find(&batchJobs).Error
	if err != nil {
		return nil, err
	}

	return batchJobs, nil
}
