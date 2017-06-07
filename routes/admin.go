package routes

import (
	"github.com/ReconfigureIO/platform/handlers/admin"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

// SetupAdmin sets up admin routes.
func SetupAdmin(r gin.IRouter, db *gorm.DB) {
	admin := admin.InviteAdmin{db}
	invites := r.Group("/invites")
	{
		invites.POST("", admin.Create)
	}
}
