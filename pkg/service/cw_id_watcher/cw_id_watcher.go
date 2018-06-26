package cw_id_watcher

import (
	"context"
	"time"

	"github.com/ReconfigureIO/platform/pkg/models"
	"github.com/ReconfigureIO/platform/pkg/service/aws"
	log "github.com/sirupsen/logrus"
)

type LogWatcher struct {
	awsService aws.Service
	batch      models.BatchRepo
}

func NewLogWatcher(awsService aws.Service, batch models.BatchRepo) *LogWatcher {
	w := LogWatcher{
		awsService: awsService,
		batch:      batch,
	}
	return &w
}

func (watcher *LogWatcher) FindLogNames(ctx context.Context, limit int, sinceTime time.Time) error {
	batchJobs, err := watcher.batch.ActiveJobsWithoutLogs(sinceTime)

	if len(batchJobs) == 0 {
		return nil
	}

	batchJobIDs := []string{}
	for _, returnedBatchJob := range batchJobs {
		batchJobIDs = append(batchJobIDs, returnedBatchJob.BatchID)
	}

	LogNames, err := watcher.awsService.GetLogNames(ctx, batchJobIDs)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{"batch_job_ids": batchJobIDs}).Error("Couldn't get cw log names for batch jobs")
		return err
	}

	for _, jobID := range batchJobIDs {
		logName, found := LogNames[jobID]
		if found {
			err := watcher.batch.SetLogName(jobID, logName)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
