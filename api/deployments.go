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

type Deployment struct{}

func (d Deployment) Create(c *gin.Context) {
	post := models.PostDeployment{}
	c.BindJSON(&post)

	if !validateRequest(c, post) {
		return
	}
	newDep := models.Deployment{ProjectID: post.ProjectID}
	db.Create(&newDep)
	successResponse(c, 201, newDep)
}

func (d Deployment) Update(c *gin.Context) {
	post := models.PostDeployment{}
	c.BindJSON(&post)
	var id int
	if !bindId(c, &id) {
		return
	}
	if !validateRequest(c, post) {
		return
	}
	outputdep := models.Deployment{}
	db.Where(&models.Deployment{ID: id}).First(&outputdep)
	dep := models.Deployment{
		ProjectID:     post.ProjectID,
		InputArtifact: post.InputArtifact,
		OutputStream:  post.OutputStream,
		Status:        post.Status,
	}
	db.Model(&outputdep).Updates(dep)
	successResponse(c, 200, outputdep)
}

func (d Deployment) Input(c *gin.Context) {
	var id int
	if !bindId(c, &id) {
		return
	}
	dep := models.Deployment{}
	db.First(&dep, id)

	if dep.Status != "SUBMITTED" {
		errResponse(c, 400, fmt.Sprintf("Deployment is '%s', not SUBMITTED", dep.Status))
		return
	}

	key := fmt.Sprintf("Deployment/%d/Deployment.tar.gz", id)

	s3Url, err := awsSession.Upload(key, c.Request.Body, c.Request.ContentLength)
	if err != nil {
		errResponse(c, 500, err)
		return
	}

	depId, err := awsSession.RunDeployment(s3Url, dep.Command)
	if err != nil {
		errResponse(c, 500, err)
		return
	}

	db.Model(&dep).Updates(models.Deployment{BatchId: depId, Status: "QUEUED"})
	successResponse(c, 200, dep)
}

func (d Deployment) List(c *gin.Context) {
	project := c.DefaultQuery("project", "")
	Deployments := []models.Deployment{}
	if id, err := strconv.Atoi(project); err == nil && project != "" {
		db.Where(&models.Deployment{ProjectID: id}).Find(&Deployments)
	} else {
		db.Find(&Deployments)
	}

	successResponse(c, 200, Deployments)
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
