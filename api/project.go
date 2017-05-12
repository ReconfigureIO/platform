package api

import (
	"github.com/ReconfigureIO/platform/auth"
	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/sugar"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

// Project handles project requests.
type Project struct{}

// PostProject is post request for new project.
type PostProject struct {
	Name string `json:"name" validate:"nonzero"`
}

// Query queries the db for current user's projects.
func (p Project) Query(c *gin.Context) *gorm.DB {
	user := auth.GetUser(c)
	return db.Where("projects.user_id=?", user.ID)
}

// ByID get the first build by ID, 404 if it doesn't exist
func (p Project) ByID(c *gin.Context) (models.Project, error) {
	project := models.Project{}
	var id int
	if !bindID(c, &id) {
		return project, errNotFound
	}
	err := p.Query(c).First(&project, id).Error

	if err != nil {
		sugar.NotFoundOrError(c, err)
		return project, err
	}
	return project, nil
}

// Create creates a new project
func (p Project) Create(c *gin.Context) {
	post := PostProject{}
	c.BindJSON(&post)
	if !sugar.ValidateRequest(c, post) {
		return
	}
	user := auth.GetUser(c)
	newProject := models.Project{UserID: user.ID, Name: post.Name}
	db.Create(&newProject)
	sugar.SuccessResponse(c, 201, newProject)
}

// Update updateds an existing project.
func (p Project) Update(c *gin.Context) {
	project, err := p.ByID(c)
	if err != nil {
		return
	}

	post := PostProject{}
	c.BindJSON(&post)

	if !sugar.ValidateRequest(c, post) {
		return
	}

	db.Model(&project).Updates(post)
	sugar.SuccessResponse(c, 200, project)
}

// List lists all projects.
func (p Project) List(c *gin.Context) {
	projects := []models.Project{}
	err := p.Query(c).Find(&projects).Error
	if err != nil {
		sugar.InternalError(c, err)
		return
	}

	sugar.SuccessResponse(c, 200, projects)
}

// Get fetches a project.
func (p Project) Get(c *gin.Context) {
	project, err := p.ByID(c)
	if err != nil {
		return
	}

	c.JSON(200, project)
}
