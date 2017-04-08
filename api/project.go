package api

import (
	"github.com/ReconfigureIO/platform/models"
	"github.com/gin-gonic/gin"
)

type Project struct{}

func (p Project) Create(c *gin.Context) {
	post := models.PostProject{}
	c.BindJSON(&post)
	if !validateRequest(c, post) {
		return
	}
	newProject := models.Project{UserID: post.UserID, Name: post.Name}
	db.Create(&newProject)
	successResponse(c, 201, newProject)
}

func (p Project) Update(c *gin.Context) {
	post := models.PostProject{}
	c.BindJSON(&post)
	var id int
	if !bindId(c, &id) {
		return
	}
	if !validateRequest(c, post) {
		return
	}
	outputproj := models.Project{}
	db.Where(&models.Project{ID: id}).First(&outputproj)
	db.Model(&outputproj).Updates(models.Project{UserID: post.UserID, Name: post.Name})
	successResponse(c, 200, outputproj)
}

func (p Project) List(c *gin.Context) {
	projects := []models.Project{}
	db.Find(&projects)
	successResponse(c, 200, projects)
}

func (p Project) Get(c *gin.Context) {
	outputproj := []models.Project{}
	var id int
	if !bindId(c, &id) {
		return
	}
	db.Where(&models.Project{ID: id}).First(&outputproj)
	c.JSON(200, outputproj)
}
