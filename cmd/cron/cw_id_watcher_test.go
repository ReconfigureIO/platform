// +build integration

package main

import (
	"context"
	"testing"
	"time"

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/batch/aws"
	"github.com/golang/mock/gomock"
)

func TestFindLogNames(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	b := models.NewMockBatchRepo(mockCtrl)
	a := aws.New(aws.ServiceConfig{}, aws.NewMockStreamService(mockCtrl))

	watcher := NewLogWatcher(*a, b)

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

	b.EXPECT().ActiveJobsWithoutLogs(sinceTime).Return(batchJobs, nil)
	b.EXPECT().SetLogName(batchJobs[0].BatchID, LogNames[batchJobs[0].BatchID]).Return(nil)

	err := watcher.FindLogNames(ctx, limit, sinceTime)

	if err != nil {
		t.Error(err)
	}
}
