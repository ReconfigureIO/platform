package api

import (
	"testing"

	"github.com/ReconfigureIO/platform/service/aws"
	"github.com/golang/mock/gomock"
)

func Test_ServiceInterface(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	s := aws.NewMockServiceInterface(mockCtrl)
	s.EXPECT()
}
