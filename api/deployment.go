package api

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ReconfigureIO/platform/auth"
	"github.com/ReconfigureIO/platform/models"
	. "github.com/ReconfigureIO/platform/sugar"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

type Deployment struct{}

func (d Deployment) Query(c *gin.Context) *gorm.DB {
	user := auth.GetUser(c)
	return db.Joins("left join builds on builds.id = deployments.build_id").Joins("left join projects on projects.id = builds.project_id").
		Where("projects.user_id=?", user.ID)
}

// Get the first deployment by ID, 404 if it doesn't exist
func (d Deployment) ById(c *gin.Context) (models.Deployment, error) {
	dep := models.Deployment{}
	var id int
	if !bindId(c, &id) {
		return dep, errNotFound
	}
	err := d.Query(c).First(&dep, id).Error

	if err != nil {
		NotFoundOrError(c, err)
		return dep, err
	}
	return dep, nil
}

func (d Deployment) Create(c *gin.Context) {
	post := models.PostDeployment{}
	c.BindJSON(&post)

	if !ValidateRequest(c, post) {
		return
	}

	// Ensure that the project exists, and the user has permissions for it
	build := models.Build{}
	err := Build{}.Query(c).First(&build, post.BuildID).Error
	if err != nil {
		NotFoundOrError(c, err)
		return
	}

	parentbuild := models.Build{}
	db.Where(&models.Build{ID: post.BuildID}).First(&parentbuild)

	newDep := models.Deployment{
		BuildID: post.BuildID,
		Command: post.Command,
	}
	err = db.Create(&newDep).Error
	if err != nil {
		InternalError(c, err)
		return
	}
	_, err = mockDeploy.RunDeployment(c, newDep)
	if err != nil {
		ErrResponse(c, 500, err)
		return
	}
	SuccessResponse(c, 201, newDep)
}

func (d Deployment) List(c *gin.Context) {
	build := c.DefaultQuery("build", "")
	deployments := []models.Deployment{}
	q := d.Query(c)

	if id, err := strconv.Atoi(build); err == nil && build != "" {
		q = q.Where(&models.Deployment{BuildID: id})
	}
	err := q.Find(&deployments).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		InternalError(c, err)
		return
	}

	SuccessResponse(c, 200, deployments)
}

func (d Deployment) Get(c *gin.Context) {
	outputdep, err := d.ById(c)
	if err != nil {
		return
	}
	SuccessResponse(c, 200, outputdep)
}

func (d Deployment) Logs(c *gin.Context) {
	targetdep, err := d.ById(c)
	logs, err := mockDeploy.GetJobStream(targetdep.ID)
	if err != nil {
		ErrResponse(c, 500, err)
		return
	}
	SuccessResponse(c, 200, logs)
}

func (d Deployment) CanPostEvent(c *gin.Context, dep models.Deployment) bool {
	user, loggedIn := auth.CheckUser(c)
	if loggedIn && dep.Build.Project.UserID == user.ID {
		return true
	}
	token, exists := c.GetQuery("token")
	if exists && dep.Token == token {
		return true
	}
	return false
}

func (d Deployment) CreateEvent(c *gin.Context) {
	dep, err := d.UnauthOne(c)
	if err != nil {
		return
	}

	if !d.CanPostEvent(c, dep) {
		c.AbortWithStatus(403)
		return
	}

	event := models.PostDepEvent{}
	c.BindJSON(&event)

	if !ValidateRequest(c, event) {
		return
	}

	currentStatus := dep.Status()

	if !models.CanTransition(currentStatus, event.Status) {
		ErrResponse(c, 400, fmt.Sprintf("%s not valid when current status is %s", event.Status, currentStatus))
		return
	}

	newEvent, err := AddEvent(&dep.DepJob, event)

	if err != nil {
		c.Error(err)
		ErrResponse(c, 500, nil)
		return
	}

	SuccessResponse(c, 200, newEvent)
}

func AddEvent(DepJob *models.DepJob, event models.PostDepEvent) (models.DepJobEvent, error) {
	newEvent := models.DepJobEvent{
		Timestamp: time.Now(),
		Status:    event.Status,
		Message:   event.Message,
		Code:      event.Code,
	}
	err := db.Model(&DepJob).Association("Events").Append(newEvent).Error
	if err != nil {
		return models.DepJobEvent{}, nil
	}
	return newEvent, nil
}

func (d Deployment) UnauthOne(c *gin.Context) (models.Deployment, error) {
	dep := models.Deployment{}
	var id int
	if !bindId(c, &id) {
		return dep, errNotFound
	}
	q := db.Preload("Project").Preload("DepJob").Preload("DepJob.Events")
	err := q.First(&dep, id).Error
	return dep, err
}
