package main

import (
	"fmt"
	"os"

	"github.com/ReconfigureIO/platform/api"
	"github.com/ReconfigureIO/platform/migration"
	"github.com/ReconfigureIO/platform/routes"
	"github.com/gin-gonic/contrib/sessions"
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

	// check migration
	if os.Getenv("RECO_PLATFORM_MIGRATE") == "1" {
		fmt.Println("performing migration...")
		migration.MigrateSchema()
	}
}

func main() {
	port, found := os.LookupEnv("PORT")
	if !found {
		port = "8080"
	}

	r := gin.Default()

	secretKey := os.Getenv("SECRET_KEY_BASE")

	store := sessions.NewCookieStore([]byte(secretKey))
	r.Use(sessions.Sessions("paus", store))
	r.LoadHTMLGlob("templates/*")

	// ping test
	r.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong pong")
	})

	// protected ping test
	r.GET("/secretping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "successful authentication"})
	})

	// setup components
	setupDB()
	routes.SetupRoutes(r)

	// Listen and Server in 0.0.0.0:$PORT
	r.Run(":" + port)
}
