package api

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ReconfigureIO/platform/models"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

type Simulation struct{}

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
	var id int
	if !bindId(c, &id) {
		return
	}
	sim := models.Simulation{}

	err := db.First(&sim, id).Error
	if err != nil {
		internalError(c, err)
		return
	}

	err = db.Model(&sim).Association("Events").Find(&sim.Events).Error
	if err != nil {
		internalError(c, err)
		return
	}

	if sim.Status() != "SUBMITTED" {
		errResponse(c, 400, fmt.Sprintf("Simulation is '%s', not SUBMITTED", sim.Status))
		return
	}

	key := fmt.Sprintf("simulation/%d/simulation.tar.gz", id)

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
		err := tx.Model(&sim).Updates(models.Simulation{BatchId: simId}).Error
		if err != nil {
			return err
		}
		newEvent := models.SimulationEvent{Timestamp: time.Now(), Status: "QUEUED"}
		return tx.Model(&sim).Association("Events").Append(newEvent).Error
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
	outputsim := models.Simulation{}
	var id int
	if !bindId(c, &id) {
		return
	}
	db.Where(&models.Simulation{ID: id}).First(&outputsim)
	db.Model(&outputsim).Association("Events").Find(&outputsim.Events)
	successResponse(c, 200, outputsim)
}

func (s Simulation) Logs(c *gin.Context) {
	var id int
	if !bindId(c, &id) {
		return
	}

	sim := models.Simulation{}
	err := db.First(&sim, id).Error
	if err != nil {
		internalError(c, err)
		return
	}
	refresh := func() error {
		return db.Model(&sim).Association("Events").Find(&sim.Events).Error
	}
	StreamBatchLogs(awsSession, c, &sim, refresh)
}

func (s Simulation) CreateEvent(c *gin.Context) {
	event := models.PostBatchEvent{}
	c.BindJSON(&event)
	var id int
	if !bindId(c, &id) {
		return
	}

	var sim models.Simulation
	err := db.First(&sim, id).Error
	if err != nil {
		c.Error(err)
		errResponse(c, 500, nil)
		return
	}

	if !validateRequest(c, event) {
		return
	}

	db.Model(&sim).Association("Events").Find(&sim.Events)

	currentStatus := sim.Status()

	if !models.CanTransition(currentStatus, event.Status) {
		errResponse(c, 400, fmt.Sprintf("%s not valid when current status is %s", event.Status, currentStatus))
		return
	}

	newEvent := models.SimulationEvent{
		SimulationID: id,
		Timestamp:    time.Now(),
		Status:       event.Status,
		Message:      event.Message,
		Code:         event.Code,
	}
	db.Create(&newEvent)

	if newEvent.Status == models.TERMINATED && len(sim.BatchId) > 0 {
		err = awsSession.HaltJob(sim.BatchId)

		if err != nil {
			c.Error(err)
			errResponse(c, 500, nil)
			return
		}
	}

	successResponse(c, 200, newEvent)
}
