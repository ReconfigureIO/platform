package main

import (
	"fmt"
	"os"

	"github.com/ReconfigureIO/platform/api"
	"github.com/ReconfigureIO/platform/routes"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

func setupDB() {
	gormConnDets := os.Getenv("DATABASE_URL")
	db, err := gorm.Open("postgres", gormConnDets)
	if err != nil {
		fmt.Println(err)
		panic("failed to connect database")
	}
	api.DB(db)
}

func main() {
	port, found := os.LookupEnv("PORT")
	if !found {
		port = "8080"
	}

	r := gin.Default()

	// ping test
	r.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong pong")
	})

	authMiddleware := gin.BasicAuth(gin.Accounts{
		"reco-test": "ffea108b2166081bcfd03a99c597be78b3cf30de685973d44d3b86480d644264",
	})

	protectedRoute := r.Group("/", authMiddleware)

	// protected ping test
	protectedRoute.GET("/secretping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "successful authentication"})
	})

	// setup components
	setupDB()
	routes.SetupRoutes(protectedRoute)

	// Listen and Server in 0.0.0.0:$PORT
	r.Run(":" + port)
}
