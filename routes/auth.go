package routes

import (
	"net/http"

	"github.com/ReconfigureIO/platform/handlers/auth"
	"github.com/ReconfigureIO/platform/middleware"
	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/github"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

// Setup sets all auth routes.
func SetupAuth(r gin.IRouter, db *gorm.DB) {
	gh := github.New(db)

	authRoutes := r.Group("/oauth")
	{

		signup := auth.SignupUser{db, gh}
		authRoutes.GET("/signin", signup.ResignIn)
		authRoutes.GET("/signup/:token", signup.SignUp)
		authRoutes.GET("/callback", signup.Callback)
		authRoutes.GET("/logout", signup.Logout)
	}

	tokenRoutes := r.Group("/token", middleware.RequiresUser())
	{
		tokenRoutes.POST("/refresh", func(c *gin.Context) {
			user := middleware.GetUser(c)
			err := db.Model(&user).Update("token", models.NewUser().Token).Error
			if err != nil {
				c.AbortWithError(500, err)
				return
			}
			c.Redirect(http.StatusFound, "/")
		})
	}
}