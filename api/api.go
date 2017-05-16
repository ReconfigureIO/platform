package api

import (
	"errors"
	"strconv"

	"github.com/ReconfigureIO/platform/service/aws"
	"github.com/ReconfigureIO/platform/service/mock_deployment"
	. "github.com/ReconfigureIO/platform/sugar"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

var (
	errNotFound = errors.New("Not Found")

	db *gorm.DB

	awsSession = aws.New(aws.ServiceConfig{
		LogGroup:      "/aws/batch/job",
		Bucket:        "reconfigureio-builds",
		Queue:         "build-jobs",
		JobDefinition: "sdaccel-builder-build",
	})

	mockDeploy = mock_deployment.New(mock_deployment.ServiceConfig{
		LogGroup: "josh-test-sdaccel",
		Image:    "398048034572.dkr.ecr.us-east-1.amazonaws.com/reconfigureio/platform/deployment:latest",
		AMI:      "ami-850c7293",
	})
)

// DB sets the database to use for the API.
func DB(d *gorm.DB) {
	db = d
}

// Run a transaction, rolling back if error != nil
func Transaction(c *gin.Context, ops func(db *gorm.DB) error) error {
	tx := db.Begin()
	err := ops(tx)
	if err != nil {
		tx.Rollback()
		c.Error(err)
		ErrResponse(c, 500, nil)
	} else {
		tx.Commit()
	}
	return err
}

func bindId(c *gin.Context, id *int) bool {
	paramId := c.Param("id")
	if i, err := strconv.Atoi(paramId); err == nil && paramId != "" {
		*id = i
		return true
	}
	ErrResponse(c, 404, nil)
	return false
}
