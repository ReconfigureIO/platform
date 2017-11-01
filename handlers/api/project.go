package api

import (
	"github.com/ReconfigureIO/platform/middleware"
	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/events"
	"github.com/ReconfigureIO/platform/sugar"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

// Project handles project requests.
type Project struct {
	Events events.EventService
}

// PostProject is post request for new project.
type PostProject struct {
	Name string `json:"name" validate:"nonzero"`
}

// Query queries the db for current user's projects.
func (p Project) Query(c *gin.Context) *gorm.DB {
	user := middleware.GetUser(c)
	return db.Where("projects.user_id=?", user.ID)
}

// ByID get the first build by ID, 404 if it doesn't exist
func (p Project) ByID(c *gin.Context) (models.Project, error) {
	project := models.Project{}
	var id string
	if !bindID(c, &id) {
		return project, errNotFound
	}
	err := p.Query(c).First(&project, "projects.id = ?", id).Error

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
	user := middleware.GetUser(c)
	newProject := models.Project{UserID: user.ID, Name: post.Name}
	if err := db.Create(&newProject).Error; err != nil {
		sugar.ErrResponse(c, 500, err)
	}

	sugar.EnqueueEvent(p.Events, c, "Created Project", map[string]interface{}{"id": newProject.ID, "name": newProject.Name})

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
