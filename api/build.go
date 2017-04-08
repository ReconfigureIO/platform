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

type Build struct{}

func (b Build) List(c *gin.Context) {
	project := c.DefaultQuery("project", "")
	Builds := []models.Build{}
	if project != "" {
		ProjID, err := stringToInt(project, c)
		if err != nil {
			return
		}
		db.Where(&models.Build{ProjectID: ProjID}).Find(&Builds)
	} else {
		db.Find(&Builds)
	}

	c.JSON(200, M{
		"builds": Builds,
	})
}

func (b Build) Get(c *gin.Context) {
	outputbuild := []models.Build{}
	if c.Param("id") != "" {
		BuildID, err := stringToInt(c.Param("id"), c)
		if err != nil {
			return
		}
		db.Where(&models.Build{ID: BuildID}).First(&outputbuild)
	}
	c.JSON(200, outputbuild)
}

func (b Build) Create(c *gin.Context) {
	post := models.PostBuild{}
	c.BindJSON(&post)

	if err := validateBuild(post, c); err != nil {
		return
	}
	newBuild := models.Build{UserID: post.UserID, ProjectID: post.ProjectID}
	db.Create(&newBuild)
	c.JSON(201, newBuild)
}

func (b Build) Update(c *gin.Context) {
	post := models.PostBuild{}
	c.BindJSON(&post)
	if c.Param("id") != "" {
		BuildID, err := stringToInt(c.Param("id"), c)
		if err != nil {
			return
		}
		if err := validateBuild(post, c); err != nil {
			return
		}
		outputbuild := models.Build{}
		db.Where(&models.Build{ID: BuildID}).First(&outputbuild)
		build := models.Build{
			UserID:         post.UserID,
			ProjectID:      post.ProjectID,
			InputArtifact:  post.InputArtifact,
			OutputArtifact: post.OutputArtifact,
			OutputStream:   post.OutputStream,
			Status:         post.Status,
		}
		db.Model(&outputbuild).Updates(build)
		c.JSON(201, outputbuild)
	}
}

func (b Build) Input(c *gin.Context) {
	id, err := stringToInt(c.Param("id"), c)
	if err != nil {
		return
	}

	build := models.Build{}
	db.First(&build, id)

	if build.Status != "SUBMITTED" {
		c.JSON(400, ApiError{
			Error: fmt.Sprintf("Build is '%s', not SUBMITTED", build.Status),
		})
		return
	}

	key := fmt.Sprintf("builds/%d/simulation.tar.gz", id)

	s3Url, err := awsSession.Upload(key, c.Request.Body, c.Request.ContentLength)

	if err != nil {
		c.AbortWithStatus(500)
		c.Error(err)
		return
	}

	buildId, err := awsSession.RunBuild(s3Url)

	if err != nil {
		c.AbortWithStatus(500)
		c.Error(err)
		return
	}

	db.Model(&build).Updates(models.Build{BatchId: buildId, Status: "QUEUED"})
	c.JSON(200, build)
}

func (b Build) Logs(c *gin.Context) {
	id, err := stringToInt(c.Param("id"), c)
	if err != nil {
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
		c.AbortWithStatus(500)
		c.Error(err)
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

func validateBuild(postb models.PostBuild, c *gin.Context) error {
	if err := validator.Validate(&postb); err != nil {
		c.AbortWithStatus(404)
		return err
	}
	return nil
}
