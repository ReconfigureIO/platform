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

	r := gin.Default()

	// Ping test
	r.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong pong")
	})

	// Listen and Server in 0.0.0.0:8080
	r.Run(":8080")
}
