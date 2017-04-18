package api

import (
	"strconv"

	"github.com/ReconfigureIO/platform/auth"
	"github.com/ReconfigureIO/platform/models"
	. "github.com/ReconfigureIO/platform/sugar"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

type Deployment struct{}

func (d Deployment) Query(c *gin.Context) *gorm.DB {
	user := auth.GetUser(c)
	return db.Joins("left join builds on builds.project_id = projects.id").Joins("left join deployments on deployments.build_id = builds.id").
		Where("projects.user_id=?", user.ID).
		Preload("Project").Preload("BatchJob").Preload("BatchJob.Events")
}

// Get the first simulation by ID, 404 if it doesn't exist
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
		Status:  "QUEUED",
	}
	err = db.Create(&newDep).Error
	if err != nil {
		InternalError(c, err)
		return
	}
	_, err = awsSession.RunDeployment(newDep.Command)
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
	SuccessResponse(c, 200, "This function does nothing yet")
}

// func (s Simulation) Logs(c *gin.Context) {
// 	sim, err := s.ById(c)
// 	if err != nil {
// 		return
// 	}

// 	StreamBatchLogs(awsSession, c, &sim.BatchJob)
// }

// func (s Simulation) CreateEvent(c *gin.Context) {
// 	sim, err := s.ById(c)
// 	if err != nil {
// 		return
// 	}

// 	event := models.PostBatchEvent{}
// 	c.BindJSON(&event)

// 	if !ValidateRequest(c, event) {
// 		return
// 	}

// 	currentStatus := sim.Status()

// 	if !models.CanTransition(currentStatus, event.Status) {
// 		ErrResponse(c, 400, fmt.Sprintf("%s not valid when current status is %s", event.Status, currentStatus))
// 		return
// 	}

// 	newEvent, err := BatchService{}.AddEvent(&sim.BatchJob, event)

// 	if err != nil {
// 		c.Error(err)
// 		ErrResponse(c, 500, nil)
// 		return
// 	}

// 	SuccessResponse(c, 200, newEvent)
// }
