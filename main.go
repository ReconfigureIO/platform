package main

import (
	"fmt"
	"os"

	"github.com/ReconfigureIO/platform/api"
	"github.com/ReconfigureIO/platform/auth"
	"github.com/ReconfigureIO/platform/migration"
	"github.com/ReconfigureIO/platform/routes"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	stripe "github.com/stripe/stripe-go"
)

func setupDB() *gorm.DB {
	gormConnDets := os.Getenv("DATABASE_URL")
	db, err := gorm.Open("postgres", gormConnDets)

	if os.Getenv("GIN_MODE") != "release" {
		db.LogMode(true)
	}

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
	return db
}

func main() {
	port, found := os.LookupEnv("PORT")
	if !found {
		port = "8080"
	}

	r := gin.Default()

	secretKey := os.Getenv("SECRET_KEY_BASE")

	// setup components
	db := setupDB()

	store := sessions.NewCookieStore([]byte(secretKey))
	r.Use(sessions.Sessions("paus", store))
	r.Use(auth.SessionAuth(db))

	r.LoadHTMLGlob("templates/*")

	// ping test
	r.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong pong")
	})

	routes.SetupRoutes(r, db)

	stripe.Key = "sk_test_NvEpeLnLAV15b9TWJzZKLkvW"

	// Listen and Server in 0.0.0.0:$PORT
	r.Run(":" + port)
}
