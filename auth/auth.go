package auth

import (
	"net/http"

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/github"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

// SetupAdmin sets up admin routes.
func SetupAdmin(r gin.IRouter, db *gorm.DB) {
	admin := inviteAdmin{db: db}
	invites := r.Group("/invites")
	{
		invites.POST("", admin.Create)
	}
}

// Setup sets all routes.
func Setup(r gin.IRouter, db *gorm.DB) {
	gh := github.New(db)

	r.GET("/", Index)

	authRoutes := r.Group("/oauth")
	{

		signup := signupUser{db: db, gh: gh}
		authRoutes.GET("/signin", signup.ResignIn)
		authRoutes.GET("/signup/:token", signup.SignUp)
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

// Index handles request to the site root.
func Index(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get(strUserID)

	if userID == nil {
		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"logged_in": false,
		})
	} else {
		user, exists := CheckUser(c)
		if exists && user != (models.User{}) {
			c.HTML(http.StatusOK, "index.tmpl", gin.H{
				"logged_in": true,
				"login":     user.GithubName,
				"name":      user.Name,
				"gh_id":     user.GithubID,
				"email":     user.Email,
				"token":     user.Token,
			})
		} else {
			session.Clear()
			c.HTML(http.StatusOK, "index.tmpl", gin.H{
				"logged_in": false,
			})
		}
	}
}
