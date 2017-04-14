package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ReconfigureIO/platform/service/aws"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	validator "gopkg.in/validator.v2"
)

var (
	errNotFound = errors.New("Not Found")

	db *gorm.DB

	awsSession = aws.New(aws.ServiceConfig{
		Bucket:        "reconfigureio-builds",
		Queue:         "build-jobs",
		JobDefinition: "sdaccel-builder-build",
	})
)

type (
	// M is a convenience wrapper for a map.
	M map[string]interface{}

	apiError struct {
		Error string `json:"error"`
	}

	apiSuccess struct {
		Value interface{} `json:"value"`
	}
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
		errResponse(c, 500, nil)
	} else {
		tx.Commit()
	}
	return err
}

func errResponse(c *gin.Context, code int, err interface{}) {
	if err == nil {
		err = http.StatusText(code)
	}
	c.JSON(code, apiError{Error: fmt.Sprint(err)})
}

func internalError(c *gin.Context, err error) {
	c.Error(err)
	errResponse(c, 500, nil)
}

func successResponse(c *gin.Context, code int, value interface{}) {
	c.JSON(code, apiSuccess{Value: value})
}

func bindId(c *gin.Context, id *int) bool {
	paramId := c.Param("id")
	if i, err := strconv.Atoi(paramId); err == nil && paramId != "" {
		*id = i
		return true
	}
	errResponse(c, 404, nil)
	return false
}

func validateRequest(c *gin.Context, object interface{}) bool {
	err := validator.Validate(object)
	if err == nil {
		return true
	}
	errResponse(c, 400, err)
	return false
}

// Check if the error is a record not found
// If so, return 404, else 500
func dbNotFoundOrError(c *gin.Context, err error) {
	if err == gorm.ErrRecordNotFound {
		errResponse(c, 404, nil)
	} else {
		internalError(c, err)
	}
}
