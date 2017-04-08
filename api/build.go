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

type Build struct{}

func (b Build) List(c *gin.Context) {
	project := c.DefaultQuery("project", "")
	builds := []models.Build{}
	if project != "" {
		projID, err := strconv.Atoi(project)
		if err != nil {
			errResponse(c, 400, nil)
			return
		}
		db.Where(&models.Build{ProjectID: projID}).Find(&builds)
	} else {
		db.Find(&builds)
	}

	successResponse(c, 200, builds)
}

func (b Build) Get(c *gin.Context) {
	build := models.Build{}
	var id int
	if !bindId(c, &id) {
		return
	}
	db.Where(&models.Build{ID: id}).First(&build)
	successResponse(c, 200, build)
}

func (b Build) Create(c *gin.Context) {
	post := models.PostBuild{}
	c.BindJSON(&post)

	if !validateRequest(c, post) {
		return
	}
	newBuild := models.Build{UserID: post.UserID, ProjectID: post.ProjectID}
	db.Create(&newBuild)
	successResponse(c, 201, newBuild)
}

func (b Build) Update(c *gin.Context) {
	post := models.PostBuild{}
	var id int
	if !bindId(c, &id) {
		return
	}
	c.BindJSON(&post)
	if !validateRequest(c, post) {
		return
	}
	outputbuild := models.Build{}
	db.Where(&models.Build{ID: id}).First(&outputbuild)
	build := models.Build{
		UserID:         post.UserID,
		ProjectID:      post.ProjectID,
		InputArtifact:  post.InputArtifact,
		OutputArtifact: post.OutputArtifact,
		OutputStream:   post.OutputStream,
		Status:         post.Status,
	}
	db.Model(&outputbuild).Updates(build)
	c.JSON(200, outputbuild)
}

func (b Build) Input(c *gin.Context) {
	var id int
	if !bindId(c, &id) {
		return
	}

	build := models.Build{}
	db.First(&build, id)

	if build.Status != "SUBMITTED" {
		errResponse(c, 400, fmt.Sprintf("Build is '%s', not SUBMITTED", build.Status))
		return
	}

	key := fmt.Sprintf("builds/%d/simulation.tar.gz", id)

	s3Url, err := awsSession.Upload(key, c.Request.Body, c.Request.ContentLength)
	if err != nil {
		errResponse(c, 500, err)
		return
	}

	buildId, err := awsSession.RunBuild(s3Url)
	if err != nil {
		errResponse(c, 500, err)
		return
	}

	db.Model(&build).Updates(models.Build{BatchId: buildId, Status: "QUEUED"})
	successResponse(c, 200, build)
}

func (b Build) Logs(c *gin.Context) {
	var id int
	if !bindId(c, &id) {
		return
	}

	build := models.Build{}
	// check for error here
	db.First(&build, id)

	for !build.HasStarted() {
		time.Sleep(time.Second)
		db.First(&build, id)
	}

	buildId := build.BatchId

	logStream, err := awsSession.GetJobStream(buildId)
	if err != nil {
		errResponse(c, 500, err)
		return
	}

	log.Printf("opening log stream: %s", *logStream.LogStreamName)

	lstream := awsSession.NewStream(*logStream)

	go func() {
		for !build.HasFinished() {
			time.Sleep(10 * time.Second)
			db.First(&build, id)
		}
		lstream.Ended = true
	}()

	stream.Stream(lstream, c)
}
