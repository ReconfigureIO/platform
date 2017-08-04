package afi_watcher

import (
	"testing"
	//	"time"

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/aws"
	"github.com/golang/mock/gomock"
)

func TestFindAFI(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	d := models.NewMockBuildRepo(mockCtrl)
	b := models.NewMockBatchRepo(mockCtrl)
	mockService := aws.NewMockService(mockCtrl)

	afistatus := map[string]string{"agfi-foobar": "available"}

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

	d.EXPECT().GetBuildsWithStatus(creating_statuses, 100).Return(builds, nil)
	mockService.EXPECT().DescribeAFIStatus(gomock.Any(), builds).Return(afistatus, nil)
	b.EXPECT().AddEvent(build.BatchJob, gomock.Any()).Return(nil)

	err := FindAFI(d, mockService, b)

	if err != nil {
		t.Error(err)
	}
}

func TestFindAFISkipsInvalidStatus(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	d := models.NewMockBuildRepo(mockCtrl)
	b := models.NewMockBatchRepo(mockCtrl)
	mockService := aws.NewMockService(mockCtrl)

	afistatus := map[string]string{"agfi-foobar": "invalid-status"}

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

	d.EXPECT().GetBuildsWithStatus(creating_statuses, 100).Return(builds, nil)
	mockService.EXPECT().DescribeAFIStatus(gomock.Any(), builds).Return(afistatus, nil)

	err := FindAFI(d, mockService, b)

	if err != nil {
		t.Error(err)
	}
}
