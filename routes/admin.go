package routes

import (
	"github.com/ReconfigureIO/platform/handlers/admin"
	"github.com/ReconfigureIO/platform/service/leads"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

// SetupAdmin sets up admin routes.
func SetupAdmin(r gin.IRouter, db *gorm.DB, leads leads.Leads) {
	inviteAdmin := admin.InviteAdmin{DB: db, Leads: leads}
	invites := r.Group("/invites")
	{
		invites.POST("", inviteAdmin.Create)
		invites.POST("/sync", inviteAdmin.Sync)
	}
	buildAdmin := admin.Build{DB: db}
	builds := r.Group("/builds")
	{
		builds.GET("", buildAdmin.List)
	}
}
