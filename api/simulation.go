package api

import (
	"fmt"
	"strconv"

	"github.com/ReconfigureIO/platform/auth"
	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/aws"
	. "github.com/ReconfigureIO/platform/sugar"
	"github.com/dchest/uniuri"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

type Simulation struct {
	Aws aws.ServiceInterface
}

func NewSimulation() Simulation {
	return Simulation{Aws: awsSession}
}

func (b Simulation) Query(c *gin.Context) *gorm.DB {
	user := auth.GetUser(c)
	return db.Joins("join projects on projects.id = simulations.project_id").
		Where("projects.user_id=?", user.ID).
		Preload("Project").Preload("BatchJob").Preload("BatchJob.Events")
}

// Get the first simulation by ID, 404 if it doesn't exist
func (s Simulation) ById(c *gin.Context) (models.Simulation, error) {
	sim := models.Simulation{}
	var id int
	if !bindId(c, &id) {
		return sim, errNotFound
	}
	err := s.Query(c).First(&sim, id).Error

	if err != nil {
		NotFoundOrError(c, err)
		return sim, err
	}
	return sim, nil
}

func (s Simulation) UnauthOne(c *gin.Context) (models.Simulation, error) {
	sim := models.Simulation{}
	var id int
	if !bindId(c, &id) {
		return sim, errNotFound
	}
	q := db.Preload("Project").Preload("BatchJob").Preload("BatchJob.Events")
	err := q.First(&sim, id).Error
	return sim, err
}

func (s Simulation) Create(c *gin.Context) {
	post := models.PostSimulation{}
	c.BindJSON(&post)

	if !ValidateRequest(c, post) {
		return
	}

	// Ensure that the project exists, and the user has permissions for it
	project := models.Project{}
	err := Project{}.Query(c).First(&project, post.ProjectID).Error
	if err != nil {
		NotFoundOrError(c, err)
		return
	}

	newSim := models.Simulation{Project: project, Command: post.Command, Token: uniuri.NewLen(64)}
	err = db.Create(&newSim).Error
	if err != nil {
		InternalError(c, err)
		return
	}

	SuccessResponse(c, 201, newSim)
}

func (s Simulation) Input(c *gin.Context) {
	sim, err := s.ById(c)
	if err != nil {
		return
	}

	if sim.Status() != "SUBMITTED" {
		ErrResponse(c, 400, fmt.Sprintf("Simulation is '%s', not SUBMITTED", sim.Status))
		return
	}

	key := fmt.Sprintf("simulation/%d/simulation.tar.gz", sim.ID)

	s3Url, err := s.Aws.Upload(key, c.Request.Body, c.Request.ContentLength)
	if err != nil {
		ErrResponse(c, 500, err)
		return
	}

	callbackUrl := fmt.Sprintf("https://%s/simulations/%d/events?token=%s", c.Request.Host, sim.ID, sim.Token)

	simId, err := s.Aws.RunSimulation(s3Url, callbackUrl, sim.Command)
	if err != nil {
		ErrResponse(c, 500, err)
		return
	}

	err = Transaction(c, func(tx *gorm.DB) error {
		batchJob := BatchService{}.New(simId)
		return tx.Model(&sim).Association("BatchJob").Append(batchJob).Error
	})

	if err != nil {
		return
	}

	SuccessResponse(c, 200, sim)
}

func (s Simulation) List(c *gin.Context) {
	project := c.DefaultQuery("project", "")
	simulations := []models.Simulation{}
	q := s.Query(c)

	if id, err := strconv.Atoi(project); err == nil && project != "" {
		q = q.Where(&models.Simulation{ProjectID: id})
	}

	err := q.Find(&simulations).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		InternalError(c, err)
		return
	}

	SuccessResponse(c, 200, simulations)
}

func (s Simulation) Get(c *gin.Context) {
	sim, err := s.ById(c)
	if err != nil {
		return
	}
	SuccessResponse(c, 200, sim)
}

func (s Simulation) Logs(c *gin.Context) {
	sim, err := s.ById(c)
	if err != nil {
		return
	}

	StreamBatchLogs(s.Aws, c, &sim.BatchJob)
}

func (s Simulation) CanPostEvent(c *gin.Context, sim models.Simulation) bool {
	user, loggedIn := auth.CheckUser(c)
	if loggedIn && sim.Project.UserID == user.ID {
		return true
	}
	token, exists := c.GetQuery("token")
	if exists && sim.Token == token {
		return true
	}
	return false
}

func (s Simulation) CreateEvent(c *gin.Context) {
	sim, err := s.UnauthOne(c)
	if err != nil {
		return
	}

	if !s.CanPostEvent(c, sim) {
		c.AbortWithStatus(403)
		return
	}

	event := models.PostBatchEvent{}
	c.BindJSON(&event)

	if !ValidateRequest(c, event) {
		return
	}

	currentStatus := sim.Status()

	if !models.CanTransition(currentStatus, event.Status) {
		ErrResponse(c, 400, fmt.Sprintf("%s not valid when current status is %s", event.Status, currentStatus))
		return
	}

	newEvent, err := BatchService{}.AddEvent(&sim.BatchJob, event)

	if err != nil {
		c.Error(err)
		ErrResponse(c, 500, nil)
		return
	}

	SuccessResponse(c, 200, newEvent)
}
