package api

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/stream"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

type Simulation struct{}

func Transaction(c *gin.Context, ops func(db *gorm.DB) error) error {
	tx := db.Begin()
	err := ops(tx)
	if err != nil {
		tx.Rollback()
		c.Error(err)
		errResponse(c, 500, nil)
	}
	tx.Commit()
	return err
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
	var id int
	if !bindId(c, &id) {
		return
	}
	sim := models.Simulation{}
	db.First(&sim, id)

	//	if sim.Status != "SUBMITTED" {
	//		errResponse(c, 400, fmt.Sprintf("Simulation is '%s', not SUBMITTED", sim.Status))
	//		return
	//	}

	key := fmt.Sprintf("simulation/%d/simulation.tar.gz", id)

	s3Url, err := awsSession.Upload(key, c.Request.Body, c.Request.ContentLength)
	if err != nil {
		errResponse(c, 500, err)
		return
	}

	callbackUrl := fmt.Sprintf("https://%s/simulations/%d", c.Request.Host, sim.ID)

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
		return tx.Model(&sim).Association("Events").Append(models.SimulationEvent{Timestamp: time.Now(), Status: "QUEUED"}).Error
	})
	if err != nil {
		successResponse(c, 200, sim)
	}
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
	// check for error here
	db.First(&sim, id)

	w := c.Writer
	clientGone := w.CloseNotify()

	for !sim.HasStarted() {
		select {
		case <-clientGone:
			return
		default:
			time.Sleep(time.Second)
			db.First(&sim, id)
		}
	}

	simId := sim.BatchId

	logStream, err := awsSession.GetJobStream(simId)
	if err != nil {
		errResponse(c, 500, err)
		return
	}

	log.Printf("opening log stream: %s", *logStream.LogStreamName)

	lstream := awsSession.NewStream(*logStream)

	go func() {
		for !sim.HasFinished() {
			time.Sleep(10 * time.Second)
			db.First(&sim, id)
		}
		lstream.Ended = true
	}()

	stream.Stream(lstream, c)
}

func (s Simulation) CreateEvent(c *gin.Context) {
	event := models.PostSimulationEvent{}
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
