package api

import (
	"errors"
	"strconv"

	"github.com/ReconfigureIO/platform/service/aws"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

var (
	NOT_FOUND = errors.New("Not Found")

	db *gorm.DB

	awsSession = aws.New(aws.ServiceConfig{
		Bucket:        "reconfigureio-builds",
		Queue:         "build-jobs",
		JobDefinition: "sdaccel-builder-build",
	})
)

type ApiError struct {
	Error string `json:"error"`
}

// M is a convenience wrapper for a map.
type M map[string]interface{}

// DB sets the database to use for the API.
func DB(d *gorm.DB) {
	db = d
}

func stringToInt(s string, c *gin.Context) (int, error) {
	i, err := strconv.Atoi(s)
	if err != nil {
		c.AbortWithStatus(404)
		return 0, NOT_FOUND
	}
	return i, nil
}
