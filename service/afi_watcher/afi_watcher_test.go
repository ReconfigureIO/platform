package afi_watcher

import (
	"context"
	"testing"
	"time"

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/aws"
	"github.com/golang/mock/gomock"
)

func TestFindAFI(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	buildRepo := models.NewMockBuildRepo(mockCtrl)
	batchRepo := models.NewMockBatchRepo(mockCtrl)
	describeAFIStatuser := aws.NewMockService(mockCtrl)

	watcher := NewAFIWatcher(buildRepo, describeAFIStatuser, batchRepo)

	// the time.Now we return as part of afistatus comes back as
	// part of the call to AddEvent
	timeNow := time.Now()
	afistatus := map[string]aws.Status{"agfi-foobar": aws.Status{"available", timeNow}}

	build := models.Build{
		FPGAImage: "agfi-foobar",
		BatchJob: models.BatchJob{
			Events: []models.BatchJobEvent{
				models.BatchJobEvent{
					Timestamp: timeNow,
					Status:    models.StatusCreatingImage,
					Message:   "afi-foobar",
				},
			},
		},
	}
	builds := []models.Build{build}
	ctx := context.Background()
	limit := 100

	event := &models.BatchJobEvent{
		Timestamp: timeNow,
		Status:    models.StatusCompleted,
		Message:   "Image generation succeeded",
		Code:      0,
	}

	buildRepo.EXPECT().GetBuildsWithStatus(creating_statuses, limit).Return(builds, nil)
	describeAFIStatuser.EXPECT().DescribeAFIStatus(ctx, builds).Return(afistatus, nil)
	batchRepo.EXPECT().AddEvent(build.BatchJob, *event).Return(nil)

	err := watcher.FindAFI(ctx, limit)

	if err != nil {
		t.Error(err)
	}
}

func TestFindAFISkipsInvalidStatus(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	buildRepo := models.NewMockBuildRepo(mockCtrl)
	batchRepo := models.NewMockBatchRepo(mockCtrl)
	describeAFIStatuser := aws.NewMockService(mockCtrl)

	watcher := NewAFIWatcher(buildRepo, describeAFIStatuser, batchRepo)

	afistatus := map[string]aws.Status{"agfi-foobar": aws.Status{"invalid-status", time.Now()}}

	build := models.Build{
		FPGAImage: "agfi-foobar",
		BatchJob: models.BatchJob{
			Events: []models.BatchJobEvent{
				models.BatchJobEvent{
					Status:  "CREATING_IMAGE",
					Message: "afi-foobar",
				},
			},
		},
	}
	builds := []models.Build{build}
	ctx := context.Background()
	limit := 100

	buildRepo.EXPECT().GetBuildsWithStatus(creating_statuses, limit).Return(builds, nil)
	describeAFIStatuser.EXPECT().DescribeAFIStatus(ctx, builds).Return(afistatus, nil)

	err := watcher.FindAFI(ctx, limit)

	if err != nil {
		t.Error(err)
	}
}
