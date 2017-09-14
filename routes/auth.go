package routes

import (
	"github.com/ReconfigureIO/platform/handlers/auth"
	"github.com/ReconfigureIO/platform/middleware"
	"github.com/ReconfigureIO/platform/service/github"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

// Setup sets all auth routes.
func SetupAuth(r gin.IRouter, db *gorm.DB) {
	gh := github.New(db)

	authRoutes := r.Group("/oauth")
	{

		signup := auth.SignupUser{DB: db, GH: gh}
		authRoutes.GET("/signin", signup.ResignIn)
		authRoutes.GET("/signup/:token", signup.SignUp)
		authRoutes.GET("/signup/", signup.NoToken)
		authRoutes.GET("/callback", signup.Callback)
		authRoutes.GET("/logout", signup.Logout)
	}

	token := auth.Token{DB: db}
	tokenRoutes := r.Group("/token", middleware.RequiresUser())
	{
		tokenRoutes.GET("/refresh", token.Refresh)
	}
}
