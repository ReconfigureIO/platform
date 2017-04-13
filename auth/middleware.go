package auth

import (
	"encoding/base64"
	"strings"

	"github.com/ReconfigureIO/platform/models"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

const (
	USER_ID = "user_id"
	USER    = "user"
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
		header := strings.TrimPrefix(c.Request.Header.Get("Authorization"), "Basic ")
		bs, err := base64.StdEncoding.DecodeString(header)
		if err != nil {
			c.Error(err)
			return
		}
		decoded := strings.SplitN(string(bs), ":", 2)
		if len(decoded) == 0 {
			return
		}
		token := models.AuthToken{Token: decoded[0]}
		err = db.Preload("User").Where(token).First(&token).Error
		if err == gorm.ErrRecordNotFound {
			return
		}
		if err != nil {
			c.Error(err)

			c.AbortWithStatus(500)
		}
		c.Set(USER, token.User)
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
