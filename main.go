package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"github.com/jinzhu/gorm"
    _ "github.com/jinzhu/gorm/dialects/sqlite"
)

type Account struct {
  gorm.Model
  GithubID string
  Email string
}

type Project struct {
  gorm.Model
  Name string
}

type AuthToken struct {
  gorm.Model
  Token string
}

type Build struct {
  gorm.Model
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
