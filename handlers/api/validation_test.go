package api

import (
	"testing"

	"github.com/ReconfigureIO/platform/models"
	validator "gopkg.in/validator.v2"
)

func TestProjValidation(t *testing.T) {
	newPProj := PostProject{
		Name: "foobar",
	}

	err := validator.Validate(newPProj)
	if err != nil {
		t.Error()
	}

}

func TestDepJobEventValidation(t *testing.T) {
	newDepJobEvent := models.DeploymentEvent{
		DeploymentID: "1",
	}

	err := validator.Validate(newDepJobEvent)
	if err != nil {
		t.Error()
	}

}
