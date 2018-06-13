package auth

import (
	"context"
	"errors"
	"net/http"

	"github.com/ReconfigureIO/platform/pkg/models"
	"github.com/ReconfigureIO/platform/pkg/service/auth"
	"github.com/ReconfigureIO/platform/pkg/service/auth/github"
	"github.com/ReconfigureIO/platform/pkg/service/leads"
	"github.com/ReconfigureIO/platform/pkg/sugar"
	"github.com/dchest/uniuri"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

const (
	strInviteToken = "invite_token"
	strLoginToken  = "login_token"
	strRedirectURL = "redirect_url"
)

type SignupUser struct {
	DB          *gorm.DB
	AuthService auth.Service
	Leads       leads.Leads
}

func (s *SignupUser) GetAuthToken(token string) (models.InviteToken, error) {
	var i models.InviteToken
	err := s.DB.Where(&models.InviteToken{Token: token}).First(&i).Error
	return i, err
}

func checkRedirURL(c *gin.Context, session sessions.Session) {
	// new auth flow. clear redirect url if still part of session.
	session.Delete(strRedirectURL)

	redirURL := c.Query(strRedirectURL)
	if redirURL != "" {
		session.Set(strRedirectURL, redirURL)
	}
}

func (s *SignupUser) ResignIn(c *gin.Context) {
	newState := uniuri.NewLen(64)
	session := sessions.Default(c)
	session.Set("login_token", newState)
	checkRedirURL(c, session)
	session.Save()

	url := s.AuthService.RedirectURL(newState)
	c.Redirect(http.StatusFound, url)
}

func (s *SignupUser) Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()

	c.Status(http.StatusNoContent)
}

func (s *SignupUser) SignUp(c *gin.Context) {
	token := c.Param("token")

	invite, err := s.GetAuthToken(token)
	if err != nil {
		sugar.TokenNotFoundOrError(c, err)
		return
	}
	session := sessions.Default(c)
	session.Set(strInviteToken, invite.Token)
	checkRedirURL(c, session)
	session.Save()

	url := s.AuthService.RedirectURL(invite.Token)
	c.Redirect(http.StatusFound, url)
}

// Create a new token for this context and
func (s *SignupUser) SignUpNoToken(c *gin.Context) {
	invite := models.NewInviteToken()
	invite.IntercomId = c.Query("lead_id")
	err := s.DB.Create(&invite).Error
	if err != nil {
		sugar.InternalError(c, err)
		return
	}

	session := sessions.Default(c)
	session.Set(strInviteToken, invite.Token)
	checkRedirURL(c, session)
	session.Save()

	url := s.AuthService.RedirectURL(invite.Token)
	c.Redirect(http.StatusFound, url)
}

// user tried to sign up without an invite token
func (s *SignupUser) NoToken(c *gin.Context) {
	sugar.ErrResponse(c, 400, "invite token required to sign up to Reconfigure.io")
	return
}

func (s *SignupUser) StoredToken(c *gin.Context, session sessions.Session) (string, bool, error) {
	stateToken := c.Query("state")

	storedToken := session.Get(strInviteToken)
	if storedToken != nil {
		session.Delete(strInviteToken)
		if s, ok := storedToken.(string); ok && s == stateToken {
			return storedToken.(string), true, nil
		}
	}

	loginToken := session.Get(strLoginToken)
	if loginToken != nil {
		session.Delete(strLoginToken)

		if s, ok := loginToken.(string); ok && s == stateToken {
			return loginToken.(string), false, nil
		}
	}

	return "", true, errors.New("Error: No valid tokens")
}

func (s *SignupUser) Callback(c *gin.Context) {
	session := sessions.Default(c)

	storedToken, newUser, err := s.StoredToken(c, session)
	session.Save()

	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	var invite models.InviteToken

	if newUser {
		invite, err = s.GetAuthToken(storedToken)
		if err != nil {
			sugar.NotFoundOrError(c, err)
			return
		}
		defer s.DB.Delete(&invite)
	}

	code := c.Query("code")

	accessToken, err := s.AuthService.Exchange(context.Background(), code)
	if err != nil {
		c.String(http.StatusBadRequest, "Error: %s", err)
		return
	}

	user, err := s.AuthService.GetOrCreateUser(c, accessToken, newUser)
	if err != nil {
		if _, ok := err.(github.UserError); ok {
			sugar.ErrResponse(c, 400, err)
			return
		}
		if err == gorm.ErrRecordNotFound {
			c.Redirect(http.StatusMovedPermanently, "https://reconfigure.io/sign-up")
		} else {
			sugar.InternalError(c, err)
		}
		return
	}

	location := "/"

	redirURL, _ := session.Get(strRedirectURL).(string)
	if redirURL != "" {
		location = redirURL
		// done with redirect_url
		session.Delete(strRedirectURL)
	}
	session.Set("user_id", user.ID)
	session.Save()

	// If we have a newUser, we need to mark the new user as invited
	if newUser {
		err = s.Leads.Invited(invite, user)
		if err != nil {
			sugar.NotFoundOrError(c, err)
			return
		}
	}

	c.Redirect(http.StatusFound, location)
}
