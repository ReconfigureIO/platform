package auth

import (
	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/sugar"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

type inviteAdmin struct {
	db *gorm.DB
}

func (inv inviteAdmin) Query(c *gin.Context) *gorm.DB {
	return inv.db
}

func (inv inviteAdmin) Create(c *gin.Context) {
	invite := models.NewInviteToken()
	err := inv.db.Create(&invite).Error
	if err != nil {
		sugar.InternalError(c, err)
		return
	}

	sugar.SuccessResponse(c, 201, invite)
}
