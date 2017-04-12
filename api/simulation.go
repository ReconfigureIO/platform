package api

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/ReconfigureIO/platform/models"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

type Simulation struct{}

// Get the first simulation by ID, 404 if it doesn't exist
func (s Simulation) ById(c *gin.Context) (models.Simulation, error) {
	sim := models.Simulation{}
	var id int
	if !bindId(c, &id) {
		errResponse(c, 404, nil)
		return sim, nil
	}
	q := db.Preload("BatchJob").Preload("BatchJob.Events").First(&sim, id)
	err := q.Error
	if err != nil {
		internalError(c, err)
		return sim, err
	}
	// check if it didn't come back
	if sim.ID == 0 {
		errResponse(c, 404, nil)
		return sim, errors.New("Not Found")
	}
	return sim, nil
}

func (s Simulation) Create(c *gin.Context) {
	post := models.PostSimulation{}
	c.BindJSON(&post)

	if !validateRequest(c, post) {
		return
	}

	newSim := models.Simulation{ProjectID: post.ProjectID, Command: post.Command}
	db.Create(&newSim)
	successResponse(c, 201, newSim)
}

func (s Simulation) Input(c *gin.Context) {
	sim, err := s.ById(c)
	if err != nil {
		return
	}

	if sim.Status() != "SUBMITTED" {
		errResponse(c, 400, fmt.Sprintf("Simulation is '%s', not SUBMITTED", sim.Status))
		return
	}

	key := fmt.Sprintf("simulation/%d/simulation.tar.gz", sim.ID)

	s3Url, err := awsSession.Upload(key, c.Request.Body, c.Request.ContentLength)
	if err != nil {
		errResponse(c, 500, err)
		return
	}

	callbackUrl := fmt.Sprintf("https://reco-test:ffea108b2166081bcfd03a99c597be78b3cf30de685973d44d3b86480d644264@%s/simulations/%d", c.Request.Host, sim.ID)

	simId, err := awsSession.RunSimulation(s3Url, callbackUrl, sim.Command)
	if err != nil {
		errResponse(c, 500, err)
		return
	}

	err = Transaction(c, func(tx *gorm.DB) error {
		batchJob := BatchService{}.New(simId, tx)
		return tx.Model(&sim).Association("BatchJob").Append(batchJob).Error
	})

	if err != nil {
		return
	}

	successResponse(c, 200, sim)
}

func (s Simulation) List(c *gin.Context) {
	project := c.DefaultQuery("project", "")
	simulations := []models.Simulation{}
	if id, err := strconv.Atoi(project); err == nil && project != "" {
		db.Where(&models.Simulation{ProjectID: id}).Find(&simulations)
	} else {
		db.Find(&simulations)
	}

	successResponse(c, 200, simulations)
}

func (s Simulation) Get(c *gin.Context) {
	sim, err := s.ById(c)
	if err != nil {
		return
	}
	successResponse(c, 200, sim)
}

func (s Simulation) Logs(c *gin.Context) {
	sim, err := s.ById(c)
	if err != nil {
		return
	}

	StreamBatchLogs(awsSession, c, &sim.BatchJob)
}

func (s Simulation) CreateEvent(c *gin.Context) {
	sim, err := s.ById(c)
	if err != nil {
		return
	}

	event := models.PostBatchEvent{}
	c.BindJSON(&event)

	if !validateRequest(c, event) {
		return
	}

	currentStatus := sim.Status()

	if !models.CanTransition(currentStatus, event.Status) {
		errResponse(c, 400, fmt.Sprintf("%s not valid when current status is %s", event.Status, currentStatus))
		return
	}

	newEvent, err := BatchService{}.AddEvent(&sim.BatchJob, event)

	if err != nil {
		c.Error(err)
		errResponse(c, 500, nil)
		return
	}

	successResponse(c, 200, newEvent)
}
