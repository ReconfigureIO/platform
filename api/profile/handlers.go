package profile

import (
	"github.com/ReconfigureIO/platform/auth"
	"github.com/ReconfigureIO/platform/sugar"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

// Profile handles requests for profile get & update
type Profile struct {
	DB *gorm.DB
}

func (p Profile) Get(c *gin.Context) {
	user := auth.GetUser(c)

	prof := ProfileData{}
	prof.FromUser(user)

	sugar.SuccessResponse(c, 200, prof)
}

func (p Profile) Update(c *gin.Context) {
	user := auth.GetUser(c)

	prof := ProfileData{}
	prof.FromUser(user)

	c.BindJSON(&prof)

	if !sugar.ValidateRequest(c, prof) {
		return
	}

	prof.Apply(&user)

	err := p.DB.Save(&user).Error

	if err != nil {
		sugar.NotFoundOrError(c, err)
		return
	}

	prof.FromUser(user)
	sugar.SuccessResponse(c, 200, prof)
}
