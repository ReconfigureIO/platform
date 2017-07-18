package afi_watcher

import (
	"testing"

	"github.com/ReconfigureIO/platform/models"
	"github.com/golang/mock/gomock"
	"github.com/jinzhu/gorm"
)

type fake_PostgresRepo struct {
	DB *gorm.DB
}

type fake_BatchService struct{}

//create a build that's waiting on an image
func (repo *fake_PostgresRepo) GetBuildsWithStatus(statuses []string, limit int) ([]models.Build, error) {
	build := models.Build{
			Command: "",
			FPGAImage{
				AFIID: "afi-foobar"
			}
			BatchJob: BatchJob{
				BatchID: "Bar",
				Events: []BatchJobEvent{
					BatchJobEvent{
						BatchJobID: "Bar",
						Status:   "CREATING_IMAGE",
						Message:  "afi-foobar"
					},
				},
			},
		}
	repo.Create(&build) 
	return build, nil
}

func TestFindAfi(t *testing.T) {
	d := fake_BuildRepo{}

	err := FindAfi(d)
	if err != nil {
		t.Fatalf("bork bork", err)
	}
}


func Test_AFI_Watcher_DescribeAFIStatus(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	afistatus = make(map[string]string)
	afistatus["afi-foobar"] = "available"

	mockService := NewMockService(mockCtrl)
	mockService.EXPECT().DescribeAFIStatus.Return(afistatus, nil)
	
	// if err != nil {
	// 	t.Error("Unexpected error returned", err)
	// }
	// if str[0] != "Buzz" {
	// 	t.Error("Expected returned value to be Buzz", str)
	// }
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
