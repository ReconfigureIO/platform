package api

import (
	"fmt"
	"strconv"

	"github.com/ReconfigureIO/platform/models"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

type Build struct{}

func (b Build) Query(c *gin.Context) *gorm.DB {
	session := sessions.Default(c)
	user_id := session.Get("user_id")
	u := 0
	if user_id != nil {
		u = user_id.(int)
	}
	return db.Joins("join projects on projects.id = builds.project_id").
		Where("projects.user_id=?", u).
		Preload("Project").Preload("BatchJob").Preload("BatchJob.Events")
}

// Get the first build by ID, 404 if it doesn't exist
func (b Build) ById(c *gin.Context) (models.Build, error) {
	build := models.Build{}
	var id int
	if !bindId(c, &id) {
		return build, errNotFound
	}
	err := b.Query(c).First(&build, id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			errResponse(c, 404, nil)
		} else {
			internalError(c, err)
		}
		return build, err
	}
	return build, nil
}

func (b Build) List(c *gin.Context) {
	project := c.DefaultQuery("project", "")
	builds := []models.Build{}
	q := b.Query(c)

	if project != "" {
		projID, err := strconv.Atoi(project)
		if err != nil {
			errResponse(c, 400, nil)
			return
		}
		q = q.Where(&models.Build{ProjectID: projID})
	}

	err := q.Find(&builds).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		internalError(c, err)
		return
	}

	successResponse(c, 200, builds)
}

func (b Build) Get(c *gin.Context) {
	build, err := b.ById(c)
	if err != nil {
		return
	}
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
	build, err := b.ById(c)
	if err != nil {
		return
	}

	if build.Status() != "SUBMITTED" {
		errResponse(c, 400, fmt.Sprintf("Build is '%s', not SUBMITTED", build.Status))
		return
	}

	key := fmt.Sprintf("builds/%d/simulation.tar.gz", build.ID)

	s3Url, err := awsSession.Upload(key, c.Request.Body, c.Request.ContentLength)
	if err != nil {
		errResponse(c, 500, err)
		return
	}
	callbackUrl := fmt.Sprintf("https://reco-test:ffea108b2166081bcfd03a99c597be78b3cf30de685973d44d3b86480d644264@%s/builds/%d/events", c.Request.Host, build.ID)
	buildId, err := awsSession.RunBuild(s3Url, callbackUrl)
	if err != nil {
		errResponse(c, 500, err)
		return
	}

	err = Transaction(c, func(tx *gorm.DB) error {
		batchJob := BatchService{}.New(buildId)
		return tx.Model(&build).Association("BatchJob").Append(batchJob).Error
	})

	if err != nil {
		return
	}

	successResponse(c, 200, build)
}

func (b Build) Logs(c *gin.Context) {
	build, err := b.ById(c)
	if err != nil {
		return
	}

	StreamBatchLogs(awsSession, c, &build.BatchJob)
}

func (b Build) CreateEvent(c *gin.Context) {
	build, err := b.ById(c)
	if err != nil {
		return
	}

	event := models.PostBatchEvent{}
	c.BindJSON(&event)

	if !validateRequest(c, event) {
		return
	}

	currentStatus := build.Status()

	if !models.CanTransition(currentStatus, event.Status) {
		errResponse(c, 400, fmt.Sprintf("%s not valid when current status is %s", event.Status, currentStatus))
		return
	}

	newEvent, err := BatchService{}.AddEvent(&build.BatchJob, event)

	if err != nil {
		c.Error(err)
		errResponse(c, 500, nil)
		return
	}

	successResponse(c, 200, newEvent)

}
