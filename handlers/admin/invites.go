package admin

import (
	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/leads"
	"github.com/ReconfigureIO/platform/sugar"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

type InviteAdmin struct {
	DB    *gorm.DB
	Leads leads.Leads
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

func (inv InviteAdmin) Sync(c *gin.Context) {
	invited, err := inv.Leads.Invite(20)
	if err != nil {
		sugar.InternalError(c, err)
		return
	}
	sugar.SuccessResponse(c, 200, invited)
}
