package main

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"gopkg.in/validator.v2"
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
	UserID    int `json:"user_id" validate:"min=1"`
	ProjectID int `json:"project_id" validate:"min=1"`
}

// nur := NewUserRequest{Username: "something", Age: 20}
// if errs := validator.Validate(nur); errs != nil {
// 	 values not valid, deal with errors here
// }

func main() {

	db, err := gorm.Open("postgres", "host=db user=postgres dbname=postgres sslmode=disable password=mysecretpassword")
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

	r.POST("/builds", func(c *gin.Context) {
		post := PostBuild{}
		c.BindJSON(&post)

		if errs := validator.Validate(&post); errs != nil {
			c.AbortWithStatus(404)
			return
		} else {
			newBuild := Build{UserID: post.UserID, ProjectID: post.ProjectID}
			db.Create(&newBuild)
			c.JSON(201, newBuild)
		}
	})

	r.PUT("/builds/:id", func(c *gin.Context) {
		outputbuild := Build{}
		if c.Param("id") != "" {
			BuildID, err := stringToInt(c.Param("id"), c)
			if err != nil {
				return
			}
			db.Where(&Build{ID: BuildID}).First(&outputbuild)
			outputbuild.OutputArtifact = c.PostForm("output_artifact")
			outputbuild.Status = c.PostForm("status")
			db.Save(&outputbuild)
		}

		c.JSON(201, outputbuild)
	})

	r.GET("/builds", func(c *gin.Context) {
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

	r.GET("/builds/:id", func(c *gin.Context) {
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

	r.POST("/projects", func(c *gin.Context) {
		id := c.PostForm("user_id")
		name := c.PostForm("name")
		userID, err := stringToInt(id, c)
		if err != nil {
			return
		}
		newProject := Project{UserID: userID, Name: name}
		db.Create(&newProject)
		c.JSON(201, newProject)
	})

	r.PUT("/projects/:id", func(c *gin.Context) {
		userid := c.PostForm("user_id")
		userID, err := stringToInt(userid, c)
		outputproj := Project{}
		if err != nil {
			return
		}
		if c.Param("id") != "" {
			ProjID, err := stringToInt(c.Param("id"), c)
			if err != nil {
				return
			}
			db.Where(&Project{ID: ProjID}).First(&outputproj)
			outputproj.Name = c.PostForm("name")
			outputproj.UserID = userID
			db.Save(&outputproj)

		}

		c.JSON(201, outputproj)
	})

	r.GET("/projects", func(c *gin.Context) {
		projects := []Project{}
		db.Find(&projects)
		c.JSON(200, gin.H{
			"projects": projects,
		})
	})

	r.GET("/projects/:id", func(c *gin.Context) {
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

	// Listen and Server in 0.0.0.0:8080
	r.Run(":8080")
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
