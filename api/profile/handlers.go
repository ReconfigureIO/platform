package profile

import (
	//	"fmt"

	//	"github.com/ReconfigureIO/platform/auth"
	//	"github.com/ReconfigureIO/platform/models"
	//	"github.com/ReconfigureIO/platform/sugar"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

// Profile handles requests for profile get & update
type Profile struct {
	DB *gorm.DB
}

func (p Profile) Get(c *gin.Context) {

}

func (p Profile) Update(c *gin.Context) {

}
