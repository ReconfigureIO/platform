package api

import (
	"errors"

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

	deploymentQueue queue.Queue
)

// DB sets the database to use for the API.
func DB(d *gorm.DB) {
	db = d
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
		sugar.InternalError(c, nil)
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
