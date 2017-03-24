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
	ID         int `gorm:"primary_key"`
	GithubID   string
	Email      string      `gorm:"type:varchar(100);unique_index"`
	AuthTokens []AuthToken //User has many AuthTokens
}

type Project struct {
	ID     int  `gorm:"primary_key"`
	User   User //Project belongs to User
	UserID int
	Name   string
	Builds []Build
}

type AuthToken struct {
	gorm.Model
	Token  string
	UserID int
}

type Build struct {
	ID             int  `gorm:"primary_key"`
	User           User //Build belongs to User, UserID is foreign key
	UserID         int
	Project        Project
	ProjectID      int
	InputArtifact  string
	OutputArtifact string
	OutputStream   string
	Status         string
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
		allBuilds := []Build{}
		db.Find(&allBuilds)
		c.JSON(200, gin.H{
			"builds": allBuilds,
		})
	})

	r.GET("/builds/:id", func(c *gin.Context) {
		BuildID, err := stringToInt(c.Param("id"), c)
		if err != nil {
			return
		}
		builddets := Build{}
		db.Where(&Build{ID: BuildID}).First(&builddets)
		c.JSON(200, gin.H{
			"build": builddets,
		})
	})

	r.GET("/builds/:id/status", func(c *gin.Context) {
		BuildID, err := stringToInt(c.Param("id"), c)
		if err != nil {
			return
		}
		buildstatus := Build{}
		db.Where(&Build{ID: BuildID}).First(&buildstatus)
		c.JSON(200, gin.H{
			"status": buildstatus.Status,
		})
	})

	r.GET("/users", func(c *gin.Context) {
		allUsers := []User{}
		db.Find(&allUsers)
		c.JSON(200, gin.H{
			"users": allUsers,
		})
	})

	r.GET("/users/:id", func(c *gin.Context) {
		UserID, err := stringToInt(c.Param("id"), c)
		if err != nil {
			return
		}
		userdets := User{}
		db.Where(&User{ID: UserID}).First(&userdets)
		c.JSON(200, gin.H{
			"user": userdets,
		})
	})

	r.GET("/users/:id/projects", func(c *gin.Context) {
		UserID, err := stringToInt(c.Param("id"), c)
		if err != nil {
			return
		}
		projects := []Project{}
		db.Where(&Project{UserID: UserID}).Find(&projects)
		c.JSON(200, gin.H{
			"projects": projects,
		})
	})

	r.GET("/projects", func(c *gin.Context) {
		allProjects := []Project{}
		db.Find(&allProjects)
		c.JSON(200, gin.H{
			"projects": allProjects,
		})
	})

	r.GET("/projects/:id", func(c *gin.Context) {
		ProjectID, err := stringToInt(c.Param("id"), c)
		if err != nil {
			return
		}
		ProjectDets := Project{}
		db.Where(&Project{ID: ProjectID}).First(&ProjectDets)
		c.JSON(200, gin.H{
			"Project": ProjectDets,
		})
	})

	r.GET("/projects/:id/builds", func(c *gin.Context) {
		ProjectID, err := stringToInt(c.Param("id"), c)
		if err != nil {
			return
		}
		Builds := Build{}
		db.Where(&Build{ProjectID: ProjectID}).Find(&Builds)
		c.JSON(200, gin.H{
			"Builds": Builds,
		})
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
