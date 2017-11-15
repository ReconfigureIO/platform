package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ReconfigureIO/platform/middleware"
	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/events"
	"github.com/ReconfigureIO/platform/service/queue"
	"github.com/ReconfigureIO/platform/sugar"
	"github.com/dchest/uniuri"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

const (
	numQueuedDeployments = 2 // number of concurrent deployments in queue per user
)

// Deployment handles request for deployments.
type Deployment struct {
	Events           events.EventService
	UseSpotInstances bool
}

// Query fetches deployments for user.
func (d Deployment) Query(c *gin.Context) ([]Deployment, error) {
	user := middleware.GetUser(c)
	dds := models.DeploymentDataSource(db)
	deployments, err := dds.GetWithUser(user.ID)
	return deployments, err
}

// ByID gets the first deployment by ID, 404 if it doesn't exist.
func (d Deployment) ByID(c *gin.Context) (models.Deployment, error) {
	dep := models.Deployment{}
	var id string
	if !bindID(c, &id) {
		return dep, errNotFound
	}

	deployments, err := d.Query(c)
	if err != nil {
		sugar.NotFoundOrError(c, err)
		return dep, err
	}
	for i := range deployments {
		if deployments[i].ID == id {
			return deployments[i], nil
		}
	}
	return dep, err
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
	user := middleware.GetUser(c)
	err := Build{}.QueryWhere("projects.id=? OR projects.user_id=?", publicProjectID, user.ID).
		First(&build, "builds.id = ?", post.BuildID).Error
	if err != nil {
		sugar.NotFoundOrError(c, err)
		return
	}

	// Ensure there is enough instance hours
	billingService := Billing{}
	billingHours := billingService.FetchBillingHours(user.ID)
	// considering the complexity in calculating instance hours,
	// a cache would be ideal here.
	// this is not optimal yet :(
	if h, err := billingHours.Net(); err == nil && h <= 0 {
		sugar.ErrResponse(c, http.StatusPaymentRequired, "No available instance hours")
		return
	}

	newDep := models.Deployment{
		Build:        build,
		BuildID:      post.BuildID,
		Command:      post.Command,
		Token:        uniuri.NewLen(64),
		SpotInstance: d.UseSpotInstances,
		UserID:       user.ID,
	}

	// use deployment queue if enabled
	if deploymentQueue != nil {
		// check number of queued deployments owned by user.
		if ad, err := deploymentQueue.CountUserJobsInStatus(user, models.StatusQueued); err != nil {
			sugar.ErrResponse(c, http.StatusInternalServerError, "Error retrieving deployment information")
			return
		} else if ad >= numQueuedDeployments {
			sugar.ErrResponse(c, http.StatusServiceUnavailable, fmt.Sprintf("Exceeded queued deployment max of %d", numQueuedDeployments))
			return
		}

		err = db.Create(&newDep).Error
		if err != nil {
			sugar.InternalError(c, err)
			return
		}

		deploymentQueue.Push(queue.Job{
			ID:     newDep.ID,
			Weight: 2, // TODO prioritize paying customers
		})
	} else {
		dds := models.DeploymentDataSource(db)
		if ad, err := dds.ActiveDeployments(user.ID); err != nil {
			sugar.InternalError(c, err)
			return
		} else if len(ad) >= numQueuedDeployments {
			sugar.ErrResponse(c, http.StatusServiceUnavailable, fmt.Sprintf("Exceeded concurrent deployment max of %d", numQueuedDeployments))
			return
		}

		err = db.Create(&newDep).Error
		if err != nil {
			sugar.InternalError(c, err)
			return
		}

		callbackURL := fmt.Sprintf("https://%s/deployments/%s/events?token=%s", c.Request.Host, newDep.ID, newDep.Token)

		instanceID, err := deploy.RunDeployment(context.Background(), newDep, callbackURL)
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
	}

	sugar.EnqueueEvent(d.Events, c, "Posted Deployment", map[string]interface{}{"deployment_id": newDep.ID, "build_id": newDep.BuildID})

	sugar.SuccessResponse(c, 201, newDep)
}

// List lists all deployments.
func (d Deployment) List(c *gin.Context) {
	build := c.DefaultQuery("build", "")
	project := c.DefaultQuery("project", "")
	deployments := []models.Deployment{}
	deployments, err := d.Query(c)
	if err != nil {
		sugar.InternalError(c, err)
		return
	}

	if project == "" && build == "" {
		sugar.SuccessResponse(c, 200, deployments)
	}

	deploymentsMatchingFilters := []Deployment{}
	for i := range deployments {
		if deployments[i].Build.Project.ID == project && deployments[i].Build.ID == build {
			deploymentsMatchingFilters = append(deploymentsMatchingFilters, deployments[i])
		} else if deployments[i].Build.Project.ID == project {
			deploymentsMatchingFilters = append(deploymentsMatchingFilters, deployments[i])
		} else if deployments[i].Build.ID == build {
			deploymentsMatchingFilters = append(deploymentsMatchingFilters, deployments[i])
		}
	}

	sugar.SuccessResponse(c, 200, deploymentsMatchingFilters)
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
	streamDeploymentLogs(deploy, c, &targetDep)
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

	eventMessage := "Deployment entered state:" + event.Status
	sugar.EnqueueEvent(d.Events, c, eventMessage, map[string]interface{}{"deployment_id": dep.ID, "project_name": dep.Build.Project.Name, "message": event.Message})

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
		err = deploy.StopDeployment(c, dep)
	}

	return newEvent, err
}

func (d Deployment) unauthOne(c *gin.Context) (models.Deployment, error) {
	dep := models.Deployment{}
	var id string
	if !bindID(c, &id) {
		return dep, errNotFound
	}
	deployments, err := d.Query(c)
	if err != nil {
		return dep, err
	}
	for i := range deployments {
		if deployments[i].ID == id {
			return deployments[i], nil
		}
	}
}
