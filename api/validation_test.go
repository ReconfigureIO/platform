package api

import (
	"testing"

	"github.com/ReconfigureIO/platform/models"
	validator "gopkg.in/validator.v2"
)

func TestProjValidation(t *testing.T) {
	newProj := models.Project{
		Name:   "",
		UserID: 1,
	}

	err := validator.Validate(newProj)
	if err != nil {
		t.Error()
	}

}

func TestDepJobEventValidation(t *testing.T) {
	newDepJobEvent := models.DepJobEvent{
		DepJobId: 0,
	}

	err = validator.Validate(newDepJobEvent)
	if err != nil {
		t.Error()
	}

}
