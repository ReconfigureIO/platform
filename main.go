package main

import (
	"fmt"
	"strconv"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

type User struct {
	gorm.Model
	GithubID   string
	Email      string      `gorm:"type:varchar(100);unique_index"`
	AuthTokens []AuthToken //User has many AuthTokens
}

type Project struct {
	ID 	   int `gorm:"primary_key"`
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
	gorm.Model
	User           User //Build belongs to User, UserID is foreign key
	UserID         int
	InputArtifact  string
	OutputArtifact string
	OutputStream   string
}

func main() {

	db, err := gorm.Open("postgres", "host=db user=postgres dbname=postgres sslmode=disable password=mysecretpassword")
	if err != nil {
		fmt.Println(err)
		panic("failed to connect database")
	}
	defer db.Close()

	// Migrate the schema
	db.AutoMigrate(&User{})
	db.AutoMigrate(&Project{})
	db.AutoMigrate(&AuthToken{})
	db.AutoMigrate(&Build{})

	//now for some test data
	db.Create(&User{GithubID: "campgareth"})
	db.Create(&Build{UserID: 1, InputArtifact: "golang code", OutputArtifact: ".bin file", OutputStream: "working working done"})
	db.Create(&Project{UserID: 1, Name: "parallel-histogram"})

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

	r.GET("/users", func(c *gin.Context) {
		allUsers := []User{}
		db.Find(&allUsers)
		c.JSON(200, gin.H{
			"users": allUsers,
		})
	})

	r.GET("/users/:githubid", func(c *gin.Context) {
		GithubID := c.Param("githubid")
		userdets := User{}
		db.Where(&User{GithubID: GithubID}).First(&userdets)
		c.JSON(200, gin.H{
			"user": userdets,
		})
	})

	// r.GET("/users/:githubid/projects", func(c *gin.Context) {
	// 	GithubID := c.Param("githubid")
	// 	db.Table("users").Select("users.githubid, projects.userid").Joins("left join projects on projects.userid = users.id").Scan(&results)

	// 	projects := []Project{}
	// 	db.Where(&User{GithubID: GithubID}).First(&userdetails)
	// 	c.JSON(200, gin.H{
	// 		"user": userdetails,
	// 	})
	// })

	r.GET("/projects", func(c *gin.Context) {
		allProjects := []Project{}
		db.Find(&allProjects)
		c.JSON(200, gin.H{
			"projects": allProjects,
		})
	})

	r.GET("/projects/:id", func(c *gin.Context) {
		ProjectID, _ := strconv.Atoi(c.Param("id"))
		ProjectDets := Project{}
		db.Where(&Project{ID: ProjectID}).First(&ProjectDets)
		c.JSON(200, gin.H{
			"Project": ProjectDets,
		})
	})

	// r.GET("/projects/:id", func(c *gin.Context) {
	// 	id := c.Param("id")
	// })

	// r.GET("/projects/:id/builds", func(c *gin.Context) {
	// 	id := c.Param("id")
	// })

	// r.GET("/users/:id", func(c *gin.Context) {
	// 	id := c.Param("id")
	// })

	// r.GET("/builds/:id", func(c *gin.Context) {
	// 	id := c.Param("id")
	// })

	// r.GET("/builds/:id/status", func(c *gin.Context) {
	// 	id := c.Param("id")
	// })

	// Listen and Server in 0.0.0.0:8080
	r.Run(":8080")
}
