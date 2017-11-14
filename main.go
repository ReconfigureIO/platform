package main

import (
	"time"

	"github.com/ReconfigureIO/platform/config"
	"github.com/ReconfigureIO/platform/deployments_migration"
	"github.com/ReconfigureIO/platform/handlers/api"
	//"github.com/ReconfigureIO/platform/migration"
	"github.com/ReconfigureIO/platform/routes"
	"github.com/ReconfigureIO/platform/service/deployment"
	"github.com/ReconfigureIO/platform/service/events"
	"github.com/ReconfigureIO/platform/service/leads"
	"github.com/ReconfigureIO/platform/service/queue"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/contrib/ginrus"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
)

var (
	version string
)

func startDeploymentQueue(conf config.Config, db *gorm.DB) queue.Queue {
	runner := queue.DeploymentRunner{
		Hostname: conf.Host,
		DB:       db,
		Service:  deployment.New(conf.Reco.Deploy),
	}
	deploymentQueue := queue.NewWithDBStore(
		db,
		runner,
		2, // TODO make this non static.
		"deployment",
	)
	go deploymentQueue.Start()
	return deploymentQueue
}

func main() {
	log.Info("Parsing Config")
	conf, err := config.ParseEnvConfig()
	if err != nil {
		log.Fatal(err)
	}

	log.Info("Setting up Logging")
	err = config.SetupLogging(version, conf)
	if err != nil {
		log.Fatal(err)
	}

	log.Info("Setting up Intercom")
	events := events.NewIntercomEventService(conf.Reco.Intercom, 100)

	if conf.Reco.FeatureIntercom {
		go events.DrainEvents()
	}

	log.Info("Setting up Routes")
	r := gin.New()
	r.Use(ginrus.Ginrus(log.StandardLogger(), time.RFC3339, true))
	r.Use(gin.Recovery())

	log.Info("Setting up DB")
	// setup components
	db := config.SetupDB(conf)
	api.DB(db)

	// check migration
	if conf.Reco.PlatformMigrate {
		log.Info("performing migration...")
		//migration.MigrateSchema()
		deployments_migration.MigrateSchema()
	}

	leads := leads.New(conf.Reco.Intercom, db)

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
	routes.SetupRoutes(conf.Reco, conf.SecretKey, r, db, events, leads)

	// queue
	var deploymentQueue queue.Queue
	if conf.Reco.FeatureDepQueue {
		log.Info("deployment queue enabled. starting...")
		deploymentQueue = startDeploymentQueue(*conf, db)
		api.DepQueue(deploymentQueue)
		log.Info("deployment queue started.")
	}

	// Listen and Server in 0.0.0.0:$PORT
	err = r.Run(":" + conf.Port)

	// Code would normally not reach here.
	if err != nil {
		if deploymentQueue != nil {
			deploymentQueue.Halt()
		}
	}
}
