package afiwatcher

import (
	"context"
	"testing"
	"time"

	"github.com/ReconfigureIO/platform/pkg/models"
	"github.com/ReconfigureIO/platform/service/fpgaimage"
	"github.com/golang/mock/gomock"
)

func TestFindAFI(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	buildRepo := models.NewMockBuildRepo(mockCtrl)
	batchRepo := models.NewMockBatchRepo(mockCtrl)
	fpgaImageService := fpgaimage.NewMockService(mockCtrl)

	watcher := AFIWatcher{
		BatchRepo:        batchRepo,
		BuildRepo:        buildRepo,
		FPGAImageService: fpgaImageService,
	}

	// The time.Now we return as part of afiStatus comes back as part of the
	// call to AddEvent.
	timeNow := time.Now()

	afiStatus := map[string]fpgaimage.Status{
		"agfi-foobar": {
			Status:    "available",
			UpdatedAt: timeNow,
		},
	}

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

	buildRepo.EXPECT().GetBuildsWithStatus(statusCreating, limit).Return(builds, nil)
	fpgaImageService.EXPECT().DescribeAFIStatus(ctx, builds).Return(afiStatus, nil)
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
	fpgaImageService := fpgaimage.NewMockService(mockCtrl)

	watcher := AFIWatcher{
		BatchRepo:        batchRepo,
		BuildRepo:        buildRepo,
		FPGAImageService: fpgaImageService,
	}

	afiStatus := map[string]fpgaimage.Status{
		"agfi-foobar": {
			Status:    "invalid-status",
			UpdatedAt: time.Now(),
		},
	}

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

	buildRepo.EXPECT().GetBuildsWithStatus(statusCreating, limit).Return(builds, nil)
	fpgaImageService.EXPECT().DescribeAFIStatus(ctx, builds).Return(afiStatus, nil)

	err := watcher.FindAFI(ctx, limit)

	if err != nil {
		t.Error(err)
	}
}
