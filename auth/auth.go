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
		user_id := session.Get("user_id")

		if user_id == nil {
			c.HTML(http.StatusOK, "index.tmpl", gin.H{
				"logged_in": false,
			})
		} else {
			user := models.User{}

			err := db.First(&user, user_id.(int)).Error
			if err != nil {
				c.Error(err)
				return
			}

			c.HTML(http.StatusOK, "index.tmpl", gin.H{
				"logged_in": true,
				"login":     user.GithubName,
				"name":      user.Name,
				"email":     user.Email,
			})
		}
	})

	authRoutes := r.Group("/oauth")
	{

		authRoutes.GET("/signin", func(c *gin.Context) {
			url := gh.OauthConf.AuthCodeURL("hoge", oauth2.AccessTypeOnline)
			c.Redirect(http.StatusMovedPermanently, url)
		})

		authRoutes.GET("/callback", func(c *gin.Context) {
			code := c.Query("code")

			token, err := gh.OauthConf.Exchange(oauth2.NoContext, code)

			if err != nil {
				c.String(http.StatusBadRequest, "Error: %s", err)
				return
			}

			user, err := gh.GetOrCreateUser(token.AccessToken)
			if err != nil {
				c.Error(err)
				//				errResponse(c, 500, nil)
				return
			}

			session := sessions.Default(c)
			session.Set("user_id", user.ID)
			session.Save()
			c.Redirect(http.StatusMovedPermanently, "/")
		})
	}
}
