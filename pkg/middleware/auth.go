package middleware

import (
	"crypto/subtle"
	"strconv"
	"strings"

	"github.com/ReconfigureIO/platform/pkg/models"
	"github.com/ReconfigureIO/platform/service/events"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

const (
	strUserID = "user_id"
	strUser   = "reco_user"
)

// SessionAuth handles session authentication.
func SessionAuth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		userID := session.Get(strUserID)
		if userID != nil {
			user := models.User{}
			err := db.First(&user, "id = ?", userID).Error
			if err != nil && err != gorm.ErrRecordNotFound {
				c.AbortWithError(500, err)
				return
			}
			if err != gorm.ErrRecordNotFound {
				c.Set(strUser, user)
			}
		}
	}
}

// TokenAuth handles token authentication.
func TokenAuth(db *gorm.DB, events events.EventService) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, exists := c.Get(strUser)
		if exists {
			return
		}

		username, pass, ok := c.Request.BasicAuth()
		if !ok {
			return
		}

		ghID, err := strconv.Atoi(strings.TrimPrefix(username, "gh_"))
		if err != nil {
			return
		}

		user := models.User{GithubID: ghID}
		err = db.Where(user).First(&user).Error

		if err == gorm.ErrRecordNotFound {
			// Credentials doesn't match, we return 401 and abort handlers chain.
			c.Header("WWW-Authenticate", "Authorization Required")
			c.AbortWithStatus(401)
			return
		}

		if err != nil {
			c.Error(err)
			c.AbortWithStatus(500)
		}

		if !secureCompare(user.Token, pass) {
			// Credentials doesn't match, we return 401 and abort handlers chain.
			c.Header("WWW-Authenticate", "Authorization Required")
			c.AbortWithStatus(401)
			return
		}

		c.Set(strUser, user)
		events.Seen(user)
	}
}

// RequiresUser exits with a 403 if the user doesn't exist
func RequiresUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		_, exists := c.Get(strUser)
		if !exists {
			c.AbortWithStatus(403)
		}
	}
}

// GetUser gets the current user.
func GetUser(c *gin.Context) models.User {
	u := c.MustGet(strUser)
	return u.(models.User)
}

// CheckUser validates a user.
func CheckUser(c *gin.Context) (models.User, bool) {
	user := models.User{}
	u, exists := c.Get(strUser)
	if exists {
		user = u.(models.User)
	}
	return user, exists
}

func secureCompare(given, actual string) bool {
	if subtle.ConstantTimeEq(int32(len(given)), int32(len(actual))) == 1 {
		return subtle.ConstantTimeCompare([]byte(given), []byte(actual)) == 1
	}
	/* Securely compare actual to itself to keep constant time, but always return false */
	return subtle.ConstantTimeCompare([]byte(actual), []byte(actual)) == 1 && false
}
