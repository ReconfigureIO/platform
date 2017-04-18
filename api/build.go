package api

import (
	"fmt"
	"strconv"

	"github.com/ReconfigureIO/platform/auth"
	"github.com/ReconfigureIO/platform/models"
	. "github.com/ReconfigureIO/platform/sugar"
	"github.com/dchest/uniuri"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

type Build struct{}

func (b Build) Query(c *gin.Context) *gorm.DB {
	user := auth.GetUser(c)
	return db.Joins("join projects on projects.id = builds.project_id").
		Where("projects.user_id=?", user.ID).
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
		NotFoundOrError(c, err)
		return build, err
	}
	return build, nil
}

func (b Build) UnauthOne(c *gin.Context) (models.Build, error) {
	build := models.Build{}
	var id int
	if !bindId(c, &id) {
		return build, errNotFound
	}
	q := db.Preload("Project").Preload("BatchJob").Preload("BatchJob.Events")
	err := q.First(&build, id).Error
	return build, err
}

func (b Build) List(c *gin.Context) {
	project := c.DefaultQuery("project", "")
	builds := []models.Build{}
	q := b.Query(c)

	if project != "" {
		projID, err := strconv.Atoi(project)
		if err != nil {
			ErrResponse(c, 400, nil)
			return
		}
		q = q.Where(&models.Build{ProjectID: projID})
	}

	err := q.Find(&builds).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		InternalError(c, err)
		return
	}

	SuccessResponse(c, 200, builds)
}

func (b Build) Get(c *gin.Context) {
	build, err := b.ById(c)
	if err != nil {
		return
	}

	SuccessResponse(c, 200, build)
}

func (b Build) Create(c *gin.Context) {
	post := models.PostBuild{}
	c.BindJSON(&post)

	if !ValidateRequest(c, post) {
		return
	}
	// Ensure that the project exists, and the user has permissions for it
	project := models.Project{}
	err := Project{}.Query(c).First(&project, post.ProjectID).Error
	if err != nil {
		NotFoundOrError(c, err)
		return
	}

	newBuild := models.Build{Project: project, Token: uniuri.NewLen(64)}
	db.Create(&newBuild)
	SuccessResponse(c, 201, newBuild)
}

func (b Build) Input(c *gin.Context) {
	build, err := b.ById(c)
	if err != nil {
		return
	}

	if build.Status() != "SUBMITTED" {
		ErrResponse(c, 400, fmt.Sprintf("Build is '%s', not SUBMITTED", build.Status))
		return
	}

	key := fmt.Sprintf("builds/%d/simulation.tar.gz", build.ID)

	s3Url, err := awsSession.Upload(key, c.Request.Body, c.Request.ContentLength)
	if err != nil {
		ErrResponse(c, 500, err)
		return
	}
	callbackUrl := fmt.Sprintf("https://%s/builds/%d/events?token=%s", c.Request.Host, build.ID, build.Token)
	buildId, err := awsSession.RunBuild(s3Url, callbackUrl)
	if err != nil {
		ErrResponse(c, 500, err)
		return
	}

	err = Transaction(c, func(tx *gorm.DB) error {
		batchJob := BatchService{}.New(buildId)
		return tx.Model(&build).Association("BatchJob").Append(batchJob).Error
	})

	if err != nil {
		return
	}

	SuccessResponse(c, 200, build)
}

func (b Build) Logs(c *gin.Context) {
	build, err := b.ById(c)
	if err != nil {
		return
	}

	StreamBatchLogs(awsSession, c, &build.BatchJob)
}

func (b Build) CanPostEvent(c *gin.Context, build models.Build) bool {
	user, loggedIn := auth.CheckUser(c)
	if loggedIn && build.Project.UserID == user.ID {
		return true
	}
	token, exists := c.GetQuery("token")
	if exists && build.Token == token {
		return true
	}
	return false
}

func (b Build) CreateEvent(c *gin.Context) {
	build, err := b.UnauthOne(c)
	if err != nil {
		return
	}

	if !b.CanPostEvent(c, build) {
		c.AbortWithStatus(403)
		return
	}

	event := models.PostBatchEvent{}
	c.BindJSON(&event)

	if !ValidateRequest(c, event) {
		return
	}

	currentStatus := build.Status()

	if !models.CanTransition(currentStatus, event.Status) {
		ErrResponse(c, 400, fmt.Sprintf("%s not valid when current status is %s", event.Status, currentStatus))
		return
	}

	newEvent, err := BatchService{}.AddEvent(&build.BatchJob, event)

	if err != nil {
		c.Error(err)
		ErrResponse(c, 500, nil)
		return
	}

	SuccessResponse(c, 200, newEvent)

}
