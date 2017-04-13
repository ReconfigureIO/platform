package api

import (
	"log"
	"strconv"
	"time"

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/stream"
	"github.com/gin-gonic/gin"
)

type Deployment struct{}

func (d Deployment) Create(c *gin.Context) {
	post := models.PostDeployment{}
	c.BindJSON(&post)

	if !validateRequest(c, post) {
		return
	}
	parentbuild := models.Build{}
	db.Where(&models.Build{ID: post.BuildID}).First(&parentbuild)

	newDep := models.Deployment{
		BuildID:       post.BuildID,
		InputArtifact: parentbuild.OutputArtifact,
		Command:       post.Command,
		Status:        "QUEUED",
	}
	db.Create(&newDep)
	_, err := awsSession.RunDeployment(newDep.InputArtifact, newDep.Command)
	if err != nil {
		errResponse(c, 500, err)
		return
	}
	successResponse(c, 201, newDep)
}

func (d Deployment) List(c *gin.Context) {
	build := c.DefaultQuery("build", "")
	deployments := []models.Deployment{}
	if id, err := strconv.Atoi(build); err == nil && build != "" {
		db.Where(&models.Deployment{BuildID: id}).Find(&deployments)
	} else {
		db.Find(&deployments)
	}

	successResponse(c, 200, deployments)
}

func (d Deployment) Get(c *gin.Context) {
	outputdep := []models.Deployment{}
	var id int
	if !bindId(c, &id) {
		return
	}
	db.Where(&models.Deployment{ID: id}).First(&outputdep)
	successResponse(c, 200, outputdep)
}

func (d Deployment) Logs(c *gin.Context) {
	var id int
	if !bindId(c, &id) {
		return
	}
	dep := models.Deployment{}
	// check for error here
	db.First(&dep, id)

	for !dep.HasStarted() {
		time.Sleep(time.Second)
		db.First(&dep, id)
	}

	depId := dep.BatchId

	logStream, err := awsSession.GetJobStream(depId)
	if err != nil {
		errResponse(c, 500, err)
		return
	}

	log.Printf("opening log stream: %s", *logStream.LogStreamName)

	lstream := awsSession.NewStream(*logStream)

	go func() {
		for !dep.HasFinished() {
			time.Sleep(10 * time.Second)
			db.First(&dep, id)
		}
		lstream.Ended = true
	}()

	stream.Stream(lstream, c)
}
