package main

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"gopkg.in/validator.v2"
	"os"
	"strconv"
)

var NOT_FOUND = errors.New("Not Found")

type User struct {
	ID         int         `gorm:"primary_key" json:"id"`
	GithubID   string      `json:"github_id"`
	Email      string      `gorm:"type:varchar(100);unique_index" json:"email"`
	AuthTokens []AuthToken `json:"auth_token"` //User has many AuthTokens
}

type Project struct {
	ID     int     `gorm:"primary_key" json:"id"`
	User   User    `json:"user"` //Project belongs to User
	UserID int     `json:"user_id"`
	Name   string  `json:"name"`
	Builds []Build `json:"builds"`
}

type PostProject struct {
	UserID int    `json:"user_id"`
	Name   string `json:"name"`
}

type AuthToken struct {
	gorm.Model
	Token  string `json:"token"`
	UserID int    `json:"user_id"`
}

type Build struct {
	ID             int     `gorm:"primary_key" json:"id"`
	User           User    `json:"user"` //Build belongs to User, UserID is foreign key
	UserID         int     `json:"user_id"`
	Project        Project `json:"project"`
	ProjectID      int     `json:"project_id"`
	InputArtifact  string  `json:"input_artifact"`
	OutputArtifact string  `json:"output_artifact"`
	OutputStream   string  `json:"output_stream"`
	Status         string  `gorm:"default:'SUBMITTED'" json:"status"`
}

type PostBuild struct {
	UserID         int    `json:"user_id" validate:"min=1"`
	ProjectID      int    `json:"project_id" validate:"min=1"`
	InputArtifact  string `json:"input_artifact"`
	OutputArtifact string `json:"output_artifact"`
	OutputStream   string `json:"output_stream"`
	Status         string `gorm:"default:'SUBMITTED'" json:"status"`
}

func main() {

	gormConnDets := os.Getenv("DATABASE_URL")
	port, found := os.LookupEnv("PORT")
	if !found {
		port = "8080"
	}

	db, err := gorm.Open("postgres", gormConnDets)
	if err != nil {
		fmt.Println(err)
		panic("failed to connect database")
	}
	defer db.Close()

	r := gin.Default()

	// Ping test
	r.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong pong")
	})

	authorized := r.Group("/", gin.BasicAuth(gin.Accounts{
		"reco-test": "ffea108b2166081bcfd03a99c597be78b3cf30de685973d44d3b86480d644264",
	}))

	authorized.GET("/secretping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "successful authentication"})
	})

	authorized.POST("/builds", func(c *gin.Context) {
		post := PostBuild{}
		c.BindJSON(&post)

		if err := validateBuild(post, c); err != nil {
			return
		}
		newBuild := Build{UserID: post.UserID, ProjectID: post.ProjectID}
		db.Create(&newBuild)
		c.JSON(201, newBuild)
	})

	authorized.PUT("/builds/:id", func(c *gin.Context) {
		post := PostBuild{}
		c.BindJSON(&post)
		if c.Param("id") != "" {
			BuildID, err := stringToInt(c.Param("id"), c)
			if err != nil {
				return
			}
			if err := validateBuild(post, c); err != nil {
				return
			}
			outputbuild := Build{}
			db.Where(&Build{ID: BuildID}).First(&outputbuild)
			db.Model(&outputbuild).Updates(Build{UserID: post.UserID, ProjectID: post.ProjectID, InputArtifact: post.InputArtifact, OutputArtifact: post.OutputArtifact, OutputStream: post.OutputStream, Status: post.Status})
			c.JSON(201, outputbuild)
		}
	})

	authorized.GET("/builds", func(c *gin.Context) {
		project := c.DefaultQuery("project", "")
		Builds := []Build{}
		if project != "" {
			ProjID, err := stringToInt(project, c)
			if err != nil {
				return
			}
			db.Where(&Build{ProjectID: ProjID}).Find(&Builds)
		} else {
			db.Find(&Builds)
		}

		c.JSON(200, gin.H{
			"builds": Builds,
		})
	})

	authorized.GET("/builds/:id", func(c *gin.Context) {
		outputbuild := []Build{}
		if c.Param("id") != "" {
			BuildID, err := stringToInt(c.Param("id"), c)
			if err != nil {
				return
			}
			db.Where(&Build{ID: BuildID}).First(&outputbuild)
		}
		c.JSON(200, outputbuild)
	})

	authorized.POST("/projects", func(c *gin.Context) {
		post := PostProject{}
		c.BindJSON(&post)
		if err := validateProject(post, c); err != nil {
			return
		}
		newProject := Project{UserID: post.UserID, Name: post.Name}
		db.Create(&newProject)
		c.JSON(201, newProject)
	})

	authorized.PUT("/projects/:id", func(c *gin.Context) {
		post := PostProject{}
		c.BindJSON(&post)
		if c.Param("id") != "" {
			ProjID, err := stringToInt(c.Param("id"), c)
			if err != nil {
				return
			}
			if err := validateProject(post, c); err != nil {
				return
			}
			outputproj := Project{}
			db.Where(&Project{ID: ProjID}).First(&outputproj)
			db.Model(&outputproj).Updates(Project{UserID: post.UserID, Name: post.Name})
			c.JSON(201, outputproj)
		}
	})

	authorized.GET("/projects", func(c *gin.Context) {
		projects := []Project{}
		db.Find(&projects)
		c.JSON(200, gin.H{
			"projects": projects,
		})
	})

	authorized.GET("/projects/:id", func(c *gin.Context) {
		outputproj := []Project{}
		if c.Param("id") != "" {
			ProjID, err := stringToInt(c.Param("id"), c)
			if err != nil {
				return
			}
			db.Where(&Project{ID: ProjID}).First(&outputproj)
		}
		c.JSON(200, outputproj)
	})

	// Listen and Server in 0.0.0.0:$PORT
	r.Run(":" + port)
}

func stringToInt(s string, c *gin.Context) (int, error) {
	i, err := strconv.Atoi(s)
	if err != nil {
		c.AbortWithStatus(404)
		return 0, NOT_FOUND
	} else {
		return i, nil
	}
}

func validateBuild(postb PostBuild, c *gin.Context) error {
	if err := validator.Validate(&postb); err != nil {
		c.AbortWithStatus(404)
		return err
	} else {
		return nil
	}
}

func validateProject(postp PostProject, c *gin.Context) error {
	if err := validator.Validate(&postp); err != nil {
		c.AbortWithStatus(404)
		return err
	} else {
		return nil
	}
}
