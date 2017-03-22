package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

type User struct {
	gorm.Model
	GithubID   string
	Emails     []Email //User has many Emails, Email.UserID is key
	Team       Team    // User belongs to Team, TeamID is foreign key
	TeamID     int
	AuthTokens []AuthToken //User has many AuthTokens
}

type Email struct {
	gorm.Model
	User   User //Email belongs to User, UserID is foreign key
	UserID int
	Email  string `gorm:"type:varchar(100);unique_index"` // `type` set sql type, `unique_index` will create unique index for this column
}

type Team struct {
	gorm.Model
	Users []User
	Name  string
}

type Project struct {
	gorm.Model
	Team   Team //Project belongs to Team
	TeamID int
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
	db.AutoMigrate(&Email{})
	db.AutoMigrate(&Team{})
	db.AutoMigrate(&Project{})
	db.AutoMigrate(&AuthToken{})
	db.AutoMigrate(&Build{})

	//now for some test data
	db.Create(&User{GithubID: "campgareth", TeamID: 1})
	db.Create(&Email{UserID: 1, Email: "max.siegieda@reconfigure.io"})
	db.Create(&Team{Name: "reconfigure.io"})
	db.Create(&Build{UserID: 1, InputArtifact: "golang code", OutputArtifact: ".bin file", OutputStream: "working working done"})

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
		userdetails := User{}
		fmt.Println(GithubID)
		db.Where(&User{GithubID: GithubID}).First(&userdetails)
		c.JSON(200, gin.H{
			"user": userdetails,
		})	
	})

	// r.GET("/users/:id/projects", func(c *gin.Context) {
	// 	id := c.Param("id")	
	// })	

	// r.GET("/users/:id", func(c *gin.Context) {
	// 	id := c.Param("id")	
	// })

	// r.GET("/projects", func(c *gin.Context) {
	// 	//is user logged in?	
	// })

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
