package routes

import (
	"github.com/ReconfigureIO/platform/handlers/admin"
	"github.com/ReconfigureIO/platform/service/leads"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

// SetupAdmin sets up admin routes.
func SetupAdmin(r gin.IRouter, db *gorm.DB, leads leads.Leads) {
	admin := admin.InviteAdmin{DB: db, Leads: leads}
	invites := r.Group("/invites")
	{
		invites.POST("", admin.Create)
		invites.POST("/sync", admin.Sync)
	}
}
