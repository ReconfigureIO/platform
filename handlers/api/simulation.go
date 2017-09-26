package api

import (
	"fmt"
	"net/http"

	"github.com/ReconfigureIO/platform/middleware"
	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/aws"
	"github.com/ReconfigureIO/platform/service/events"
	"github.com/ReconfigureIO/platform/sugar"
	"github.com/dchest/uniuri"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

const (
	maxConcurrentSimulations = 2 // number of concurrent simulations per user
)

// Simulation handles simulation requests.
type Simulation struct {
	Aws    aws.Service
	Events events.EventService
}

// NewSimulation creates a new Simulation.
func NewSimulation(events events.EventService) Simulation {
	return Simulation{
		Aws:    awsSession,
		Events: events,
	}
}

// Common preload functionality.
func (s Simulation) Preload(db *gorm.DB) *gorm.DB {
	return db.Preload("Project").
		Preload("BatchJob").
		Preload("BatchJob.Events", func(db *gorm.DB) *gorm.DB {
			return db.Order("timestamp ASC")
		})
}

// Query fetches simulations for user and project.
func (s Simulation) Query(c *gin.Context) *gorm.DB {
	user := middleware.GetUser(c)
	joined := db.Joins("join projects on projects.id = simulations.project_id").
		Where("projects.user_id=?", user.ID)
	return s.Preload(joined)
}

// ByID gets the first simulation by ID, 404 if it doesn't exist
func (s Simulation) ByID(c *gin.Context) (models.Simulation, error) {
	sim := models.Simulation{}
	var id string
	if !bindID(c, &id) {
		return sim, errNotFound
	}
	err := s.Query(c).First(&sim, "simulations.id = ?", id).Error

	if err != nil {
		sugar.NotFoundOrError(c, err)
		return sim, err
	}
	return sim, nil
}

func (s Simulation) unauthOne(c *gin.Context) (models.Simulation, error) {
	sim := models.Simulation{}
	var id string
	if !bindID(c, &id) {
		return sim, errNotFound
	}
	q := s.Preload(db)
	err := q.First(&sim, "simulations.id = ?", id).Error
	return sim, err
}

// Create creates a new simulation.
func (s Simulation) Create(c *gin.Context) {
	post := models.PostSimulation{}
	c.BindJSON(&post)

	if !sugar.ValidateRequest(c, post) {
		return
	}

	// Ensure that the project exists, and the user has permissions for it
	project := models.Project{}
	err := Project{}.Query(c).First(&project, "projects.id = ?", post.ProjectID).Error
	if err != nil {
		sugar.NotFoundOrError(c, err)
		return
	}

	// check for number of concurrently running simulations.
	user := middleware.GetUser(c)
	simData := models.SimulationDataSource(db)
	if activeSims, err := simData.ActiveSimulations(user); err != nil {
		sugar.ErrResponse(c, http.StatusInternalServerError, "Error retrieving simulation information")
		return
	} else if len(activeSims) >= maxConcurrentSimulations {
		sugar.ErrResponse(c, http.StatusServiceUnavailable, fmt.Sprintf("Exceeded concurrent simulation max of %d", maxConcurrentSimulations))
		return
	}

	newSim := models.Simulation{Project: project, Command: post.Command, Token: uniuri.NewLen(64)}
	err = db.Create(&newSim).Error
	if err != nil {
		sugar.InternalError(c, err)
		return
	}
	sugar.EnqueueEvent(s.Events, c, "Posted Simulation", map[string]interface{}{"simulation_id": newSim.ID, "project_name": newSim.Project.Name})
	sugar.SuccessResponse(c, 201, newSim)
}

// Input handles input upload for simulation.
func (s Simulation) Input(c *gin.Context) {
	sim, err := s.ByID(c)
	if err != nil {
		return
	}

	if sim.Status() != "SUBMITTED" {
		sugar.ErrResponse(c, 400, fmt.Sprintf("Simulation is '%s', not SUBMITTED", sim.Status()))
		return
	}

	key := fmt.Sprintf("simulation/%s/simulation.tar.gz", sim.ID)

	s3Url, err := s.Aws.Upload(key, c.Request.Body, c.Request.ContentLength)
	if err != nil {
		sugar.ErrResponse(c, 500, err)
		return
	}

	callbackURL := fmt.Sprintf("https://%s/simulations/%s/events?token=%s", c.Request.Host, sim.ID, sim.Token)

	simID, err := s.Aws.RunSimulation(s3Url, callbackURL, sim.Command)
	if err != nil {
		sugar.ErrResponse(c, 500, err)
		return
	}

	err = Transaction(c, func(tx *gorm.DB) error {
		batchJob := BatchService{}.New(simID)
		return tx.Model(&sim).Association("BatchJob").Append(batchJob).Error
	})

	if err != nil {
		return
	}

	sugar.SuccessResponse(c, 200, sim)
}

// List lists all simulations.
func (s Simulation) List(c *gin.Context) {
	project := c.DefaultQuery("project", "")
	simulations := []models.Simulation{}
	q := s.Query(c)

	if project != "" {
		q = q.Where(&models.Simulation{ProjectID: project})
	}

	err := q.Find(&simulations).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		sugar.InternalError(c, err)
		return
	}

	sugar.SuccessResponse(c, 200, simulations)
}

// Get gets a simulation.
func (s Simulation) Get(c *gin.Context) {
	sim, err := s.ByID(c)
	if err != nil {
		return
	}
	sugar.SuccessResponse(c, 200, sim)
}

// Logs stream logs for simulation.
func (s Simulation) Logs(c *gin.Context) {
	sim, err := s.ByID(c)
	if err != nil {
		return
	}

	StreamBatchLogs(s.Aws, c, &sim.BatchJob)
}

func (s Simulation) canPostEvent(c *gin.Context, sim models.Simulation) bool {
	user, loggedIn := middleware.CheckUser(c)
	if loggedIn && sim.Project.UserID == user.ID {
		return true
	}
	token, exists := c.GetQuery("token")
	if exists && sim.Token == token {
		return true
	}
	return false
}

// CreateEvent creates a new event.
func (s Simulation) CreateEvent(c *gin.Context) {
	sim, err := s.unauthOne(c)
	if err != nil {
		return
	}

	if !s.canPostEvent(c, sim) {
		c.AbortWithStatus(403)
		return
	}

	event := models.PostBatchEvent{}
	c.BindJSON(&event)

	if !sugar.ValidateRequest(c, event) {
		return
	}

	currentStatus := sim.Status()

	if !models.CanTransition(currentStatus, event.Status) {
		sugar.ErrResponse(c, 400, fmt.Sprintf("%s not valid when current status is %s", event.Status, currentStatus))
		return
	}

	newEvent, err := BatchService{}.AddEvent(&sim.BatchJob, event)

	if err != nil {
		c.Error(err)
		sugar.ErrResponse(c, 500, nil)
		return
	}

	eventMessage := "Simulation entered state:" + event.Status
	sugar.EnqueueEvent(s.Events, c, eventMessage, map[string]interface{}{"simulation_id": sim.ID, "project_name": sim.Project.Name, "message": event.Message})

	sugar.SuccessResponse(c, 200, newEvent)
}
