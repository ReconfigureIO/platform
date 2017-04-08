package api

import (
	"github.com/ReconfigureIO/platform/models"
	"github.com/gin-gonic/gin"
	validator "gopkg.in/validator.v2"
)

type Project struct{}

func (p Project) Create(c *gin.Context) {
	post := models.PostProject{}
	c.BindJSON(&post)
	if err := validateProject(post, c); err != nil {
		return
	}
	newProject := models.Project{UserID: post.UserID, Name: post.Name}
	db.Create(&newProject)
	c.JSON(201, newProject)
}

func (p Project) Update(c *gin.Context) {
	post := models.PostProject{}
	c.BindJSON(&post)
	if c.Param("id") != "" {
		ProjID, err := stringToInt(c.Param("id"), c)
		if err != nil {
			return
		}
		if err := validateProject(post, c); err != nil {
			return
		}
		outputproj := models.Project{}
		db.Where(&models.Project{ID: ProjID}).First(&outputproj)
		db.Model(&outputproj).Updates(models.Project{UserID: post.UserID, Name: post.Name})
		c.JSON(201, outputproj)
	}
}

func (p Project) List(c *gin.Context) {
	projects := []models.Project{}
	db.Find(&projects)
	c.JSON(200, gin.H{
		"projects": projects,
	})
}

func (p Project) Get(c *gin.Context) {
	outputproj := []models.Project{}
	if c.Param("id") != "" {
		ProjID, err := stringToInt(c.Param("id"), c)
		if err != nil {
			return
		}
		db.Where(&models.Project{ID: ProjID}).First(&outputproj)
	}
	c.JSON(200, outputproj)
}

func validateProject(postp models.PostProject, c *gin.Context) error {
	if err := validator.Validate(&postp); err != nil {
		c.AbortWithStatus(404)
		return err
	}
	return nil
}
