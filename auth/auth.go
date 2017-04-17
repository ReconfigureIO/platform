package auth

import (
	"net/http"

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/github"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"golang.org/x/oauth2"
)

func Setup(r gin.IRouter, db *gorm.DB) {
	gh := github.NewService(db)

	r.GET("/", func(c *gin.Context) {
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
	})

	authRoutes := r.Group("/oauth")
	{

		authRoutes.GET("/signin/:token", func(c *gin.Context) {
			token := c.Param("token")
			session := sessions.Default(c)
			session.Set("invite_token", token)
			session.Save()

			url := gh.OauthConf.AuthCodeURL(token, oauth2.AccessTypeOnline)
			c.Redirect(http.StatusFound, url)
		})

		authRoutes.GET("/callback", func(c *gin.Context) {
			state_token := c.Query("state")
			session := sessions.Default(c)
			stored_token := session.Get("invite_token")

			if state_token != stored_token {
				c.String(http.StatusBadRequest, "Error: Invalid token")
				return
			}

			code := c.Query("code")

			token, err := gh.OauthConf.Exchange(oauth2.NoContext, code)

			if err != nil {
				c.String(http.StatusBadRequest, "Error: %s", err)
				return
			}

			user, err := gh.GetOrCreateUser(c, token.AccessToken)
			if err != nil {
				c.Error(err)
				//				errResponse(c, 500, nil)
				return
			}

			session.Set("user_id", user.ID)
			session.Save()
			c.Redirect(http.StatusMovedPermanently, "/")
		})
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
