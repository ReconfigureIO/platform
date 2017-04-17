package auth

import (
	"github.com/ReconfigureIO/platform/models"
	. "github.com/ReconfigureIO/platform/sugar"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

type InviteAdmin struct {
	db *gorm.DB
}

func (inv InviteAdmin) Query(c *gin.Context) *gorm.DB {
	return inv.db
}

func (inv InviteAdmin) Create(c *gin.Context) {
	invite := models.NewInviteToken()
	err := inv.db.Create(&invite).Error
	if err != nil {
		InternalError(c, err)
		return
	}

	SuccessResponse(c, 201, invite)
}
