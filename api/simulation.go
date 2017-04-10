package api

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/stream"
	"github.com/gin-gonic/gin"
)

type Simulation struct{}

func (s Simulation) Create(c *gin.Context) {
	post := models.PostSimulation{}
	c.BindJSON(&post)

	if !validateRequest(c, post) {
		return
	}
	newSim := models.Simulation{UserID: post.UserID, ProjectID: post.ProjectID}
	db.Create(&newSim)
	successResponse(c, 201, newSim)
}

func (s Simulation) Update(c *gin.Context) {
	post := models.PostSimulation{}
	c.BindJSON(&post)
	var id int
	if !bindId(c, &id) {
		return
	}
	if !validateRequest(c, post) {
		return
	}
	outputsim := models.Simulation{}
	db.Where(&models.Simulation{ID: id}).First(&outputsim)
	sim := models.Simulation{
		UserID:        post.UserID,
		ProjectID:     post.ProjectID,
		InputArtifact: post.InputArtifact,
		OutputStream:  post.OutputStream,
		Status:        post.Status,
	}
	db.Model(&outputsim).Updates(sim)
	successResponse(c, 200, outputsim)
}

func (s Simulation) Input(c *gin.Context) {
	var id int
	if !bindId(c, &id) {
		return
	}
	sim := models.Simulation{}
	db.First(&sim, id)

	if sim.Status != "SUBMITTED" {
		errResponse(c, 400, fmt.Sprintf("Simulation is '%s', not SUBMITTED", sim.Status))
		return
	}

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

	db.Model(&sim).Updates(models.Simulation{BatchId: simId, Status: "QUEUED"})
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
	outputsim := []models.Simulation{}
	var id int
	if !bindId(c, &id) {
		return
	}
	db.Where(&models.Simulation{ID: id}).First(&outputsim)
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

	for !sim.HasStarted() {
		time.Sleep(time.Second)
		db.First(&sim, id)
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
