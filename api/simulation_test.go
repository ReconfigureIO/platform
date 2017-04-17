package api

import (
	"github.com/ReconfigureIO/platform/service/aws"
	"github.com/golang/mock/gomock"
	"testing"
)

func Test_ServiceInterface(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	s := aws.NewMockServiceInterface(mockCtrl)
	s.EXPECT().RunBuild("foo", "bar")
}
