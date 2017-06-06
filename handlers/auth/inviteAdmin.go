package auth

import (
	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/sugar"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

type InviteAdmin struct {
	DB *gorm.DB
}

func (inv InviteAdmin) Query(c *gin.Context) *gorm.DB {
	return inv.DB
}

func (inv InviteAdmin) Create(c *gin.Context) {
	invite := models.NewInviteToken()
	err := inv.DB.Create(&invite).Error
	if err != nil {
		sugar.InternalError(c, err)
		return
	}

	sugar.SuccessResponse(c, 201, invite)
}
