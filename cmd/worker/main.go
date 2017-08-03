package main

import (
	"context"
	"log"
	"os"

	"github.com/ReconfigureIO/platform/handlers/api"
	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/afi_watcher"
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
		apideployment := api.Deployment{}
		d := models.DeploymentDataSource(db)

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
				terminating++
			}
		}

		log.Printf("terminated %d deployments", terminating)
		c.Status(200)
	})

	r.POST("/generated-afis", func(c *gin.Context) {
		err := afi_watcher.FindAFI(models.BuildDataSource(db), awsService, api.BatchService{})
		if err != nil {
			log.Println(err.Error())
			c.JSON(500, err)
		} else {
			c.Status(200)
		}
	})

	r.POST("/check-hours", func(c *gin.Context) {
		if err := CheckUserHours(db); err == nil {
			c.String(200, "done")
		} else {
			c.String(500, err.Error())
		}
	})

	// Listen and Server in 0.0.0.0:$PORT
	r.Run(":" + port)
}

// CheckUserHours check running deployments and deduct a minute (cron interval) from
// instance hours of the user.
func CheckUserHours(db *gorm.DB) error {
	api.DB(db)

	ds := models.SubscriptionDataSource(db)
	users, err := ds.ActiveUsers()
	if err != nil {
		return err
	}

	for _, user := range users {
		h, err := api.FetchBillingHours(user.ID).Net()
		if err == nil && h <= 0 {
			// TODO terminate all deployments for user
		}
	}
	return nil
}
