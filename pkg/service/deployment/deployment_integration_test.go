// +build integration

package deployment

import (
	"context"
	"testing"

	"github.com/ReconfigureIO/platform/pkg/models"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/caarlos0/env"
)

func DryRunOk(err error) bool {
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "DryRunOperation":
				return true
			default:
				return false
			}
		}
	}
	return false
}

const (
	ENCODED_CONFIG = "ewogICJjb250YWluZXIiOiB7CiAgICAiaW1hZ2UiOiAicmVjb25maWd1cmVpby9kb2NrZXItYXdzLWZwZ2EtcnVudGltZTpsYXRlc3QiLAogICAgImNvbW1hbmQiOiAiYmVuY2gtaGlzdG9ncmFtIgogIH0sCiAgImxvZ3MiOiB7CiAgICAiZ3JvdXAiOiAiam9zaC10ZXN0LXNkYWNjZWwiLAogICAgInByZWZpeCI6ICJkZXBsb3ltZW50LTEiCiAgfSwKICAiY2FsbGJhY2tfdXJsIjogIiIsCiAgImJ1aWxkIjogewogICAgICAiYXJ0aWZhY3RfdXJsIjogInMzOi8vcmVjb25maWd1cmVpby1idWlsZHMvdG1wL2U5MWQ3Yjc4LTczMGQtMTFlNy1iMjdmLTEyN2Y1ZTNhZjkyOC5kaXN0LnppcCIsCiAgICAgICJhZ2ZpIjogImFnZmktMGUzZDViNzE3NTlhMmRhMTAiCiAgfQp9Cg=="
)

func TestDeploySpotInstance(t *testing.T) {
	c := ServiceConfig{}
	err := env.Parse(&c)
	if err != nil {
		t.Error(err)
	}

	d := newService(c)
	_, err = d.runSpotInstance(context.Background(), ENCODED_CONFIG, true)
	if !DryRunOk(err) {
		t.Error(err)
	}
}

func TestDeployInstance(t *testing.T) {
	c := ServiceConfig{}
	err := env.Parse(&c)
	if err != nil {
		t.Error(err)
	}

	d := newService(c)
	_, err = d.runInstance(context.Background(), ENCODED_CONFIG, true)
	if !DryRunOk(err) {
		t.Error(err)
	}
}

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
