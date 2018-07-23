// +build integration

package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ReconfigureIO/platform/models"
	awsaws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	awsbatch "github.com/aws/aws-sdk-go/service/batch"
	"github.com/aws/aws-sdk-go/service/batch/batchiface"
	"github.com/golang/mock/gomock"
)

func TestFindLogNames(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	repo := models.NewMockBatchRepo(mockCtrl)

	watcher := NewLogWatcher(repo, fakeBatchClient{})

	batchJobs := []models.BatchJob{
		models.BatchJob{
			ID:      123456789,
			BatchID: "foobar",
			Events: []models.BatchJobEvent{
				models.BatchJobEvent{
					Timestamp: time.Unix(20, 0),
					Status:    "STARTED",
				},
				models.BatchJobEvent{
					Timestamp: time.Unix(0, 0),
					Status:    "QUEUED",
				},
			},
		},
	}
	LogNames := map[string]string{batchJobs[0].BatchID: "LogName"}

	ctx := context.Background()
	limit := 100
	sinceTime := time.Unix(0, 0)

	repo.EXPECT().ActiveJobsWithoutLogs(sinceTime).Return(batchJobs, nil)
	repo.EXPECT().SetLogName(batchJobs[0].BatchID, LogNames[batchJobs[0].BatchID]).Return(nil)

	err := watcher.FindLogNames(ctx, limit, sinceTime)

	if err != nil {
		t.Error(err)
	}
}

type fakeBatchClient struct {
	batchiface.BatchAPI
}

func (
	batch fakeBatchClient,
) DescribeJobsWithContext(
	ctx awsaws.Context,
	req *awsbatch.DescribeJobsInput,
	opts ...request.Option,
) (
	*awsbatch.DescribeJobsOutput,
	error,
) {
	return &awsbatch.DescribeJobsOutput{
		Jobs: []*awsbatch.JobDetail{
			&awsbatch.JobDetail{
				JobId: awsaws.String("foobar"),
				Container: &awsbatch.ContainerDetail{
					LogStreamName: awsaws.String("LogName"),
				},
			},
		},
	}, nil
}
