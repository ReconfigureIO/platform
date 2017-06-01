package api

import (
	"context"
	"fmt"
	"time"

	"github.com/ReconfigureIO/platform/auth"
	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/sugar"
	"github.com/dchest/uniuri"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

// Deployment handles request for deployments.
type Deployment struct{}

// Query fetches deployment for user and project.
func (d Deployment) Query(c *gin.Context) *gorm.DB {
	user := auth.GetUser(c)
	return db.Joins("left join builds on builds.id = deployments.build_id").Joins("left join projects on projects.id = builds.project_id").
		Where("projects.user_id=?", user.ID).
		Preload("Build").Preload("DepJob.Events").Preload("DepJob")
}

// ByID gets the first deployment by ID, 404 if it doesn't exist.
func (d Deployment) ByID(c *gin.Context) (models.Deployment, error) {
	dep := models.Deployment{}
	var id string
	if !bindID(c, &id) {
		return dep, errNotFound
	}
	err := d.Query(c).First(&dep, "deployments.id = ?", id).Error

	if err != nil {
		sugar.NotFoundOrError(c, err)
		return dep, err
	}
	return dep, nil
}

// Create creates a new deployment
func (d Deployment) Create(c *gin.Context) {
	post := models.PostDeployment{}
	c.BindJSON(&post)

	if !sugar.ValidateRequest(c, post) {
		return
	}

	// Ensure that the project exists, and the user has permissions for it
	build := models.Build{}
	err := Build{}.Query(c).First(&build, "builds.id = ?", post.BuildID).Error
	if err != nil {
		sugar.NotFoundOrError(c, err)
		return
	}

	depJob := models.DepJob{}
	db.Create(&depJob)

	newDep := models.Deployment{
		BuildID:  post.BuildID,
		Command:  post.Command,
		DepJobID: depJob.ID,
		Token:    uniuri.NewLen(64),
	}
	err = db.Create(&newDep).Error
	if err != nil {
		sugar.InternalError(c, err)
		return
	}

	callbackUrl := fmt.Sprintf("https://%s/deployments/%d/events?token=%s", c.Request.Host, newDep.ID, newDep.Token)

	instanceID, err := mockDeploy.RunDeployment(context.Background(), newDep, callbackUrl)
	if err != nil {
		sugar.InternalError(c, err)
		return
	}

	err = db.Model(&newDep).Update("InstanceID", instanceID).Error
	if err != nil {
		sugar.InternalError(c, err)
		return
	}

	sugar.SuccessResponse(c, 201, newDep)
}

// List lists all deployments.
func (d Deployment) List(c *gin.Context) {
	build := c.DefaultQuery("build", "")
	project := c.DefaultQuery("project", "")
	deployments := []models.Deployment{}
	q := d.Query(c)

	if project != "" {
		q = q.Where("builds.project_id=?", project)
	}

	if build != "" {
		q = q.Where(&models.Deployment{BuildID: build})
	}

	err := q.Find(&deployments).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		sugar.InternalError(c, err)
		return
	}

	sugar.SuccessResponse(c, 200, deployments)
}

// Get fetches a deployment.
func (d Deployment) Get(c *gin.Context) {
	outputdep, err := d.ByID(c)
	if err != nil {
		return
	}
	sugar.SuccessResponse(c, 200, outputdep)
}

func (d Deployment) Stop(c *gin.Context) {
	// set deployment status to "terminating"

	dep, err := d.unauthOne(c)
	if err != nil {
		return
	}

	event := models.PostDepEvent{
		Status:  "TERMINATING",
		Message: "USER REQUESTED TERMINATION",
		Code:    0,
	}

	if !sugar.ValidateRequest(c, event) {
		return
	}

	currentStatus := dep.Status()

	if !models.CanTransition(currentStatus, event.Status) {
		sugar.ErrResponse(c, 400, fmt.Sprintf("%s not valid when current status is %s", event.Status, currentStatus))
		return
	}

	newEvent, err := addEvent(&dep.DepJob, event)

	if err != nil {
		c.Error(err)
		sugar.ErrResponse(c, 500, nil)
		return
	}
	// stop instance (ID)

	err = mockDeploy.StopDeployment(context.Background(), dep)

	sugar.SuccessResponse(c, 200, newEvent)
}

// Logs stream logs for deployments.
func (d Deployment) Logs(c *gin.Context) {
	targetdep, err := d.ByID(c)
	if err != nil {
		return
	}
	streamDeploymentLogs(mockDeploy, c, &targetdep)
}

func (d Deployment) canPostEvent(c *gin.Context, dep models.Deployment) bool {
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

// CreateEvent creates a deployment event.
func (d Deployment) CreateEvent(c *gin.Context) {
	dep, err := d.unauthOne(c)
	if err != nil {
		return
	}

	if !d.canPostEvent(c, dep) {
		c.AbortWithStatus(403)
		return
	}

	event := models.PostDepEvent{}
	c.BindJSON(&event)

	if !sugar.ValidateRequest(c, event) {
		return
	}

	currentStatus := dep.Status()

	if !models.CanTransition(currentStatus, event.Status) {
		sugar.ErrResponse(c, 400, fmt.Sprintf("%s not valid when current status is %s", event.Status, currentStatus))
		return
	}

	newEvent, err := addEvent(&dep.DepJob, event)

	if err != nil {
		c.Error(err)
		sugar.ErrResponse(c, 500, nil)
		return
	}

	sugar.SuccessResponse(c, 200, newEvent)
}

func addEvent(DepJob *models.DepJob, event models.PostDepEvent) (models.DepJobEvent, error) {
	newEvent := models.DepJobEvent{
		DepJobID:  DepJob.ID,
		Timestamp: time.Now(),
		Status:    event.Status,
		Message:   event.Message,
		Code:      event.Code,
	}

	err := db.Create(&newEvent).Error
	if err != nil {
		return models.DepJobEvent{}, err
	}
	return newEvent, nil
}

func (d Deployment) unauthOne(c *gin.Context) (models.Deployment, error) {
	dep := models.Deployment{}
	var id string
	if !bindID(c, &id) {
		return dep, errNotFound
	}
	q := db.Preload("DepJob").Preload("DepJob.Events")
	err := q.First(&dep, "deployments.id = ?", id).Error
	return dep, err
}
