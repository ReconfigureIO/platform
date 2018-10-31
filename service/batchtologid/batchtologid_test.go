package batchtologid

import (
	"testing"

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

func TestBidToLidAwait(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	batchRepo := models.NewMockBatchRepo(mockCtrl)
	batchRepo.EXPECT().AwaitStarted(batchID).Return(nil)
	batchRepo.EXPECT().GetLogName(batchID).Return("", nil)
	batchRepo.EXPECT().SetLogName(batchID, logName).Return(nil)

	b2l := Adapter{
		batchRepo: batchRepo,
		aws:       &fakeAWS{},
	}

	returned, err := b2l.bidToLid(batchID)
	if err != nil {
		t.Error(err)
	}
	if returned != logName {
		t.Errorf("Returned log name did not expected value. Returned: %v Expected: %v \n", returned, logName)
	}
}
