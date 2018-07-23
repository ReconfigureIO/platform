package main

import (
	"context"
	"time"

	"github.com/ReconfigureIO/platform/models"
	awsaws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/batch"
	"github.com/aws/aws-sdk-go/service/batch/batchiface"
	log "github.com/sirupsen/logrus"
)

type LogWatcher struct {
	batchRepo    models.BatchRepo
	batchSession batchiface.BatchAPI
}

func NewLogWatcher(batchRepo models.BatchRepo, batchSession batchiface.BatchAPI) *LogWatcher {
	w := LogWatcher{
		batchRepo:    batchRepo,
		batchSession: batchSession,
	}
	return &w
}

func (watcher *LogWatcher) FindLogNames(ctx context.Context, limit int, sinceTime time.Time) error {
	batchJobs, err := watcher.batchRepo.ActiveJobsWithoutLogs(sinceTime)

	if len(batchJobs) == 0 {
		return nil
	}

	batchJobIDs := []string{}
	for _, returnedBatchJob := range batchJobs {
		batchJobIDs = append(batchJobIDs, returnedBatchJob.BatchID)
	}

	LogNames, err := watcher.getLogNames(ctx, batchJobIDs)
	if err != nil {
		log.WithError(err).
			WithFields(log.Fields{"batch_job_ids": batchJobIDs}).
			Error("Couldn't get cw log names for batch jobs")
		return err
	}

	for _, jobID := range batchJobIDs {
		logName, found := LogNames[jobID]
		if found {
			err := watcher.batchRepo.SetLogName(jobID, logName)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (watcher *LogWatcher) getLogNames(ctx context.Context, batchJobIDs []string) (map[string]string, error) {
	ret := make(map[string]string)

	cfg := batch.DescribeJobsInput{
		Jobs: awsaws.StringSlice(batchJobIDs),
	}

	results, err := watcher.batchSession.DescribeJobsWithContext(ctx, &cfg)
	if err != nil {
		return ret, err
	}

	for _, job := range results.Jobs {
		ret[*job.JobId] = *job.Container.LogStreamName
	}

	return ret, nil
}
