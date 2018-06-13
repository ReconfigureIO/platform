package api

import (
	"testing"

	"github.com/ReconfigureIO/platform/pkg/service/aws"
	"github.com/golang/mock/gomock"
)

func Test_ServiceInterface(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	s := aws.NewMockService(mockCtrl)
	s.EXPECT().RunSimulation("foo", "bar", "test").Return("foobar", nil)
	ss, err := s.RunSimulation("foo", "bar", "test")
	if err != nil || ss != "foobar" {
		t.Error("unexpected result")
	}
}
