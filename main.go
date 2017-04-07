package main

import (
	"errors"
	"fmt"
	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/aws"
	"github.com/ReconfigureIO/platform/service/stream"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"gopkg.in/validator.v2"
	"log"
	"os"
	"strconv"
	"time"
)

var NOT_FOUND = errors.New("Not Found")

type ApiError struct {
	Error string `json:"error"`
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

	db.AutoMigrate(&models.Simulation{})
	db.AutoMigrate(&models.Build{})
	db.AutoMigrate(&models.Project{})
	db.AutoMigrate(&models.User{})

	awsSession := aws.New(aws.ServiceConfig{
		Bucket:        "reconfigureio-builds",
		Queue:         "build-jobs",
		JobDefinition: "sdaccel-builder-build",
	})

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
		id, err := stringToInt(c.Param("id"), c)
		if err != nil {
			return
		}

		build := models.Build{}
		db.First(&build, id)

		if build.Status != "SUBMITTED" {
			c.JSON(400, ApiError{
				Error: fmt.Sprintf("Build is '%s', not SUBMITTED", build.Status),
			})
			return
		}

		key := fmt.Sprintf("builds/%d/simulation.tar.gz", id)

		s3Url, err := awsSession.Upload(key, c.Request.Body, c.Request.ContentLength)

		if err != nil {
			c.AbortWithStatus(500)
			c.Error(err)
			return
		}

		buildId, err := awsSession.RunBuild(s3Url)

		if err != nil {
			c.AbortWithStatus(500)
			c.Error(err)
			return
		}

		db.Model(&build).Updates(models.Build{BatchId: buildId, Status: "QUEUED"})
		c.JSON(200, build)
	})

	// Log streaming test
	r.GET("/builds/:id/logs", func(c *gin.Context) {
		id, err := stringToInt(c.Param("id"), c)
		if err != nil {
			return
		}

		build := models.Build{}
		// check for error here
		db.First(&build, id)

		for !build.HasStarted() {
			time.Sleep(time.Second)
			db.First(&build, id)
		}

		buildId := build.BatchId

		logStream, err := awsSession.GetJobStream(buildId)

		if err != nil {
			c.AbortWithStatus(500)
			c.Error(err)
			return
		}

		log.Printf("opening log stream: %s", *logStream.LogStreamName)

		lstream := awsSession.NewStream(*logStream)

		go func() {
			for !build.HasFinished() {
				time.Sleep(10 * time.Second)
				db.First(&build, id)
			}
			lstream.Ended = true
		}()

		stream.Stream(lstream, c)
	})

	r.PATCH("/builds/:id", func(c *gin.Context) {
		patch := models.PostBuild{}
		c.BindJSON(&patch)
		if err := validateBuild(patch, c); err != nil {
			return
		}

		BuildID, err := stringToInt(c.Param("id"), c)
		if err != nil {
			return
		}

		outputbuild := models.Build{}
		db.Where(&models.Build{ID: BuildID}).First(&outputbuild)
		db.Model(&outputbuild).Updates(models.Build{UserID: patch.UserID, ProjectID: patch.ProjectID, InputArtifact: patch.InputArtifact, OutputArtifact: patch.OutputArtifact, OutputStream: patch.OutputStream, Status: patch.Status})
		c.JSON(201, outputbuild)
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
		id, err := stringToInt(c.Param("id"), c)
		if err != nil {
			return
		}

		sim := models.Simulation{}
		db.First(&sim, id)

		if sim.Status != "SUBMITTED" {
			c.JSON(400, ApiError{
				Error: fmt.Sprintf("Simulation is '%s', not SUBMITTED", sim.Status),
			})
			return
		}

		key := fmt.Sprintf("simulation/%d/simulation.tar.gz", id)

		s3Url, err := awsSession.Upload(key, c.Request.Body, c.Request.ContentLength)

		if err != nil {
			c.AbortWithStatus(500)
			c.Error(err)
			return
		}

		simId, err := awsSession.RunSimulation(s3Url, sim.Command)

		if err != nil {
			c.AbortWithStatus(500)
			c.Error(err)
			return
		}

		db.Model(&sim).Updates(models.Simulation{BatchId: simId, Status: "QUEUED"})
		c.JSON(200, sim)
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
		id, err := stringToInt(c.Param("id"), c)
		if err != nil {
			return
		}

		sim := models.Simulation{}
		// check for error here
		db.First(&sim, id)

		for !sim.HasStarted() {
			time.Sleep(time.Second)
			db.First(&sim, id)
		}

		simId := sim.BatchId

		logStream, err := awsSession.GetJobStream(simId)

		if err != nil {
			c.AbortWithStatus(500)
			c.Error(err)
			return
		}

		log.Printf("opening log stream: %s", *logStream.LogStreamName)

		lstream := awsSession.NewStream(*logStream)

		go func() {
			for !sim.HasFinished() {
				time.Sleep(10 * time.Second)
				db.First(&sim, id)
			}
			lstream.Ended = true
		}()

		stream.Stream(lstream, c)
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
