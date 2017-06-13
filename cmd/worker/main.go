package main

import (
	"log"
	"os"

	"github.com/ReconfigureIO/platform/models"
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
)

func main() {
	gormConnDets := os.Getenv("DATABASE_URL")
	db, err := gorm.Open("postgres", gormConnDets)

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
		d := PostgresRepo{db}
		terminatingdeps, err = d.GetWithStatus([]string{"TERMINATING"}, 10)

		statuses, err := mockDeploy.DescribeInstanceStatus(context.Background(), terminatingdeps)
		if err != nil {
			c.JSON(500, err)
			return
		}

		for _, status := range statuses {
			for dep := range terminatingdeps {
				if terminatingdeps[dep].InstanceID == status.ID {
					if status.Status == "TERMINATED" {
						event := models.PostDepEvent{
							Status:  "TERMINATED",
							Message: "TERMINATED",
							Code:    0,
						}
						_, err := addEvent(&dep.DepJob, event)
						if err != nil {
							c.JSON(500, err)
							return
						}
					}
				}
			}
		}
		c.JSON(200, "events posted")
	})

	// Listen and Server in 0.0.0.0:$PORT
	r.Run(":" + port)
}
