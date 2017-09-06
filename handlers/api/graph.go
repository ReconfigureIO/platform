package api

import (
	"errors"
	"fmt"

	"github.com/ReconfigureIO/platform/middleware"
	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/sugar"
	"github.com/dchest/uniuri"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

// Graph handles requests for graphs.
type Graph struct{}

// Common preload functionality.
func (b Graph) Preload(db *gorm.DB) *gorm.DB {
	return db.Preload("Project").
		Preload("BatchJob").
		Preload("BatchJob.Events", func(db *gorm.DB) *gorm.DB {
			return db.Order("timestamp ASC")
		})
}

// Query fetches graphs for user and project.
func (b Graph) Query(c *gin.Context) *gorm.DB {
	user := middleware.GetUser(c)
	joined := db.Joins("join projects on projects.id = graphs.project_id").
		Where("projects.user_id=?", user.ID)
	return b.Preload(joined)
}

// ByID gets the first graph by ID, 404 if it doesn't exist.
func (b Graph) ByID(c *gin.Context) (models.Graph, error) {
	graph := models.Graph{}
	var id string
	if !bindID(c, &id) {
		return graph, errNotFound
	}
	err := b.Query(c).First(&graph, "graphs.id = ?", id).Error

	if err != nil {
		sugar.NotFoundOrError(c, err)
		return graph, err
	}
	return graph, nil
}

func (b Graph) unauthOne(c *gin.Context) (models.Graph, error) {
	graph := models.Graph{}
	var id string
	if !bindID(c, &id) {
		return graph, errNotFound
	}
	q := b.Preload(db)
	err := q.First(&graph, "id = ?", id).Error
	return graph, err
}

// List lists all graphs.
func (b Graph) List(c *gin.Context) {
	project := c.DefaultQuery("project", "")
	graphs := []models.Graph{}
	q := b.Query(c)

	if project != "" {
		q = q.Where(&models.Graph{ProjectID: project})
	}

	err := q.Find(&graphs).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		sugar.InternalError(c, err)
		return
	}

	sugar.SuccessResponse(c, 200, graphs)
}

// Get fetches a graph.
func (b Graph) Get(c *gin.Context) {
	graph, err := b.ByID(c)
	if err != nil {
		return
	}

	sugar.SuccessResponse(c, 200, graph)
}

// Create creates a graph.
func (b Graph) Create(c *gin.Context) {
	post := models.PostGraph{}
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

	newGraph := models.Graph{Project: project, Token: uniuri.NewLen(64)}
	if err := db.Create(&newGraph).Error; err != nil {
		sugar.InternalError(c, err)
		return
	}
	sugar.SuccessResponse(c, 201, newGraph)
}

// Input handles graph inputs.
func (b Graph) Input(c *gin.Context) {
	graph, err := b.ByID(c)
	if err != nil {
		return
	}

	if graph.Status() != "SUBMITTED" {
		sugar.ErrResponse(c, 400, fmt.Sprintf("Graph is '%s', not SUBMITTED", graph.Status()))
		return
	}

	_, err = awsSession.Upload(graph.InputUrl(), c.Request.Body, c.Request.ContentLength)
	if err != nil {
		sugar.ErrResponse(c, 500, err)
		return
	}
	callbackURL := fmt.Sprintf("https://%s/graphs/%s/events?token=%s", c.Request.Host, graph.ID, graph.Token)
	graphID, err := awsSession.RunGraph(graph, callbackURL)
	if err != nil {
		sugar.ErrResponse(c, 500, err)
		return
	}

	err = Transaction(c, func(tx *gorm.DB) error {
		batchJob := BatchService{}.New(graphID)
		return tx.Model(&graph).Association("BatchJob").Append(batchJob).Error
	})

	if err != nil {
		return
	}

	sugar.SuccessResponse(c, 200, graph)
}

// Download handles graph inputs.
func (b Graph) Download(c *gin.Context) {
	// TODO
}

func (b Graph) canPostEvent(c *gin.Context, graph models.Graph) bool {
	user, loggedIn := middleware.CheckUser(c)
	if loggedIn && graph.Project.UserID == user.ID {
		return true
	}
	token, exists := c.GetQuery("token")
	if exists && graph.Token == token {
		return true
	}
	return false
}

// CreateEvent creates graph event.
func (b Graph) CreateEvent(c *gin.Context) {
	graph, err := b.unauthOne(c)
	if err != nil {
		return
	}

	if !b.canPostEvent(c, graph) {
		c.AbortWithStatus(403)
		return
	}

	event := models.PostBatchEvent{}
	c.BindJSON(&event)

	if !sugar.ValidateRequest(c, event) {
		return
	}

	currentStatus := graph.Status()

	if !models.CanTransition(currentStatus, event.Status) {
		sugar.ErrResponse(c, 400, fmt.Sprintf("%s not valid when current status is %s", event.Status, currentStatus))
		return
	}

	newEvent, err := BatchService{}.AddEvent(&graph.BatchJob, event)

	if err != nil {
		c.Error(err)
		sugar.ErrResponse(c, 500, nil)
		return
	}

	sugar.SuccessResponse(c, 200, newEvent)

}
