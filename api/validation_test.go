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
	newDepJobEvent := models.DepJobEvent{
		DepJobID: 1,
	}

	err := validator.Validate(newDepJobEvent)
	if err != nil {
		t.Error()
	}

}

func TestDepJobValidation(t *testing.T) {
	newDepJob := models.DepJob{
		DepID: "1",
	}

	err := validator.Validate(newDepJob)
	if err != nil {
		t.Error()
	}

}
