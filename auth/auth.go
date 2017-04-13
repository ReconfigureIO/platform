package auth

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	githuboauth "golang.org/x/oauth2/github"
)

func Setup(r gin.IRouter) {
	oauthConf := &oauth2.Config{
		ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		Scopes:       []string{"user"},
		Endpoint:     githuboauth.Endpoint,
	}
	log.Printf("%+v\n", oauthConf)

	r.GET("/", func(c *gin.Context) {
		context := context.Background()
		session := sessions.Default(c)
		token := session.Get("token")

		if token == nil {
			c.HTML(http.StatusOK, "index.tmpl", gin.H{
				"logged_in": false,
			})
		} else {
			oauthClient := oauthConf.Client(oauth2.NoContext, &oauth2.Token{AccessToken: token.(string)})
			client := github.NewClient(oauthClient)

			user, _, err := client.Users.Get(context, "")

			if err != nil {
				c.String(http.StatusNotFound, "User not found")
				return
			}

			c.HTML(http.StatusOK, "index.tmpl", gin.H{
				"logged_in": true,
				"login":     user.Login,
				"name":      user.Name,
				"avatar":    user.GetAvatarURL(),
				"email":     user.GetEmail(),
			})
		}
	})

	authRoutes := r.Group("/oauth")
	{

		authRoutes.GET("/signin", func(c *gin.Context) {
			url := oauthConf.AuthCodeURL("hoge", oauth2.AccessTypeOnline)
			c.Redirect(http.StatusMovedPermanently, url)
		})

		authRoutes.GET("/callback", func(c *gin.Context) {
			code := c.Query("code")

			token, err := oauthConf.Exchange(oauth2.NoContext, code)

			if err != nil {
				c.String(http.StatusBadRequest, "Error: %s", err)
				return
			}

			session := sessions.Default(c)
			fmt.Println(token.AccessToken)
			session.Set("token", token.AccessToken)
			session.Save()

			c.Redirect(http.StatusMovedPermanently, "/")
		})
	}
}
