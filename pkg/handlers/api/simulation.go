package api

import (
	"fmt"

	"github.com/ReconfigureIO/platform/pkg/middleware"
	"github.com/ReconfigureIO/platform/pkg/models"
	"github.com/ReconfigureIO/platform/pkg/sugar"
	"github.com/ReconfigureIO/platform/service/aws"
	"github.com/ReconfigureIO/platform/service/events"
	"github.com/ReconfigureIO/platform/service/storage"
	"github.com/dchest/uniuri"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

// Simulation handles simulation requests.
type Simulation struct {
	AWS     aws.Service
	Events  events.EventService
	Storage storage.Service
}

// NewSimulation creates a new Simulation.
func NewSimulation(events events.EventService, storageService storage.Service, awsSession aws.Service) Simulation {
	return Simulation{
		AWS:     awsSession,
		Events:  events,
		Storage: storageService,
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

	newSim := models.Simulation{Project: project, Command: post.Command, Token: uniuri.NewLen(64)}
	err = db.Create(&newSim).Error
	if err != nil {
		sugar.InternalError(c, err)
		return
	}
	sugar.EnqueueEvent(s.Events, c, "Posted Simulation", project.UserID, map[string]interface{}{"simulation_id": newSim.ID, "project_name": newSim.Project.Name})
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

	s3Url, err := s.Storage.Upload(key, c.Request.Body)
	if err != nil {
		sugar.InternalError(c, err)
		return
	}

	callbackURL := fmt.Sprintf("https://%s/simulations/%s/events?token=%s", c.Request.Host, sim.ID, sim.Token)

	simID, err := s.AWS.RunSimulation(s3Url, callbackURL, sim.Command)
	if err != nil {
		sugar.InternalError(c, err)
		return
	}

	err = Transaction(c, func(tx *gorm.DB) error {
		batchJob := BatchService{AWS: s.AWS}.New(simID)
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

	StreamBatchLogs(s.AWS, c, &sim.BatchJob)
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

	_, isUser := middleware.CheckUser(c)
	if event.Status == models.StatusTerminated && isUser {
		sugar.ErrResponse(c, 400, fmt.Sprintf("Users cannot post TERMINATED events, please upgrade to reco v0.3.1 or above"))
	}

	newEvent, err := BatchService{AWS: s.AWS}.AddEvent(&sim.BatchJob, event)

	if err != nil {
		sugar.InternalError(c, nil)
		return
	}

	eventMessage := "Simulation entered state:" + event.Status
	sugar.EnqueueEvent(s.Events, c, eventMessage, sim.Project.UserID, map[string]interface{}{"simulation_id": sim.ID, "project_name": sim.Project.Name, "message": event.Message})

	sugar.SuccessResponse(c, 200, newEvent)
}
