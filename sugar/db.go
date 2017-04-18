package sugar

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	validator "gopkg.in/validator.v2"
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

func ErrResponse(c *gin.Context, code int, err interface{}) {
	if err == nil {
		err = http.StatusText(code)
	}
	c.JSON(code, apiError{Error: fmt.Sprint(err)})
}

func InternalError(c *gin.Context, err error) {
	c.Error(err)
	ErrResponse(c, 500, nil)
}

func SuccessResponse(c *gin.Context, code int, value interface{}) {
	c.JSON(code, apiSuccess{Value: value})
}

// Check if the error is a record not found
// If so, return 404, else 500
func NotFoundOrError(c *gin.Context, err error) {
	if err == gorm.ErrRecordNotFound {
		ErrResponse(c, 404, nil)
	} else {
		InternalError(c, err)
	}
}

func ValidateRequest(c *gin.Context, object interface{}) bool {
	err := validator.Validate(object)
	if err == nil {
		return true
	}
	ErrResponse(c, 400, err)
	return false
}
