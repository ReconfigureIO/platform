package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"os"
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

	// Listen and Server in 0.0.0.0:$PORT
	r.Run(":" + port)
}
