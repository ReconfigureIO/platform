package afi_watcher

import (
	"context"
	"testing"
	"time"

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/aws"
	"github.com/golang/mock/gomock"
)

type fake_PostgresRepo struct{}

type fake_BatchService struct{}

//create a build that's waiting on an image
func (repo fake_PostgresRepo) GetBuildsWithStatus(statuses []string, limit int) ([]models.Build, error) {
	build := models.Build{
		FPGAImage: models.FPGAImage{
			AFIID: "afi-foobar",
		},
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

	afistatus := make(map[string]string)
	afistatus["afi-foobar"] = "available"
	build, _ := d.GetBuildsWithStatus([]string{"bar"}, 1)

	mockService := aws.NewMockService(mockCtrl)
	mockService.EXPECT().DescribeAFIStatus(context.Background(), build).Return(afistatus, nil)

	err := FindAFI(d, mockService, b)
	if err != nil {
		t.Fatalf("Error in FindAFI function: ", err)
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
