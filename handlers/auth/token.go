package auth

import (
	"net/http"

	"github.com/ReconfigureIO/platform/middleware"
	"github.com/ReconfigureIO/platform/models"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

type Token struct {
	DB *gorm.DB
}

func (t Token) Refresh(c *gin.Context) {
	user := middleware.GetUser(c)
	err := t.DB.Model(&user).Update("token", models.NewUser().Token).Error
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	c.Redirect(http.StatusFound, "/")
}
