package auth

import (
	"hash/fnv"
	"net/http"

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/sugar"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

type SignupUserOnPrem struct {
	DB *gorm.DB
}

type User struct {
	Email string `validate:"nonzero"`
}

// SignUpNoToken
func (s *SignupUserOnPrem) SignUpNoToken(c *gin.Context) {
	session := sessions.Default(c)
	newUser := User{}
	newUser.Email = c.Query("email")
	h := fnv.New32a()
	_, err := h.Write([]byte(newUser.Email))

	if sugar.ValidateRequest(c, newUser) {
		user, err := models.CreateOrUpdateUser(s.DB, models.User{
			Email:    newUser.Email,
			GithubID: int(int32(h.Sum32())),
		}, true)
		if err != nil {
			sugar.InternalError(c, err)
			return
		}
		session.Set("user_id", user.ID)
		session.Save()
	} else {
		sugar.ErrResponse(c, 400, err)
		return
	}
	location := "/"
	c.Redirect(http.StatusFound, location)
}
