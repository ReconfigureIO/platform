package main

import (
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	// Ping test
	r.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})

	// Listen and Server in 0.0.0.0:8080
	r.Run(":8080")
}
