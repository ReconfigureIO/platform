package api

import (
	"fmt"
	"log"
	"time"

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/stream"
	"github.com/gin-gonic/gin"
	validator "gopkg.in/validator.v2"
)

type Simulation struct{}

func (s Simulation) Create(c *gin.Context) {
	post := models.PostSimulation{}
	c.BindJSON(&post)

	if err := validateSimulation(post, c); err != nil {
		return
	}
	newSim := models.Simulation{UserID: post.UserID, ProjectID: post.ProjectID}
	db.Create(&newSim)
	c.JSON(201, newSim)
}

func (s Simulation) Update(c *gin.Context) {
	post := models.PostSimulation{}
	c.BindJSON(&post)
	if c.Param("id") != "" {
		SimID, err := stringToInt(c.Param("id"), c)
		if err != nil {
			return
		}
		if err := validateSimulation(post, c); err != nil {
			return
		}
		outputsim := models.Simulation{}
		db.Where(&models.Simulation{ID: SimID}).First(&outputsim)
		db.Model(&outputsim).Updates(models.Simulation{UserID: post.UserID, ProjectID: post.ProjectID, InputArtifact: post.InputArtifact, OutputStream: post.OutputStream, Status: post.Status})
		c.JSON(201, outputsim)
	}
}

func (s Simulation) Input(c *gin.Context) {
	id, err := stringToInt(c.Param("id"), c)
	if err != nil {
		return
	}

	sim := models.Simulation{}
	db.First(&sim, id)

	if sim.Status != "SUBMITTED" {
		c.JSON(400, ApiError{
			Error: fmt.Sprintf("Simulation is '%s', not SUBMITTED", sim.Status),
		})
		return
	}

	key := fmt.Sprintf("simulation/%d/simulation.tar.gz", id)

	s3Url, err := awsSession.Upload(key, c.Request.Body, c.Request.ContentLength)

	if err != nil {
		c.AbortWithStatus(500)
		c.Error(err)
		return
	}

	simId, err := awsSession.RunSimulation(s3Url, sim.Command)

	if err != nil {
		c.AbortWithStatus(500)
		c.Error(err)
		return
	}

	db.Model(&sim).Updates(models.Simulation{BatchId: simId, Status: "QUEUED"})
	c.JSON(200, sim)
}

func (s Simulation) List(c *gin.Context) {
	project := c.DefaultQuery("project", "")
	Simulations := []models.Simulation{}
	if project != "" {
		ProjID, err := stringToInt(project, c)
		if err != nil {
			return
		}
		db.Where(&models.Simulation{ProjectID: ProjID}).Find(&Simulations)
	} else {
		db.Find(&Simulations)
	}

	c.JSON(200, gin.H{
		"simulations": Simulations,
	})
}

func (s Simulation) Get(c *gin.Context) {
	outputsim := []models.Simulation{}
	if c.Param("id") != "" {
		simulationID, err := stringToInt(c.Param("id"), c)
		if err != nil {
			return
		}
		db.Where(&models.Simulation{ID: simulationID}).First(&outputsim)
	}
	c.JSON(200, outputsim)
}

func (s Simulation) Logs(c *gin.Context) {
	id, err := stringToInt(c.Param("id"), c)
	if err != nil {
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
		c.AbortWithStatus(500)
		c.Error(err)
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

func validateSimulation(posts models.PostSimulation, c *gin.Context) error {
	if err := validator.Validate(&posts); err != nil {
		c.AbortWithStatus(404)
		return err
	}
	return nil
}
