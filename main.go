package main

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
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
	OutputStream   string  `json:"outout_stream"`
	Status         string  `json:"status"`
}

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

	r.POST("/form_post", func(c *gin.Context) {
		message := c.PostForm("message")
		nick := c.DefaultPostForm("nick", "anonymous")

		c.JSON(200, gin.H{
			"status":  "posted",
			"message": message,
			"nick":    nick,
		})
	})

	r.GET("/projects", func(c *gin.Context) {
		Projects := []Project{}
		db.Find(&Projects)
		c.JSON(200, gin.H{
			"projects": Projects,
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
