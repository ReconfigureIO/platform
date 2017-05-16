package api

import (
	"testing"

	"github.com/ReconfigureIO/platform/service/aws"
	"github.com/golang/mock/gomock"
)

func Test_ServiceInterface(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	s := aws.NewMockService(mockCtrl)
	s.EXPECT().RunBuild("foo", "bar").Return("foobar", nil)
	ss, err := s.RunBuild("foo", "bar")
	if err != nil || ss != "foobar" {
		t.Error("unexpected result")
	}
}
