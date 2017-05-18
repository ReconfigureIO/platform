package auth

import (
	"errors"
	"net/http"

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/github"
	"github.com/ReconfigureIO/platform/sugar"
	"github.com/dchest/uniuri"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"golang.org/x/oauth2"
)

type signupUser struct {
	db *gorm.DB
	gh *github.Service
}

func (s *signupUser) GetAuthToken(token string) (models.InviteToken, error) {
	var i models.InviteToken
	err := s.db.Where(&models.InviteToken{Token: token}).First(&i).Error
	return i, err
}

func (s *signupUser) ResignIn(c *gin.Context) {
	newState := uniuri.NewLen(64)
	session := sessions.Default(c)
	session.Set("login_token", newState)
	session.Save()

	url := s.gh.OauthConf.AuthCodeURL(newState, oauth2.AccessTypeOnline)
	c.Redirect(http.StatusFound, url)
}

func (s *signupUser) SignUp(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		sugar.ErrResponse(c, 400, "invite token required")
		return
	}
	invite, err := s.GetAuthToken(token)
	if err != nil {
		sugar.NotFoundOrError(c, err)
		return
	}

	session := sessions.Default(c)
	session.Set("invite_token", invite.Token)
	session.Save()

	url := s.gh.OauthConf.AuthCodeURL(invite.Token, oauth2.AccessTypeOnline)
	c.Redirect(http.StatusFound, url)
}

func (s *signupUser) StoredToken(c *gin.Context, session sessions.Session) (string, bool, error) {
	stateToken := c.Query("state")

	storedToken := session.Get("invite_token")
	if storedToken != nil {
		session.Delete("invite_token")
		if storedToken.(string) == stateToken {
			return storedToken.(string), true, nil
		}
	}

	loginToken := session.Get("login_token")
	if loginToken != nil {
		session.Delete("login_token")

		if loginToken.(string) == stateToken {
			return loginToken.(string), false, nil
		}
	}

	return "", true, errors.New("Error: No valid tokens")
}

func (s *signupUser) Callback(c *gin.Context) {
	session := sessions.Default(c)

	storedToken, newUser, err := s.StoredToken(c, session)

	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	if newUser {
		invite, err := s.GetAuthToken(storedToken)
		if err != nil {
			sugar.NotFoundOrError(c, err)
			return
		}
		defer s.db.Delete(&invite)
	}

	code := c.Query("code")

	token, err := s.gh.OauthConf.Exchange(oauth2.NoContext, code)

	if err != nil {
		c.String(http.StatusBadRequest, "Error: %s", err)
		return
	}

	var user models.User
	if newUser {
		user, err = s.gh.GetOrCreateUser(c, token.AccessToken)
	} else {
		user, err = s.gh.GetUser(c, token.AccessToken)
	}

	if err != nil {
		c.Error(err)
		return
	}

	session.Set("user_id", user.ID)
	session.Save()

	c.Redirect(http.StatusMovedPermanently, "/")
}
