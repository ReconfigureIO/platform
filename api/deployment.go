package api

import (
	"strconv"

	"github.com/ReconfigureIO/platform/models"
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
	successResponse(c, 200, "This function does nothing yet")
}
