// +build integration

package deployment

import (
	"context"
	"testing"

	"github.com/ReconfigureIO/platform/models"
)

func TestDescribeInstanceStatus(t *testing.T) {
	s := New(ServiceConfig{})
	_, err := s.DescribeInstanceStatus(context.Background(), []models.Deployment{
		models.Deployment{
			ID:           "foo",
			InstanceID:   "sir-px1i434k",
			SpotInstance: false,
		},
	})

	if err != nil {
		t.Error(err)
		return
	}
}
