package api

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ReconfigureIO/platform/models"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
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
	err := db.First(&build, id).Error
	if err != nil {
		internalError(c, err)
		return
	}
	refresh := func() error {
		return db.Model(&build).Association("Events").Find(&build.Events).Error
	}
	StreamBatchLogs(awsSession, c, &build, refresh)
}

func (s Build) CreateEvent(c *gin.Context) {
	event := models.PostBatchEvent{}
	c.BindJSON(&event)
	var id int
	if !bindId(c, &id) {
		return
	}

	var build models.Build
	err := db.First(&build, id).Error
	if err != nil {
		c.Error(err)
		errResponse(c, 500, nil)
		return
	}

	if !validateRequest(c, event) {
		return
	}

	db.Model(&build).Association("Events").Find(&build.Events)

	currentStatus := build.Status()

	if !models.CanTransition(currentStatus, event.Status) {
		errResponse(c, 400, fmt.Sprintf("%s not valid when current status is %s", event.Status, currentStatus))
		return
	}

	newEvent := models.BuildEvent{
		BuildID:   id,
		Timestamp: time.Now(),
		Status:    event.Status,
		Message:   event.Message,
		Code:      event.Code,
	}
	db.Create(&newEvent)

	if newEvent.Status == models.TERMINATED && len(build.BatchId) > 0 {
		err = awsSession.HaltJob(build.BatchId)

		if err != nil {
			c.Error(err)
			errResponse(c, 500, nil)
			return
		}
	}

	successResponse(c, 200, newEvent)
}
