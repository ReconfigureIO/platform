package cw_id_watcher

import (
	"context"

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/aws"
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

func (watcher *LogWatcher) FindLogNames(ctx context.Context, limit int) error {
	batchJobIDs, err := watcher.awsService.ListBatchJobs(ctx, limit)
	if err != nil {
		log.WithError(err).Error("Couldn't list batch jobs")
		return err
	}

	if len(batchJobIDs) == 0 {
		return nil
	}

	cwLogNames, err := watcher.awsService.GetCwLogNames(ctx, batchJobIDs)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{"batch_job_ids": batchJobIDs}).Error("Couldn't get cw log names for batch jobs")
		return err
	}

	for _, jobID := range batchJobIDs {
		logName, found := cwLogNames[jobID]
		if found {
			err := watcher.batch.SetCwLogName(jobID, logName)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
