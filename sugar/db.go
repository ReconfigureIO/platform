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

// ErrResponse responds to the request with code and error.
func ErrResponse(c *gin.Context, code int, err interface{}) {
	if err == nil {
		err = http.StatusText(code)
	}
	c.JSON(code, apiError{Error: fmt.Sprint(err)})
}

// InternalError responds to request with an internal error.
func InternalError(c *gin.Context, err error) {
	c.Error(err)
	ErrResponse(c, 500, nil)
}

// SuccessResponse responds to request with code and value.
func SuccessResponse(c *gin.Context, code int, value interface{}) {
	c.JSON(code, apiSuccess{Value: value})
}

// NotFoundOrError checks if the error is a record not found error.
// If so, return 404, else 500
func NotFoundOrError(c *gin.Context, err error) {
	if err == gorm.ErrRecordNotFound {
		ErrResponse(c, 404, nil)
	} else {
		InternalError(c, err)
	}
}

// TokenNotFoundOrError checks if the error is a record not found error.
// If so, return token invalid, else 500
func TokenNotFoundOrError(c *gin.Context, err error) {
	if err == gorm.ErrRecordNotFound {
		ErrResponse(c, 400, "Signup Token Invalid")
	} else {
		InternalError(c, err)
	}
}

// ValidateRequest validates the request using validator.
func ValidateRequest(c *gin.Context, object interface{}) bool {
	err := validator.Validate(object)
	if err == nil {
		return true
	}
	ErrResponse(c, 400, err)
	return false
}
