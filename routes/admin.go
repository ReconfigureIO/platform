package routes

import (
	"github.com/ReconfigureIO/platform/handlers/auth"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

// SetupAdmin sets up admin routes.
func SetupAdmin(r gin.IRouter, db *gorm.DB) {
	admin := auth.InviteAdmin{db}
	invites := r.Group("/invites")
	{
		invites.POST("", admin.Create)
	}
}
