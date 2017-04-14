package api

import (
	"github.com/ReconfigureIO/platform/auth"

	"github.com/ReconfigureIO/platform/models"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

type Project struct{}

type PostProject struct {
	Name string `json:"name"`
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
		if err == gorm.ErrRecordNotFound {
			errResponse(c, 404, nil)
		} else {
			internalError(c, err)
		}
		return project, err
	}
	return project, nil
}

func (p Project) Create(c *gin.Context) {
	post := models.PostProject{}
	c.BindJSON(&post)
	if !validateRequest(c, post) {
		return
	}
	user := auth.GetUser(c)
	newProject := models.Project{UserID: user.ID, Name: post.Name}
	db.Create(&newProject)
	successResponse(c, 201, newProject)
}

func (p Project) Update(c *gin.Context) {
	project, err := p.ById(c)
	if err != nil {
		return
	}

	post := models.PostProject{}
	c.BindJSON(&post)

	if !validateRequest(c, post) {
		return
	}

	db.Model(&project).Updates(post)
	successResponse(c, 200, project)
}

func (p Project) List(c *gin.Context) {
	projects := []models.Project{}
	err := p.Query(c).Find(&projects).Error
	if err != nil {
		internalError(c, err)
		return
	}

	successResponse(c, 200, projects)
}

func (p Project) Get(c *gin.Context) {
	project, err := p.ById(c)
	if err != nil {
		return
	}

	c.JSON(200, project)
}
