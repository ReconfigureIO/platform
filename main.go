package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/ReconfigureIO/platform/models"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/batch"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"gopkg.in/validator.v2"
	"io"
	"log"
	"os"
	"strconv"
	"time"
)

var NOT_FOUND = errors.New("Not Found")

func main() {

	gormConnDets := os.Getenv("DATABASE_URL")
	port, found := os.LookupEnv("PORT")
	if !found {
		port = "8080"
	}

	db, err := gorm.Open("postgres", gormConnDets)
	if err != nil {
		fmt.Println(err)
		panic("failed to connect database")
	}
	defer db.Close()

	db.AutoMigrate(&models.Simulation{})

	awsSession := session.Must(session.NewSession(aws.NewConfig().WithRegion("us-east-1")))

	r := gin.Default()

	// Ping test
	r.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong pong")
	})

	r.Use(gin.BasicAuth(gin.Accounts{
		"reco-test": "ffea108b2166081bcfd03a99c597be78b3cf30de685973d44d3b86480d644264",
	}))

	r.GET("/secretping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "successful authentication"})
	})

	r.GET("/secretpong", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "successful authentication"})
	})

	r.POST("/builds", func(c *gin.Context) {
		post := models.PostBuild{}
		c.BindJSON(&post)

		if err := validateBuild(post, c); err != nil {
			return
		}
		newBuild := models.Build{UserID: post.UserID, ProjectID: post.ProjectID}
		db.Create(&newBuild)
		c.JSON(201, newBuild)
	})

	r.PUT("/builds/:id", func(c *gin.Context) {
		post := models.PostBuild{}
		c.BindJSON(&post)
		if c.Param("id") != "" {
			BuildID, err := stringToInt(c.Param("id"), c)
			if err != nil {
				return
			}
			if err := validateBuild(post, c); err != nil {
				return
			}
			outputbuild := models.Build{}
			db.Where(&models.Build{ID: BuildID}).First(&outputbuild)
			db.Model(&outputbuild).Updates(models.Build{UserID: post.UserID, ProjectID: post.ProjectID, InputArtifact: post.InputArtifact, OutputArtifact: post.OutputArtifact, OutputStream: post.OutputStream, Status: post.Status})
			c.JSON(201, outputbuild)
		}
	})

	r.PUT("/builds/:id/input", func(c *gin.Context) {
		session := s3.New(awsSession)
		batchSession := batch.New(awsSession)

		if c.Param("id") != "" {
			_, err := stringToInt(c.Param("id"), c)
			if err != nil {
				return
			}

			// This is bad and buffers the entire body in memory :(
			body := bytes.Buffer{}
			body.ReadFrom(c.Request.Body)

			putParams := &s3.PutObjectInput{
				Bucket:        aws.String("reconfigureio-builds"),           // Required
				Key:           aws.String(c.Param("id") + "/bundle.tar.gz"), // Required
				Body:          bytes.NewReader(body.Bytes()),
				ContentLength: aws.Int64(c.Request.ContentLength),
			}
			_, err = session.PutObject(putParams)
			if err != nil {
				c.AbortWithStatus(500)
				c.Error(err)
				return
			}

			params := &batch.SubmitJobInput{
				JobDefinition: aws.String("sdaccel-builder-build"), // Required
				JobName:       aws.String("example"),               // Required
				JobQueue:      aws.String("build-jobs"),            // Required
				ContainerOverrides: &batch.ContainerOverrides{
					Environment: []*batch.KeyValuePair{
						{
							Name:  aws.String("PART"),
							Value: aws.String("xcvu9p-flgb2104-2-i-es2"),
						},
						{
							Name:  aws.String("PART_FAMILY"),
							Value: aws.String("virtexuplus"),
						},
						{
							Name:  aws.String("INPUT_URL"),
							Value: aws.String("s3://reconfigureio-builds/" + c.Param("id") + "/bundle.tar.gz"),
						},
					},
				},
			}
			resp, err := batchSession.SubmitJob(params)
			if err != nil {
				c.AbortWithStatus(500)
				c.Error(err)
				return
			}

			c.JSON(200, resp)
		}
	})

	// Log streaming test
	r.GET("/builds/:id/logs", func(c *gin.Context) {
		id := c.Params.ByName("id")
		cwLogs := cloudwatchlogs.New(awsSession)
		batchSession := batch.New(awsSession)

		getJobStatus := func() (*batch.JobDetail, error) {
			inp := &batch.DescribeJobsInput{Jobs: []*string{&id}}
			resp, err := batchSession.DescribeJobs(inp)
			if err != nil {
				return nil, err
			}
			if len(resp.Jobs) == 0 {
				return nil, nil
			}
			return resp.Jobs[0], nil
		}

		job, err := getJobStatus()
		if err != nil {
			c.AbortWithStatus(500)
			c.Error(err)
			return
		}
		if job == nil {
			c.AbortWithStatus(404)
			return
		}

		log.Printf("found job:  %+v", *job)

		searchParams := &cloudwatchlogs.DescribeLogStreamsInput{
			LogGroupName:        aws.String("/aws/batch/job"), // Required
			Descending:          aws.Bool(true),
			Limit:               aws.Int64(1),
			LogStreamNamePrefix: aws.String("example/" + id),
		}
		resp, err := cwLogs.DescribeLogStreams(searchParams)
		if err != nil {
			c.AbortWithStatus(500)
			c.Error(err)
			return
		}

		if len(resp.LogStreams) == 0 {
			c.AbortWithStatus(404)
			return
		}
		logStream := resp.LogStreams[0]
		log.Printf("opening log stream: %s", *logStream.LogStreamName)

		logs := make(chan *cloudwatchlogs.GetLogEventsOutput)

		// Stop streaming as soon as we get a stop
		stop := make(chan struct{}, 1)
		defer func() {
			stop <- struct{}{}
		}()

		params := (&cloudwatchlogs.GetLogEventsInput{}).
			SetLogGroupName("/aws/batch/job").
			SetLogStreamName(*logStream.LogStreamName).
			SetStartFromHead(true)

		go func() {
			defer func() {
				close(logs)
			}()
			err := cwLogs.GetLogEventsPages(params, func(page *cloudwatchlogs.GetLogEventsOutput, lastPage bool) bool {
				select {
				case logs <- page:
					if lastPage || (len(page.Events) == 0 && (*job.Status) == "FAILED") {
						return false
					}
					if len(page.Events) == 0 {
						time.Sleep(10 * time.Second)
					}
					return true
				case <-stop:
					return false
				}
			})
			if err != nil {
				c.Error(err)
			}
		}()

		c.Stream(func(w io.Writer) bool {
			log, ok := <-logs
			if ok {
				for _, e := range log.Events {
					_, err := bytes.NewBufferString((*e.Message) + "\n").WriteTo(w)
					if err != nil {
						c.Error(err)
						return false
					}
				}
			}
			return ok
		})
	})

	r.PATCH("/builds/:id", func(c *gin.Context) {
		patch := models.PostBuild{}
		c.BindJSON(&patch)
		if c.Param("id") != "" {
			BuildID, err := stringToInt(c.Param("id"), c)
			if err != nil {
				return
			}
			if err := validateBuild(patch, c); err != nil {
				return
			}
			outputbuild := models.Build{}
			db.Where(&models.Build{ID: BuildID}).First(&outputbuild)
			db.Model(&outputbuild).Updates(models.Build{UserID: patch.UserID, ProjectID: patch.ProjectID, InputArtifact: patch.InputArtifact, OutputArtifact: patch.OutputArtifact, OutputStream: patch.OutputStream, Status: patch.Status})
			c.JSON(201, outputbuild)
		}
	})

	r.GET("/builds", func(c *gin.Context) {
		project := c.DefaultQuery("project", "")
		Builds := []models.Build{}
		if project != "" {
			ProjID, err := stringToInt(project, c)
			if err != nil {
				return
			}
			db.Where(&models.Build{ProjectID: ProjID}).Find(&Builds)
		} else {
			db.Find(&Builds)
		}

		c.JSON(200, gin.H{
			"builds": Builds,
		})
	})

	r.GET("/builds/:id", func(c *gin.Context) {
		outputbuild := []models.Build{}
		if c.Param("id") != "" {
			BuildID, err := stringToInt(c.Param("id"), c)
			if err != nil {
				return
			}
			db.Where(&models.Build{ID: BuildID}).First(&outputbuild)
		}
		c.JSON(200, outputbuild)
	})

	r.POST("/projects", func(c *gin.Context) {
		post := models.PostProject{}
		c.BindJSON(&post)
		if err := validateProject(post, c); err != nil {
			return
		}
		newProject := models.Project{UserID: post.UserID, Name: post.Name}
		db.Create(&newProject)
		c.JSON(201, newProject)
	})

	r.PUT("/projects/:id", func(c *gin.Context) {
		post := models.PostProject{}
		c.BindJSON(&post)
		if c.Param("id") != "" {
			ProjID, err := stringToInt(c.Param("id"), c)
			if err != nil {
				return
			}
			if err := validateProject(post, c); err != nil {
				return
			}
			outputproj := models.Project{}
			db.Where(&models.Project{ID: ProjID}).First(&outputproj)
			db.Model(&outputproj).Updates(models.Project{UserID: post.UserID, Name: post.Name})
			c.JSON(201, outputproj)
		}
	})

	r.GET("/projects", func(c *gin.Context) {
		projects := []models.Project{}
		db.Find(&projects)
		c.JSON(200, gin.H{
			"projects": projects,
		})
	})

	r.GET("/projects/:id", func(c *gin.Context) {
		outputproj := []models.Project{}
		if c.Param("id") != "" {
			ProjID, err := stringToInt(c.Param("id"), c)
			if err != nil {
				return
			}
			db.Where(&models.Project{ID: ProjID}).First(&outputproj)
		}
		c.JSON(200, outputproj)
	})

	r.POST("/simulations", func(c *gin.Context) {
		post := models.PostSimulation{}
		c.BindJSON(&post)

		if err := validateSimulation(post, c); err != nil {
			return
		}
		newSim := models.Simulation{UserID: post.UserID, ProjectID: post.ProjectID}
		db.Create(&newSim)
		c.JSON(201, newSim)
	})

	r.PUT("/simulations/:id", func(c *gin.Context) {
		post := models.PostSimulation{}
		c.BindJSON(&post)
		if c.Param("id") != "" {
			SimID, err := stringToInt(c.Param("id"), c)
			if err != nil {
				return
			}
			if err := validateSimulation(post, c); err != nil {
				return
			}
			outputsim := models.Simulation{}
			db.Where(&models.Simulation{ID: SimID}).First(&outputsim)
			db.Model(&outputsim).Updates(models.Simulation{UserID: post.UserID, ProjectID: post.ProjectID, InputArtifact: post.InputArtifact, OutputStream: post.OutputStream, Status: post.Status})
			c.JSON(201, outputsim)
		}
	})

	r.PUT("/simulations/:id/input", func(c *gin.Context) {
		session := s3.New(awsSession)
		batchSession := batch.New(awsSession)

		if c.Param("id") != "" {
			id, err := stringToInt(c.Param("id"), c)
			if err != nil {
				return
			}

			cursim := models.Simulation{}
			db.Where(&models.Simulation{ID: id}).First(&cursim)

			// This is bad and buffers the entire body in memory :(
			body := bytes.Buffer{}
			body.ReadFrom(c.Request.Body)

			const bucket = "reconfigureio-builds"
			key := "simulations/" + c.Param("id") + "/bundle.tar.gz"

			putParams := &s3.PutObjectInput{
				Bucket:        aws.String(bucket), // Required
				Key:           aws.String(key),    // Required
				Body:          bytes.NewReader(body.Bytes()),
				ContentLength: aws.Int64(c.Request.ContentLength),
			}
			_, err = session.PutObject(putParams)
			if err != nil {
				c.AbortWithStatus(500)
				c.Error(err)
				return
			}

			params := &batch.SubmitJobInput{
				JobDefinition: aws.String("sdaccel-builder-build"), // Required
				JobName:       aws.String("example"),               // Required
				JobQueue:      aws.String("build-jobs"),            // Required
				ContainerOverrides: &batch.ContainerOverrides{
					Command: []*string{
						aws.String("/opt/simulate.sh"),
					},
					Environment: []*batch.KeyValuePair{
						{
							Name:  aws.String("PART"),
							Value: aws.String("xcvu9p-flgb2104-2-i-es2"),
						},
						{
							Name:  aws.String("PART_FAMILY"),
							Value: aws.String("virtexuplus"),
						},
						{
							Name:  aws.String("INPUT_URL"),
							Value: aws.String("s3://" + bucket + "/" + key),
						},
						{
							Name:  aws.String("CMD"),
							Value: aws.String(cursim.Command),
						},
						{
							Name:  aws.String("DEVICE"),
							Value: aws.String("xilinx_adm-pcie-ku3_2ddr-xpr_3_3"),
						},
						{
							Name:  aws.String("DEVICE_FULL"),
							Value: aws.String("xilinx:adm-pcie-ku3:2ddr-xpr:3.3"),
						},
					},
				},
			}
			resp, err := batchSession.SubmitJob(params)
			if err != nil {
				c.AbortWithStatus(500)
				c.Error(err)
				return
			}

			c.JSON(200, resp)
		}
	})

	r.PATCH("/simulations/:id", func(c *gin.Context) {
		patch := models.PostSimulation{}
		c.BindJSON(&patch)
		if c.Param("id") != "" {
			SimID, err := stringToInt(c.Param("id"), c)
			if err != nil {
				return
			}
			if err := validateSimulation(patch, c); err != nil {
				return
			}
			outputsim := models.Simulation{}
			db.Where(&models.Simulation{ID: SimID}).First(&outputsim)
			db.Model(&outputsim).Updates(models.Simulation{UserID: patch.UserID, ProjectID: patch.ProjectID, InputArtifact: patch.InputArtifact, OutputStream: patch.OutputStream, Status: patch.Status})
			c.JSON(201, outputsim)
		}
	})

	r.GET("/simulations", func(c *gin.Context) {
		project := c.DefaultQuery("project", "")
		Simulations := []models.Simulation{}
		if project != "" {
			ProjID, err := stringToInt(project, c)
			if err != nil {
				return
			}
			db.Where(&models.Simulation{ProjectID: ProjID}).Find(&Simulations)
		} else {
			db.Find(&Simulations)
		}

		c.JSON(200, gin.H{
			"simulations": Simulations,
		})
	})

	r.GET("/simulations/:id", func(c *gin.Context) {
		outputsim := []models.Simulation{}
		if c.Param("id") != "" {
			simulationID, err := stringToInt(c.Param("id"), c)
			if err != nil {
				return
			}
			db.Where(&models.Simulation{ID: simulationID}).First(&outputsim)
		}
		c.JSON(200, outputsim)
	})

	// Log streaming test
	r.GET("/simulations/:id/logs", func(c *gin.Context) {
		id := c.Params.ByName("id")
		cwLogs := cloudwatchlogs.New(awsSession)
		batchSession := batch.New(awsSession)

		getJobStatus := func() (*batch.JobDetail, error) {
			inp := &batch.DescribeJobsInput{Jobs: []*string{&id}}
			resp, err := batchSession.DescribeJobs(inp)
			if err != nil {
				return nil, err
			}
			if len(resp.Jobs) == 0 {
				return nil, nil
			}
			return resp.Jobs[0], nil
		}

		job, err := getJobStatus()
		if err != nil {
			c.AbortWithStatus(500)
			c.Error(err)
			return
		}
		if job == nil {
			c.AbortWithStatus(404)
			return
		}

		log.Printf("found job:  %+v", *job)

		searchParams := &cloudwatchlogs.DescribeLogStreamsInput{
			LogGroupName:        aws.String("/aws/batch/job"), // Required
			Descending:          aws.Bool(true),
			Limit:               aws.Int64(1),
			LogStreamNamePrefix: aws.String("example/" + id),
		}
		resp, err := cwLogs.DescribeLogStreams(searchParams)
		if err != nil {
			c.AbortWithStatus(500)
			c.Error(err)
			return
		}

		if len(resp.LogStreams) == 0 {
			c.AbortWithStatus(404)
			return
		}
		logStream := resp.LogStreams[0]
		log.Printf("opening log stream: %s", *logStream.LogStreamName)

		logs := make(chan *cloudwatchlogs.GetLogEventsOutput)

		// Stop streaming as soon as we get a stop
		stop := make(chan struct{}, 1)
		defer func() {
			stop <- struct{}{}
		}()

		params := (&cloudwatchlogs.GetLogEventsInput{}).
			SetLogGroupName("/aws/batch/job").
			SetLogStreamName(*logStream.LogStreamName).
			SetStartFromHead(true)

		go func() {
			defer func() {
				close(logs)
			}()
			err := cwLogs.GetLogEventsPages(params, func(page *cloudwatchlogs.GetLogEventsOutput, lastPage bool) bool {
				select {
				case logs <- page:
					if lastPage || (len(page.Events) == 0 && (*job.Status) == "FAILED") {
						return false
					}
					if len(page.Events) == 0 {
						time.Sleep(10 * time.Second)
					}
					return true
				case <-stop:
					return false
				}
			})
			if err != nil {
				c.Error(err)
			}
		}()

		c.Stream(func(w io.Writer) bool {
			log, ok := <-logs
			if ok {
				for _, e := range log.Events {
					_, err := bytes.NewBufferString((*e.Message) + "\n").WriteTo(w)
					if err != nil {
						c.Error(err)
						return false
					}
				}
			}
			return ok
		})
	})

	// Listen and Server in 0.0.0.0:$PORT
	r.Run(":" + port)
}

func stringToInt(s string, c *gin.Context) (int, error) {
	i, err := strconv.Atoi(s)
	if err != nil {
		c.AbortWithStatus(404)
		return 0, NOT_FOUND
	} else {
		return i, nil
	}
}

func validateBuild(postb models.PostBuild, c *gin.Context) error {
	if err := validator.Validate(&postb); err != nil {
		c.AbortWithStatus(404)
		return err
	} else {
		return nil
	}
}

func validateProject(postp models.PostProject, c *gin.Context) error {
	if err := validator.Validate(&postp); err != nil {
		c.AbortWithStatus(404)
		return err
	} else {
		return nil
	}
}

func validateSimulation(posts models.PostSimulation, c *gin.Context) error {
	if err := validator.Validate(&posts); err != nil {
		c.AbortWithStatus(404)
		return err
	} else {
		return nil
	}
}
