package auth

import (
	"github.com/ReconfigureIO/platform/models"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

const (
	USER_ID = "user_id"
	USER    = "user"
)

func SessionAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		user_id := session.Get(USER_ID)
		if user_id != nil {
			c.Set(USER_ID, user_id.(int))
		}
	}
}

func LoadUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user_id, exists := c.Get(USER_ID)
		if !exists {
			return
		}
		user := models.User{}
		err := db.First(&user, user_id.(int)).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			c.AbortWithError(500, err)
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
