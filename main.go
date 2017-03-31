package main

import (
	"bytes"
	"errors"
	"fmt"
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
)

var NOT_FOUND = errors.New("Not Found")

type User struct {
	ID         int         `gorm:"primary_key" json:"id"`
	GithubID   string      `json:"github_id"`
	Email      string      `gorm:"type:varchar(100);unique_index" json:"email"`
	AuthTokens []AuthToken `json:"auth_token"` //User has many AuthTokens
}

type Project struct {
	ID     int     `gorm:"primary_key" json:"id"`
	User   User    `json:"user"` //Project belongs to User
	UserID int     `json:"user_id"`
	Name   string  `json:"name"`
	Builds []Build `json:"builds"`
}

type PostProject struct {
	UserID int    `json:"user_id"`
	Name   string `json:"name"`
}

type AuthToken struct {
	gorm.Model
	Token  string `json:"token"`
	UserID int    `json:"user_id"`
}

type Build struct {
	ID             int     `gorm:"primary_key" json:"id"`
	User           User    `json:"user"` //Build belongs to User, UserID is foreign key
	UserID         int     `json:"user_id"`
	Project        Project `json:"project"`
	ProjectID      int     `json:"project_id"`
	InputArtifact  string  `json:"input_artifact"`
	OutputArtifact string  `json:"output_artifact"`
	OutputStream   string  `json:"output_stream"`
	Status         string  `gorm:"default:'SUBMITTED'" json:"status"`
}

type PostBuild struct {
	UserID         int    `json:"user_id" validate:"min=1"`
	ProjectID      int    `json:"project_id" validate:"min=1"`
	InputArtifact  string `json:"input_artifact"`
	OutputArtifact string `json:"output_artifact"`
	OutputStream   string `json:"output_stream"`
	Status         string `gorm:"default:'SUBMITTED'" json:"status"`
}

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

	awsSession := session.Must(session.NewSession(aws.NewConfig().WithRegion("us-east-1")))

	r := gin.Default()

	// Ping test
	r.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong pong")
	})

	r.POST("/builds", func(c *gin.Context) {
		post := PostBuild{}
		c.BindJSON(&post)

		if err := validateBuild(post, c); err != nil {
			return
		}
		newBuild := Build{UserID: post.UserID, ProjectID: post.ProjectID}
		db.Create(&newBuild)
		c.JSON(201, newBuild)
	})

	r.PUT("/builds/:id", func(c *gin.Context) {
		post := PostBuild{}
		c.BindJSON(&post)
		if c.Param("id") != "" {
			BuildID, err := stringToInt(c.Param("id"), c)
			if err != nil {
				return
			}
			if err := validateBuild(post, c); err != nil {
				return
			}
			outputbuild := Build{}
			db.Where(&Build{ID: BuildID}).First(&outputbuild)
			db.Model(&outputbuild).Updates(Build{UserID: post.UserID, ProjectID: post.ProjectID, InputArtifact: post.InputArtifact, OutputArtifact: post.OutputArtifact, OutputStream: post.OutputStream, Status: post.Status})
			c.JSON(201, outputbuild)
		}
	})

	r.GET("/builds", func(c *gin.Context) {
		project := c.DefaultQuery("project", "")
		Builds := []Build{}
		if project != "" {
			ProjID, err := stringToInt(project, c)
			if err != nil {
				return
			}
			db.Where(&Build{ProjectID: ProjID}).Find(&Builds)
		} else {
			db.Find(&Builds)
		}

		c.JSON(200, gin.H{
			"builds": Builds,
		})
	})

	r.GET("/builds/:id", func(c *gin.Context) {
		outputbuild := []Build{}
		if c.Param("id") != "" {
			BuildID, err := stringToInt(c.Param("id"), c)
			if err != nil {
				return
			}
			db.Where(&Build{ID: BuildID}).First(&outputbuild)
		}
		c.JSON(200, outputbuild)
	})

	r.POST("/projects", func(c *gin.Context) {
		post := PostProject{}
		c.BindJSON(&post)
		if err := validateProject(post, c); err != nil {
			return
		}
		newProject := Project{UserID: post.UserID, Name: post.Name}
		db.Create(&newProject)
		c.JSON(201, newProject)
	})

	r.PUT("/projects/:id", func(c *gin.Context) {
		post := PostProject{}
		c.BindJSON(&post)
		if c.Param("id") != "" {
			ProjID, err := stringToInt(c.Param("id"), c)
			if err != nil {
				return
			}
			if err := validateProject(post, c); err != nil {
				return
			}
			outputproj := Project{}
			db.Where(&Project{ID: ProjID}).First(&outputproj)
			db.Model(&outputproj).Updates(Project{UserID: post.UserID, Name: post.Name})
			c.JSON(201, outputproj)
		}
	})

	r.GET("/projects", func(c *gin.Context) {
		projects := []Project{}
		db.Find(&projects)
		c.JSON(200, gin.H{
			"projects": projects,
		})
	})

	r.GET("/projects/:id", func(c *gin.Context) {
		outputproj := []Project{}
		if c.Param("id") != "" {
			ProjID, err := stringToInt(c.Param("id"), c)
			if err != nil {
				return
			}
			db.Where(&Project{ID: ProjID}).First(&outputproj)
		}
		c.JSON(200, outputproj)
	})

	r.POST("/build/", func(c *gin.Context) {
		session := s3.New(awsSession)
		//		batchSession := batch.New(awsSession)

		// This is bad and buffers the entire body in memory :(
		body := bytes.Buffer{}
		body.ReadFrom(c.Request.Body)

		putParams := &s3.PutObjectInput{
			Bucket:        aws.String("reconfigureio-builds"), // Required
			Key:           aws.String("bundle.tar.gz"),        // Required
			Body:          bytes.NewReader(body.Bytes()),
			ContentLength: aws.Int64(c.Request.ContentLength),
		}
		_, err := session.PutObject(putParams)
		if err != nil {
			c.AbortWithStatus(500)
			c.Error(err)
			return
		}
	})

	// Log streaming test
	r.GET("/build/:id/logs", func(c *gin.Context) {
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

func validateBuild(postb PostBuild, c *gin.Context) error {
	if err := validator.Validate(&postb); err != nil {
		c.AbortWithStatus(404)
		return err
	} else {
		return nil
	}
}

func validateProject(postp PostProject, c *gin.Context) error {
	if err := validator.Validate(&postp); err != nil {
		c.AbortWithStatus(404)
		return err
	} else {
		return nil
	}
}
