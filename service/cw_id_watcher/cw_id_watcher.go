package cw_id_watcher

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/batch"
	log "github.com/sirupsen/logrus"

	"github.com/ReconfigureIO/platform/models"
)

type awsBatchIface interface {
	DescribeJobsWithContext(
		aws.Context,
		*batch.DescribeJobsInput,
		...request.Option,
	) (
		*batch.DescribeJobsOutput,
		error,
	)
}

// LogWatcher associates CloudWatchLogs log names with our BatchRepo model.
type LogWatcher struct {
	BatchAPI  awsBatchIface
	BatchRepo models.BatchRepo
}

// FindLogNames associates CloudWatchLogs log names with our BatchRepo model for
// an AWS Batch job, by interrogating AWS Batch. This is necessary because AWS
// batch jobs do not last forever, but our jobs contained within the batch model
// do.
func (watcher *LogWatcher) FindLogNames(
	ctx context.Context,
	limit int,
	sinceTime time.Time,
) error {
	batchJobs, err := watcher.BatchRepo.ActiveJobsWithoutLogs(sinceTime)
	if err != nil {
		return err
	}
	if len(batchJobs) == 0 {
		return nil
	}

	batchJobIDs := []string{}
	for _, returnedBatchJob := range batchJobs {
		batchJobIDs = append(batchJobIDs, returnedBatchJob.BatchID)
	}

	jobToLogName, err := watcher.GetLogNames(ctx, batchJobIDs)
	if err != nil {
		log.WithError(err).
			WithFields(log.Fields{"batch_job_ids": batchJobIDs}).
			Error("Couldn't get cw log names for batch jobs")
		return err
	}

	for _, jobID := range batchJobIDs {
		logName, found := jobToLogName[jobID]
		if found {
			err := watcher.BatchRepo.SetLogName(jobID, logName)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// GetLogNames takes a list of batchJobIDs and returns the log names
// corresponding to those batch IDs, as reported from the AWS batch API. This
// function is exported so that it may be mocked.
func (watcher *LogWatcher) GetLogNames(
	ctx context.Context,
	batchJobIDs []string,
) (map[string]string, error) {
	jobToLogName := make(map[string]string)

	results, err := watcher.BatchAPI.DescribeJobsWithContext(
		ctx, &batch.DescribeJobsInput{
			Jobs: aws.StringSlice(batchJobIDs),
		})
	if err != nil {
		return nil, err
	}

	for _, job := range results.Jobs {
		jobToLogName[*job.JobId] = *job.Container.LogStreamName
	}

	return jobToLogName, nil
}
