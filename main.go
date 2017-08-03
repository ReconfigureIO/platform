package main

import (
	"fmt"
	"log"

	"github.com/ReconfigureIO/platform/config"
	"github.com/ReconfigureIO/platform/handlers/api"
	"github.com/ReconfigureIO/platform/migration"
	"github.com/ReconfigureIO/platform/routes"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	stripe "github.com/stripe/stripe-go"
)

func setupDB(conf config.Config) *gorm.DB {
	db, err := gorm.Open("postgres", conf.DbUrl)

	if conf.Reco.Env != "release" {
		db.LogMode(true)
	}

	if err != nil {
		fmt.Println(err)
		panic("failed to connect database")
	}

	api.DB(db)

	// check migration
	if conf.Reco.PlatformMigrate {
		fmt.Println("performing migration...")
		migration.MigrateSchema()
	}
	return db
}

func main() {
	conf, err := config.ParseEnvConfig()
	if err != nil {
		log.Fatal(err)
	}

	stripe.Key = conf.StripeKey

	r := gin.Default()

	// setup components
	db := setupDB(*conf)

	api.Configure(*conf)

	// ping test
	r.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong pong")
	})

	// cors
	corsConfig := cors.DefaultConfig()
	// allow cookies from other domains
	corsConfig.AllowCredentials = true

	switch conf.Reco.Env {
	case "production":
		corsConfig.AllowOrigins = []string{
			"http://app.reconfigure.io",
			"https://app.reconfigure.io",
			"http://local.reconfigure.io",
			"http://local.reconfigure.io:4200",
		}
	default:
		corsConfig.AllowOrigins = []string{
			"http://app-staging.reconfigure.io",
			"https://app-staging.reconfigure.io",
			"http://local.reconfigure.io",
			"http://local.reconfigure.io:4200",
		}
	}

	r.Use(cors.New(corsConfig))
	r.LoadHTMLGlob("templates/*")

	// routes
	routes.SetupRoutes(conf.SecretKey, r, db)

	// Listen and Server in 0.0.0.0:$PORT
	r.Run(":" + conf.Port)
}
