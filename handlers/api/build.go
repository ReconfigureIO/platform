package api

import (
	"errors"
	"fmt"

	"github.com/ReconfigureIO/platform/service/aws"
	"github.com/ReconfigureIO/platform/service/storage"

	"github.com/ReconfigureIO/platform/middleware"
	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/events"
	"github.com/ReconfigureIO/platform/sugar"
	"github.com/dchest/uniuri"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

// Build handles requests for builds.
type Build struct {
	Events          events.EventService
	Storage         storage.Service
	Aws             aws.Service
	PublicProjectID string
}

// Common preload functionality.
func (b Build) Preload(db *gorm.DB) *gorm.DB {
	return db.Preload("Project").
		Preload("BatchJob").
		Preload("BatchJob.Events", func(db *gorm.DB) *gorm.DB {
			return db.Order("timestamp ASC")
		})
}

// Query fetches builds for user and project.
func (b Build) Query(c *gin.Context) *gorm.DB {
	user := middleware.GetUser(c)
	joined := db.Joins("join projects on projects.id = builds.project_id").
		Where("projects.user_id=?", user.ID)
	return b.Preload(joined)
}

// QueryWhere is like Query but accepts custom where clause.
func (b Build) QueryWhere(where ...interface{}) *gorm.DB {
	joined := db.Joins("join projects on projects.id = builds.project_id")
	if len(where) > 0 {
		joined = joined.Where(where[0], where[1:]...)
	}
	return b.Preload(joined)
}

// ByID gets the first build by ID, 404 if it doesn't exist.
func (b Build) ByID(c *gin.Context) (models.Build, error) {
	build := models.Build{}
	var id string
	if !bindID(c, &id) {
		return build, errNotFound
	}
	err := b.Query(c).First(&build, "builds.id = ?", id).Error
	// Not found? Might be a public build ID
	if err == gorm.ErrRecordNotFound {
		err = b.QueryWhere("projects.id=?", b.PublicProjectID).
			Where(&models.Build{ProjectID: b.PublicProjectID}).First(&build, "builds.id = ?", id).Error
	}

	if err != nil {
		sugar.NotFoundOrError(c, err)
		return build, err
	}
	return build, nil
}

func (b Build) unauthOne(c *gin.Context) (models.Build, error) {
	build := models.Build{}
	var id string
	if !bindID(c, &id) {
		return build, errNotFound
	}
	q := b.Preload(db)
	err := q.First(&build, "id = ?", id).Error
	return build, err
}

// List lists all builds.
func (b Build) List(c *gin.Context) {
	project := c.DefaultQuery("project", "")
	public := c.DefaultQuery("public", "")
	builds := []models.Build{}
	var err error

	if public == "true" {
		builds, err = b.publicBuilds()
	} else {
		builds, err = b.userBuilds(c, project)
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		sugar.InternalError(c, err)
		return
	}

	sugar.SuccessResponse(c, 200, builds)
}

func (b Build) userBuilds(c *gin.Context, project string) (builds []models.Build, err error) {
	q := b.Query(c)

	if project != "" {
		q = q.Where(&models.Build{ProjectID: project})
	}

	err = q.Find(&builds).Error
	return
}

func (b Build) publicBuilds() (builds []models.Build, err error) {
	if b.PublicProjectID == "" {
		err = errors.New("global project configuration missing")
		return
	}

	q := b.QueryWhere("projects.id=?", b.PublicProjectID)

	err = q.Find(&builds).Error
	return
}

// Report fetches a build's report.
func (b Build) Report(c *gin.Context) {
	buildRepo := models.BuildDataSource(db)
	build, err := b.ByID(c)
	if err != nil {
		return
	}

	report, err := buildRepo.GetBuildReport(build)

	if err != nil {
		sugar.NotFoundOrError(c, err)
		return
	}

	sugar.SuccessResponse(c, 200, report)
}

// Get fetches a build.
func (b Build) Get(c *gin.Context) {
	build, err := b.ByID(c)
	if err != nil {
		return
	}

	sugar.SuccessResponse(c, 200, build)
}

// Create creates a build.
func (b Build) Create(c *gin.Context) {
	post := models.PostBuild{}
	c.BindJSON(&post)

	if !sugar.ValidateRequest(c, post) {
		return
	}
	// Ensure that the project exists, and the user has permissions for it
	project := models.Project{}
	err := Project{}.Query(c).First(&project, "projects.id = ?", post.ProjectID).Error
	if err != nil {
		sugar.NotFoundOrError(c, err)
		return
	}

	newBuild := models.Build{Project: project, Token: uniuri.NewLen(64)}
	if err := db.Create(&newBuild).Error; err != nil {
		sugar.InternalError(c, err)
		return
	}
	sugar.EnqueueEvent(b.Events, c, "Posted Build", project.UserID, map[string]interface{}{"build_id": newBuild.ID, "project_name": newBuild.Project.Name})
	sugar.SuccessResponse(c, 201, newBuild)
}

// Input handles build inputs.
func (b Build) Input(c *gin.Context) {
	build, err := b.ByID(c)
	if err != nil {
		return
	}

	if build.Status() != "SUBMITTED" {
		sugar.ErrResponse(c, 400, fmt.Sprintf("Build is '%s', not SUBMITTED", build.Status()))
		return
	}

	_, err = b.Storage.Upload(build.InputUrl(), c.Request.Body, c.Request.ContentLength)
	if err != nil {
		sugar.ErrResponse(c, 500, err)
		return
	}
	callbackURL := fmt.Sprintf("https://%s/builds/%s/events?token=%s", c.Request.Host, build.ID, build.Token)
	reportsURL := fmt.Sprintf("https://%s/builds/%s/reports?token=%s", c.Request.Host, build.ID, build.Token)
	buildID, err := b.Aws.RunBuild(build, callbackURL, reportsURL)
	if err != nil {
		sugar.ErrResponse(c, 500, err)
		return
	}

	err = Transaction(c, func(tx *gorm.DB) error {
		batchJob := BatchService{Aws: b.Aws}.New(buildID)
		return tx.Model(&build).Association("BatchJob").Append(batchJob).Error
	})

	if err != nil {
		return
	}

	sugar.SuccessResponse(c, 200, build)
}

// Logs stream logs for builds.
func (b Build) Logs(c *gin.Context) {
	build, err := b.ByID(c)
	if err != nil {
		return
	}

	StreamBatchLogs(b.Aws, c, &build.BatchJob)
}

func (b Build) canPostEvent(c *gin.Context, build models.Build) bool {
	user, loggedIn := middleware.CheckUser(c)
	if loggedIn && build.Project.UserID == user.ID {
		return true
	}
	token, exists := c.GetQuery("token")
	if exists && build.Token == token {
		return true
	}
	return false
}

// CreateEvent creates build event.
func (b Build) CreateEvent(c *gin.Context) {
	build, err := b.unauthOne(c)
	if err != nil {
		return
	}

	if !b.canPostEvent(c, build) {
		c.AbortWithStatus(403)
		return
	}

	event := models.PostBatchEvent{}
	c.BindJSON(&event)

	if !sugar.ValidateRequest(c, event) {
		return
	}

	currentStatus := build.Status()

	if !models.CanTransition(currentStatus, event.Status) {
		sugar.ErrResponse(c, 400, fmt.Sprintf("%s not valid when current status is %s", event.Status, currentStatus))
		return
	}

	_, isUser := middleware.CheckUser(c)
	if event.Status == models.StatusTerminated && isUser {
		sugar.ErrResponse(c, 400, fmt.Sprintf("Users cannot post TERMINATED events, please upgrade to reco v0.3.1 or above"))
	}

	newEvent, err := BatchService{Aws: b.Aws}.AddEvent(&build.BatchJob, event)

	if event.Status == "CREATING_IMAGE" {
		err = db.Model(&build).Update("FPGAImage", event.Message).Error
	}

	if err != nil {
		c.Error(err)
		sugar.ErrResponse(c, 500, nil)
		return
	}
	eventMessage := "Build entered state:" + event.Status
	sugar.EnqueueEvent(b.Events, c, eventMessage, build.Project.UserID, map[string]interface{}{"build_id": build.ID, "project_name": build.Project.Name, "message": event.Message})
	sugar.SuccessResponse(c, 200, newEvent)

}

// CreateReport creates build report.
func (b Build) CreateReport(c *gin.Context) {
	buildRepo := models.BuildDataSource(db)
	build, err := b.unauthOne(c)
	if err != nil {
		return
	}

	switch c.ContentType() {
	case "application/vnd.reconfigure.io/reports-v1+json":
		report := models.ReportV1{}
		c.BindJSON(&report)
		err = buildRepo.StoreBuildReport(build, report)
	default:
		err = errors.New("Not a valid report version")
	}

	if err != nil {
		c.Error(err)
		sugar.ErrResponse(c, 500, nil)
		return
	}

	sugar.SuccessResponse(c, 200, nil)

}
