package afi_watcher

import (
	"testing"
	"time"

	"github.com/ReconfigureIO/platform/handlers/api"
	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/aws"
	"github.com/golang/mock/gomock"
)

type fake_PostgresRepo struct{}

type fake_BatchService struct{}

//create a build that's waiting on an image
func (repo fake_PostgresRepo) GetBuildsWithStatus(statuses []string, limit int) ([]models.Build, error) {
	build := models.Build{
		FPGAImage: "afi-foobar",
		BatchJob: models.BatchJob{
			Events: []models.BatchJobEvent{
				models.BatchJobEvent{
					Status:  "CREATING_IMAGE",
					Message: "afi-foobar",
				},
			},
		},
	}
	return []models.Build{build}, nil
}

func TestFindAFI(t *testing.T) {
	d := fake_PostgresRepo{}
	b := fake_BatchService{}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	afistatus := map[string]string{"afi-foobar": "available"}

	mockService := aws.NewMockService(mockCtrl)
	mockService.EXPECT().DescribeAFIStatus(gomock.Any(), gomock.Any()).Return(afistatus, nil)

	err := FindAFI(d, mockService, b)
	if err != nil {
		t.Fatalf("Error in FindAFI function: %s", err)
	}
}

func TestFindAFISkipsInvalidStatus(t *testing.T) {
	d := fake_PostgresRepo{}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	afistatus := map[string]string{"afi-foobar": "invalid-status"}

	mockService := aws.NewMockService(mockCtrl)
	mockService.EXPECT().DescribeAFIStatus(gomock.Any(), gomock.Any()).Return(afistatus, nil)

	// Don't setup any expected calls, since we don't expect this to be called
	mockBatch := api.NewMockBatchInterface(mockCtrl)

	err := FindAFI(d, mockService, mockBatch)
	if err != nil {
		t.Fatalf("Error in FindAFI function: %s", err)
	}
}

func (b fake_BatchService) AddEvent(batchJob *models.BatchJob, event models.PostBatchEvent) (models.BatchJobEvent, error) {
	newEvent := models.BatchJobEvent{
		Timestamp: time.Now(),
		Status:    event.Status,
		Message:   event.Message,
		Code:      event.Code,
	}

	return newEvent, nil
}
