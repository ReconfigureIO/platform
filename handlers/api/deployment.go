package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ReconfigureIO/platform/middleware"
	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/sugar"
	"github.com/dchest/uniuri"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

// Deployment handles request for deployments.
type Deployment struct{}

// Preload is common preload functionality.
func (d Deployment) Preload(db *gorm.DB) *gorm.DB {
	return db.Preload("Build").Preload("Build.Project").
		Preload("Events", func(db *gorm.DB) *gorm.DB {
			return db.Order("timestamp ASC")
		})
}

// Query fetches deployment for user and project.
func (d Deployment) Query(c *gin.Context) *gorm.DB {
	user := middleware.GetUser(c)
	joined := db.Joins("left join builds on builds.id = deployments.build_id").Joins("left join projects on projects.id = builds.project_id").
		Where("projects.user_id=?", user.ID)
	return d.Preload(joined)
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

	// Ensure there is enough instance hours
	user := middleware.GetUser(c)
	billingHours := FetchBillingHours(user.ID)
	// considering the complexity in calculating instance hours,
	// a cache would be ideal here.
	// this is not optimal yet :(
	if h, err := billingHours.Net(); err == nil && h <= 0 {
		sugar.ErrResponse(c, http.StatusUnauthorized, "No available instance hours")
		return
	}

	newDep := models.Deployment{
		Build:   build,
		BuildID: post.BuildID,
		Command: post.Command,
		Token:   uniuri.NewLen(64),
	}

	err = db.Create(&newDep).Error
	if err != nil {
		sugar.InternalError(c, err)
		return
	}

	callbackURL := fmt.Sprintf("https://%s/deployments/%s/events?token=%s", c.Request.Host, newDep.ID, newDep.Token)

	instanceID, err := mockDeploy.RunDeployment(context.Background(), newDep, callbackURL)
	if err != nil {
		sugar.InternalError(c, err)
		return
	}

	err = db.Model(&newDep).Update("InstanceID", instanceID).Error

	if err != nil {
		sugar.InternalError(c, err)
		return
	}

	newEvent := models.DeploymentEvent{Timestamp: time.Now(), Status: "QUEUED"}
	err = db.Model(&newDep).Association("Events").Append(newEvent).Error

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
	outputDep, err := d.ByID(c)
	if err != nil {
		return
	}
	sugar.SuccessResponse(c, 200, outputDep)
}

// Logs stream logs for deployments.
func (d Deployment) Logs(c *gin.Context) {
	targetDep, err := d.ByID(c)
	if err != nil {
		return
	}
	streamDeploymentLogs(mockDeploy, c, &targetDep)
}

func (d Deployment) canPostEvent(c *gin.Context, dep models.Deployment) bool {
	user, loggedIn := middleware.CheckUser(c)
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

	newEvent, err := d.AddEvent(c, dep, event)

	if err != nil {
		c.Error(err)
		sugar.ErrResponse(c, 500, nil)
		return
	}

	sugar.SuccessResponse(c, 200, newEvent)
}

// AddEvent adds a deployment event.
func (d Deployment) AddEvent(c *gin.Context, dep models.Deployment, event models.PostDepEvent) (models.DeploymentEvent, error) {
	newEvent := models.DeploymentEvent{
		DeploymentID: dep.ID,
		Timestamp:    time.Now(),
		Status:       event.Status,
		Message:      event.Message,
		Code:         event.Code,
	}

	err := db.Create(&newEvent).Error
	if err != nil {
		return models.DeploymentEvent{}, err
	}

	if event.Status == "TERMINATING" {
		err = mockDeploy.StopDeployment(c, dep)
	}

	return newEvent, err
}

func (d Deployment) unauthOne(c *gin.Context) (models.Deployment, error) {
	dep := models.Deployment{}
	var id string
	if !bindID(c, &id) {
		return dep, errNotFound
	}
	q := d.Preload(db)
	err := q.First(&dep, "deployments.id = ?", id).Error
	return dep, err
}
