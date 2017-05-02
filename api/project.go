package api

import (
	"github.com/ReconfigureIO/platform/auth"

	"github.com/ReconfigureIO/platform/models"
	. "github.com/ReconfigureIO/platform/sugar"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

type Project struct{}

type PostProject struct {
	Name string `json:"name" validate:"nonzero"`
}

func (p Project) Query(c *gin.Context) *gorm.DB {
	user := auth.GetUser(c)
	return db.Where("projects.user_id=?", user.ID)
}

// Get the first build by ID, 404 if it doesn't exist
func (p Project) ById(c *gin.Context) (models.Project, error) {
	project := models.Project{}
	var id int
	if !bindId(c, &id) {
		return project, errNotFound
	}
	err := p.Query(c).First(&project, id).Error

	if err != nil {
		NotFoundOrError(c, err)
		return project, err
	}
	return project, nil
}

func (p Project) Create(c *gin.Context) {
	post := PostProject{}
	c.BindJSON(&post)
	if !ValidateRequest(c, post) {
		return
	}
	user := auth.GetUser(c)
	newProject := models.Project{UserID: user.ID, Name: post.Name}
	db.Create(&newProject)
	SuccessResponse(c, 201, newProject)
}

func (p Project) Update(c *gin.Context) {
	project, err := p.ById(c)
	if err != nil {
		return
	}

	post := PostProject{}
	c.BindJSON(&post)

	if !ValidateRequest(c, post) {
		return
	}

	db.Model(&project).Updates(post)
	SuccessResponse(c, 200, project)
}

func (p Project) List(c *gin.Context) {
	projects := []models.Project{}
	err := p.Query(c).Find(&projects).Error
	if err != nil {
		InternalError(c, err)
		return
	}

	SuccessResponse(c, 200, projects)
}

func (p Project) Get(c *gin.Context) {
	project, err := p.ById(c)
	if err != nil {
		return
	}

	c.JSON(200, project)
}
