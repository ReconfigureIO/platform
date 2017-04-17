package auth

import (
	"net/http"

	"github.com/ReconfigureIO/platform/service/github"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"golang.org/x/oauth2"
)

type Signup struct {
	db *gorm.DB
	gh *github.GithubService
}

func (s *Signup) SignIn(c *gin.Context) {
	token := c.Param("token")
	session := sessions.Default(c)
	session.Set("invite_token", token)
	session.Save()

	url := s.gh.OauthConf.AuthCodeURL(token, oauth2.AccessTypeOnline)
	c.Redirect(http.StatusFound, url)
}

func (s *Signup) Callback(c *gin.Context) {
	state_token := c.Query("state")
	session := sessions.Default(c)
	stored_token := session.Get("invite_token")

	if state_token != stored_token {
		c.String(http.StatusBadRequest, "Error: Invalid token")
		return
	}

	code := c.Query("code")

	token, err := s.gh.OauthConf.Exchange(oauth2.NoContext, code)

	if err != nil {
		c.String(http.StatusBadRequest, "Error: %s", err)
		return
	}

	user, err := s.gh.GetOrCreateUser(c, token.AccessToken)
	if err != nil {
		c.Error(err)
		//				errResponse(c, 500, nil)
		return
	}

	session.Set("user_id", user.ID)
	session.Save()
	c.Redirect(http.StatusMovedPermanently, "/")
}
