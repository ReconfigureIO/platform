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
	newBuild := models.Build{ProjectID: post.ProjectID}
	db.Create(&newBuild)
	successResponse(c, 201, newBuild)
}

func (b Build) Input(c *gin.Context) {
	var id int
	if !bindId(c, &id) {
		return
	}

	build := models.Build{}

	err := db.First(&build, id).Error
	if err != nil {
		internalError(c, err)
		return
	}

	err = db.Model(&build).Association("Events").Find(&build.Events).Error
	if err != nil {
		internalError(c, err)
		return
	}

	if build.Status() != "SUBMITTED" {
		errResponse(c, 400, fmt.Sprintf("Build is '%s', not SUBMITTED", build.Status))
		return
	}

	key := fmt.Sprintf("builds/%d/simulation.tar.gz", id)

	s3Url, err := awsSession.Upload(key, c.Request.Body, c.Request.ContentLength)
	if err != nil {
		errResponse(c, 500, err)
		return
	}

	callbackUrl := fmt.Sprintf("https://reco-test:ffea108b2166081bcfd03a99c597be78b3cf30de685973d44d3b86480d644264@%s/builds/%d", c.Request.Host, build.ID)

	buildId, err := awsSession.RunBuild(s3Url, callbackUrl)
	if err != nil {
		errResponse(c, 500, err)
		return
	}

	err = Transaction(c, func(tx *gorm.DB) error {
		err := db.Model(&build).Updates(models.Build{BatchId: buildId}).Error
		if err != nil {
			return err
		}
		newEvent := models.BuildEvent{Timestamp: time.Now(), Status: "QUEUED"}
		return db.Model(&build).Association("Events").Append(newEvent).Error
	})
	if err != nil {
		return
	}

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
