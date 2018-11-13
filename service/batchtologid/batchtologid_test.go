package batchtologid

import (
	"context"
	"testing"
	"time"

	"github.com/ReconfigureIO/platform/models"
	"github.com/aws/aws-sdk-go/service/batch"
	"github.com/golang/mock/gomock"
)

var logName = "foobarLogName"
var batchID = "foobarBatchID"

type fakeAWS struct{}

func (aws *fakeAWS) DescribeJobs(input *batch.DescribeJobsInput) (*batch.DescribeJobsOutput, error) {
	return &batch.DescribeJobsOutput{
		Jobs: []*batch.JobDetail{
			&batch.JobDetail{
				Container: &batch.ContainerDetail{
					LogStreamName: &logName,
				},
			},
		},
	}, nil
}

func TestBatchToLogID(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	batchRepo := models.NewMockBatchRepo(mockCtrl)
	batchRepo.EXPECT().HasStarted(batchID).Return(true, nil)
	batchRepo.EXPECT().GetLogName(batchID).Return("", nil)
	batchRepo.EXPECT().SetLogName(batchID, logName).Return(nil)

	b2l := Adapter{
		BatchRepo:     batchRepo,
		AWS:           &fakeAWS{},
		PollingPeriod: time.Microsecond,
	}

	ctxtimeout, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	returned, err := b2l.Do(ctxtimeout, batchID)
	if err != nil {
		t.Error(err)
	}
	if returned != logName {
		t.Errorf("Returned log name did not match expected value. Returned: %v Expected: %v \n", returned, logName)
	}
}
