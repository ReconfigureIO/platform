package main

import (
	"context"
	"log"
	"os"

	"github.com/ReconfigureIO/platform/handlers/api"
	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/aws"
	"github.com/ReconfigureIO/platform/service/mock_deployment"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

var (
	mockDeploy = mock_deployment.New(mock_deployment.ServiceConfig{
		LogGroup: "josh-test-sdaccel",
		Image:    "398048034572.dkr.ecr.us-east-1.amazonaws.com/reconfigureio/platform/deployment:latest",
		AMI:      "ami-850c7293",
	})
	aws = aws.New(aws.ServiceConfig{
		LogGroup: "josh-test-sdaccel",
		Image:    "398048034572.dkr.ecr.us-east-1.amazonaws.com/reconfigureio/platform/deployment:latest",
		AMI:      "ami-850c7293",
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
		apideployment := api.Deployment{}
		d := models.PostgresRepo{db}
		//get list of deployments in terminating state
		terminatingdeployments, err := d.GetWithStatus([]string{models.StatusTerminating, models.StatusCompleted, models.StatusErrored}, 100)
		log.Printf("Looking up %d deployments", len(terminatingdeployments))

		if len(terminatingdeployments) == 0 {
			c.Status(200)
			return
		}
		//get the status of the associated EC2 instances
		statuses, err := mockDeploy.DescribeInstanceStatus(context.Background(), terminatingdeployments)
		if err != nil {
			c.JSON(500, err)
			return
		}
		log.Printf("statuses of %v", statuses)
		terminating := 0
		//for each deployment, if instance is terminated, send event
		for _, deployment := range terminatingdeployments {
			status, found := statuses[deployment.InstanceID]
			if found && status == ec2.InstanceStateNameTerminated {
				event := models.PostDepEvent{
					Status:  models.StatusTerminated,
					Message: models.StatusTerminated,
					Code:    0,
				}
				_, err := apideployment.AddEvent(c, deployment, event)
				if err != nil {
					c.JSON(500, err)
					return
				}
				terminating += 1
			}
		}

		log.Printf("terminated %d deployments", terminating)
		c.Status(200)
	})

	r.POST("/generated-afis", func(c *gin.Context) {
		tempbuild := api.Build{}
		d := models.PostgresRepo{db}
		//get list of builds waiting for AFI generation to finish
		buildswaitingonafis, err := d.GetBuildsWithStatus([]string{models.StatusCreatingImage}, 100)
		log.Printf("Looking up %d builds", len(buildswaitingonafis))

		if len(buildswaitingonafis) == 0 {
			c.Status(200)
			return
		}
		//get the status of the associated AFIs
		statuses, err := aws.DescribeAFIStatus(context.Background(), buildswaitingonafis)
		if err != nil {
			c.JSON(500, err)
			return
		}
		log.Printf("statuses of %v", statuses)
		waiting := 0
		//for each build check associated AFI, if done, post event
		for _, build := range buildswaitingonafis {
			status, found := statuses[build.FPGAImage.AFIID]
			if found && status == "available" {
				event := models.PostBatchEvent{
					Status:  models.StatusCompleted,
					Message: models.StatusCompleted,
					Code:    0,
				}
				_, err := tempbuild.AddEvent(c, build, event)
				if err != nil {
					c.JSON(500, err)
					return
				}
				waiting += 1
			}
		}

		log.Printf("%d builds have finished generating AFIs", waiting)
		c.Status(200)
	})

	// Listen and Server in 0.0.0.0:$PORT
	r.Run(":" + port)
}
