package auth

import (
	"crypto/subtle"
	"strconv"
	"strings"

	"github.com/ReconfigureIO/platform/models"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

const (
	USER_ID = "user_id"
	USER    = "reco_user"
)

func SessionAuth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		user_id := session.Get(USER_ID)
		if user_id != nil {
			user := models.User{}
			err := db.First(&user, user_id.(int)).Error
			if err != nil && err != gorm.ErrRecordNotFound {
				c.AbortWithError(500, err)
				return
			}
			c.Set(USER, user)
		}
	}
}

func TokenAuth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, exists := c.Get(USER)
		if exists {
			return
		}

		username, pass, ok := c.Request.BasicAuth()
		if !ok {
			return
		}

		gh_id, err := strconv.Atoi(strings.TrimPrefix(username, "gh_"))
		if err != nil {
			return
		}

		user := models.User{GithubID: gh_id}
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

		c.Set(USER, user)
	}
}

// exit with a 403 if the user doesn't exist
func RequiresUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		_, exists := c.Get(USER)
		if !exists {
			c.AbortWithStatus(403)
		}
	}
}

func GetUser(c *gin.Context) models.User {
	u := c.MustGet(USER)
	return u.(models.User)
}

func secureCompare(given, actual string) bool {
	if subtle.ConstantTimeEq(int32(len(given)), int32(len(actual))) == 1 {
		return subtle.ConstantTimeCompare([]byte(given), []byte(actual)) == 1
	}
	/* Securely compare actual to itself to keep constant time, but always return false */
	return subtle.ConstantTimeCompare([]byte(actual), []byte(actual)) == 1 && false
}
