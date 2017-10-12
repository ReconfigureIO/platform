package api

import (
	"errors"

	"github.com/ReconfigureIO/platform/config"
	"github.com/ReconfigureIO/platform/service/aws"
	"github.com/ReconfigureIO/platform/service/deployment"
	"github.com/ReconfigureIO/platform/service/queue"
	"github.com/ReconfigureIO/platform/sugar"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

const (
	maxConcurrentJobs = 2
)

var (
	errNotFound = errors.New("Not Found")

	db *gorm.DB

	awsSession aws.Service

	deploy deployment.Service

	deploymentQueue queue.Queue
)

// DB sets the database to use for the API.
func DB(d *gorm.DB) {
	db = d
}

func Configure(conf config.Config) {
	awsSession = aws.New(conf.Reco.AWS)

	deploy = deployment.New(conf.Reco.Deploy)
}

// DepQueue sets the deployment queue.
func DepQueue(q queue.Queue) {
	deploymentQueue = q
}

// Transaction runs a transaction, rolling back if error != nil.
func Transaction(c *gin.Context, ops func(db *gorm.DB) error) error {
	tx := db.Begin()
	err := ops(tx)
	if err != nil {
		tx.Rollback()
		c.Error(err)
		sugar.ErrResponse(c, 500, nil)
	} else {
		tx.Commit()
	}
	return err
}

func bindID(c *gin.Context, id *string) bool {
	paramID := c.Param("id")
	if paramID != "" {
		*id = paramID
		return true
	}
	sugar.ErrResponse(c, 404, nil)
	return false
}
