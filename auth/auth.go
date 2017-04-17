package auth

import (
	"net/http"

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/github"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

func SetupAdmin(r gin.IRouter, db *gorm.DB) {
	admin := InviteAdmin{db: db}
	invites := r.Group("/invites")
	{
		invites.POST("", admin.Create)
	}
}

func Setup(r gin.IRouter, db *gorm.DB) {
	gh := github.NewService(db)

	r.GET("/", Index)

	authRoutes := r.Group("/oauth")
	{

		signup := Signup{db: db, gh: gh}
		authRoutes.GET("/signin/:token", signup.SignIn)
		authRoutes.GET("/callback", signup.Callback)
	}

	tokenRoutes := r.Group("/token", RequiresUser())
	{
		tokenRoutes.POST("/refresh", func(c *gin.Context) {
			user := GetUser(c)
			err := db.Model(&user).Update("token", models.NewUser().Token).Error
			if err != nil {
				c.AbortWithError(500, err)
				return
			}
			c.Redirect(http.StatusFound, "/")
		})
	}
}

func Index(c *gin.Context) {
	session := sessions.Default(c)
	user_id := session.Get(USER_ID)

	if user_id == nil {
		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"logged_in": false,
		})
	} else {
		user := GetUser(c)

		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"logged_in": true,
			"login":     user.GithubName,
			"name":      user.Name,
			"gh_id":     user.GithubID,
			"email":     user.Email,
			"token":     user.Token,
		})
	}
}
