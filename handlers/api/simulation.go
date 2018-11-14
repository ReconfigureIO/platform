package api

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"net/url"

	"github.com/ReconfigureIO/platform/middleware"
	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/batch"
	"github.com/ReconfigureIO/platform/service/events"
	"github.com/ReconfigureIO/platform/service/storage"
	"github.com/ReconfigureIO/platform/sugar"
	"github.com/dchest/uniuri"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

// Simulation handles simulation requests.
type Simulation struct {
	APIBaseURL url.URL
	AWS     batch.Service
	Events  events.EventService
	Storage storage.Service
	Repo    models.SimulationRepo
}

// NewSimulation creates a new Simulation.
func NewSimulation(APIBaseURL url.URL, events events.EventService, storageService storage.Service, awsSession batch.Service, repo models.SimulationRepo) Simulation {
	return Simulation{
		APIBaseURL: APIBaseURL,
    AWS:     awsSession,
		Events:  events,
		Storage: storageService,
		Repo:    repo,
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

	urlEvents := s.APIBaseURL
	urlEvents.Query().Set("token", sim.Token)
	urlEvents.Path = "/simulations/" + sim.ID + "/events"

	simID, err := s.AWS.RunSimulation(s3Url, urlEvents.String(), sim.Command)
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

// isTokenAuthorized handles authentication and authorization for workers. On a
// job's (e.g. simulation) creation it is given a token which is also given to
// the worker that processes the job. When the worker sends events or reports to
// the API it includes this token in the request.
func isTokenAuthorized(c *gin.Context, correctToken string) bool {
	gotToken, ok := c.GetQuery("token")
	return ok && subtle.ConstantTimeCompare([]byte(gotToken), []byte(correctToken)) == 1
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
		return
	}

	newEvent, err := BatchService{AWS: s.AWS}.AddEvent(&sim.BatchJob, event)

	if err != nil {
		sugar.InternalError(c, err)
		return
	}

	eventMessage := "Simulation entered state:" + event.Status
	sugar.EnqueueEvent(s.Events, c, eventMessage, sim.Project.UserID, map[string]interface{}{"simulation_id": sim.ID, "project_name": sim.Project.Name, "message": event.Message})

	sugar.SuccessResponse(c, 200, newEvent)
}

// Report fetches a simulation's report.
func (s Simulation) Report(c *gin.Context) {
	user := middleware.GetUser(c)
	var id string
	if !bindID(c, &id) {
		sugar.ErrResponse(c, 404, nil)
		return
	}
	sim, err := s.Repo.ByIDForUser(id, user.ID)
	if err != nil {
		sugar.NotFoundOrError(c, err)
		return
	}

	report, err := s.Repo.GetReport(sim.ID)
	if err != nil {
		sugar.NotFoundOrError(c, err)
		return
	}

	sugar.SuccessResponse(c, 200, report)
}

// CreateReport creates simulation report.
func (s Simulation) CreateReport(c *gin.Context) {
	var id string
	if !bindID(c, &id) {
		sugar.ErrResponse(c, 404, nil)
		return
	}
	sim, err := s.Repo.ByID(id)
	if err != nil {
		sugar.NotFoundOrError(c, err)
		return
	}

	if !isTokenAuthorized(c, sim.Token) {
		c.AbortWithStatus(403)
		return
	}

	if c.ContentType() != "application/vnd.reconfigure.io/reports-v1+json" {
		err = errors.New("Not a valid report version")
		sugar.ErrResponse(c, 400, err)
		return
	}

	var report models.Report
	err = c.BindJSON(&report)
	if err != nil {
		sugar.ErrResponse(c, 500, err)
		return
	}

	err = s.Repo.StoreReport(sim.ID, report)
	if err != nil {
		sugar.ErrResponse(c, 500, err)
		return
	}

	sugar.SuccessResponse(c, 200, nil)
}
