package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"github.com/jinzhu/gorm"
    _ "github.com/jinzhu/gorm/dialects/sqlite"
)

type User struct {
	gorm.Model
	GithubID string
	Emails []Email //User has many Emails, Email.UserID is key
	Team Team // User belongs to Team, TeamID is foreign key
	TeamID int
	AuthTokens []AuthToken //User has many AuthTokens
}

type Email struct {
	gorm.Model
	User User //Email belongs to User, UserID is foreign key
    UserID  int 
    Email   string  `gorm:"type:varchar(100);unique_index"` // `type` set sql type, `unique_index` will create unique index for this column
}

type Team struct {
	gorm.Model
	Users []User
}

type Project struct {
  	gorm.Model
  	Team Team //Project belongs to Team
  	TeamID int
  	Name string
  	Builds []Build
}

type AuthToken struct {
	gorm.Model
  	Token string
  	UserID int
}

type Build struct {
  	gorm.Model
  	User User //Build belongs to User, UserID is foreign key
  	UserID int
  	InputArtifact string
  	OutputArtifact string
  	CreatedTime string
  	OutputStream string
}

func main() {

  	db, err := gorm.Open("sqlite3", "test.db")
	if err != nil {
	panic("failed to connect database")
	}
	defer db.Close()

	// Migrate the schema
	db.AutoMigrate(&Account{})
	db.AutoMigrate(&Project{})
	db.AutoMigrate(&AuthToken{})
	db.AutoMigrate(&Build{})

	db.Create(&Account{GithubID: "campgareth", Email: "max.siegieda@reconfigure.io"})


	r := gin.Default()

	// Ping test
	r.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong pong")
	})

	// Listen and Server in 0.0.0.0:8080
	r.Run(":8080")
}
