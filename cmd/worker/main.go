package main

import (
	"context"
	"log"
	"os"

	"github.com/ReconfigureIO/platform/handlers/api"
	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/afi_watcher"
	"github.com/ReconfigureIO/platform/service/aws"
	"github.com/ReconfigureIO/platform/service/billing_hours"
	"github.com/ReconfigureIO/platform/service/deployment"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

var (
	deploy = deployment.New(deployment.ServiceConfig{
		LogGroup: "josh-test-sdaccel",
		Image:    "398048034572.dkr.ecr.us-east-1.amazonaws.com/reconfigureio/platform/deployment:latest",
		AMI:      "ami-850c7293",
	})
	awsService = aws.New(aws.ServiceConfig{
		LogGroup:      "/aws/batch/job",
		Bucket:        "reconfigureio-builds",
		Queue:         "build-jobs",
		JobDefinition: "sdaccel-builder-build",
	})
)

func main() {
	gormConnDets := os.Getenv("DATABASE_URL")
	db, err := gorm.Open("postgres", gormConnDets)
	db.LogMode(true)
	api.DB(db)

	if err != nil {
		log.Fatalf("failed to connect to database: %s", err.Error())
	}

	port, found := os.LookupEnv("PORT")
	if !found {
		port = "8080"
	}

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		err := db.DB().Ping()
		if err != nil {
			c.String(500, "error connecting to db")
		} else {
			c.String(200, "OK")
		}
	})

	r.POST("/hello", func(c *gin.Context) {
		log.Printf("Hello world\n")
		c.String(200, "hello")
	})

	r.POST("/terminate-deployments", func(c *gin.Context) {
		d := models.DeploymentDataSource(db)
		ctx := context.Background()

		err := deployment.NewInstances(d, deploy).UpdateInstanceStatus(ctx)

		if err != nil {
			log.Println(err.Error())
			c.JSON(500, err)
		} else {
			c.Status(200)
		}

	})

	r.POST("/generated-afis", func(c *gin.Context) {
		watcher := afi_watcher.NewAFIWatcher(models.BuildDataSource(db), awsService, models.BatchDataSource(db))

		err := watcher.FindAFI(c, 100)

		if err != nil {
			log.Println(err.Error())
			c.JSON(500, err)
		} else {
			c.Status(200)
		}
	})

	r.POST("/check-hours", func(c *gin.Context) {
		if err := billing_hours.CheckUserHours(models.SubscriptionDataSource(db), models.DeploymentDataSource(db), deploy); err == nil {
			c.String(200, "done")
		} else {
			c.String(500, err.Error())
		}
	})

	r.POST("/update-debit", func(c *gin.Context) {
		if err := billing_hours.UpdateDebits(models.UserBalanceDataSource(db), models.DeploymentDataSource(db), deploy); err == nil {
			c.String(200, "done")
		} else {
			c.String(500, err.Error())
		}
	})

	// Listen and Server in 0.0.0.0:$PORT
	r.Run(":" + port)
}
