package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	port, found := os.LookupEnv("PORT")
	if !found {
		port = "8080"
	}

	r := gin.Default()
	r.POST("/hello", func(c *gin.Context) {
		log.Printf("Hello world\n")
		c.String(200, "hello")
	})

	// Listen and Server in 0.0.0.0:$PORT
	r.Run(":" + port)
}
