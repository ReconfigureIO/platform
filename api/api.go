package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	validator "gopkg.in/validator.v2"

	"github.com/ReconfigureIO/platform/service/aws"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
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

func errResponse(c *gin.Context, code int, err interface{}) {
	if err == nil {
		err = http.StatusText(code)
	}
	c.JSON(code, apiError{Error: fmt.Sprint(err)})
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
	if err := validator.Validate(object); err == nil {
		return true
	}
	errResponse(c, 400, nil)
	return false
}
