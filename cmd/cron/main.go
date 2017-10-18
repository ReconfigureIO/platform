package main

import (
	"context"
	"os"
	"time"

	"github.com/ReconfigureIO/platform/config"
	"github.com/ReconfigureIO/platform/handlers/api"
	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/afi_watcher"
	"github.com/ReconfigureIO/platform/service/aws"
	"github.com/ReconfigureIO/platform/service/billing_hours"
	"github.com/ReconfigureIO/platform/service/deployment"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/robfig/cron"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	stripe "github.com/stripe/stripe-go"
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

	db *gorm.DB

	RootCmd = &cobra.Command{
		Use:              "worker",
		Short:            "The worker for reconfigure.io's platform",
		PersistentPreRun: setup,
	}

	version string
)

func setup(*cobra.Command, []string) {
	conf, err := config.ParseEnvConfig()
	if err != nil {
		log.Fatal(err)
	}

	err = config.SetupLogging(version, conf)
	if err != nil {
		log.Fatal(err)
	}

	db := config.SetupDB(conf)
	api.DB(db)

	if err != nil {
		log.Fatalf("failed to connect to database: %s", err.Error())
	}

}

// add commands to root command
func main() {
	conf, err := config.ParseEnvConfig()
	if err != nil {
		log.Fatal(err)
	}
	stripe.Key = conf.StripeKey

	RootCmd.AddCommand(commands...)

	if err := RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var commands = []*cobra.Command{
	// health
	&cobra.Command{
		Use:   "health",
		Short: "Check platform health",
		Run: func(*cobra.Command, []string) {
			healthCmd()
		},
	},
	// cron
	&cobra.Command{
		Use:   "cron",
		Short: "Start cron worker",
		Run: func(*cobra.Command, []string) {
			cronCmd()
		},
	},
}

func healthCmd() {
	if err := db.DB().Ping(); err != nil {
		exitWithErr("error connecting to db")
	}
}

func cronCmd() {
	worker := cron.New()
	schedule := func(d time.Duration, f func()) {
		worker.Schedule(cron.Every(d), cron.FuncJob(f))
	}

	schedule(5*time.Minute, generatedAFIs)
	schedule(time.Minute, terminateDeployments)
	schedule(time.Minute, checkHours)
	schedule(time.Hour, updateDebits)

	worker.Start()
	log.Printf("starting workers")

	waitForever := make(chan struct{})
	<-waitForever

}

func terminateDeployments() {
	log.Printf("terminating deployments")
	d := models.DeploymentDataSource(db)
	ctx := context.Background()

	err := deployment.NewInstances(d, deploy).UpdateInstanceStatus(ctx)

	if err != nil {
		exitWithErr(err)
	}
}

func generatedAFIs() {
	log.Printf("checking afis")
	watcher := afi_watcher.NewAFIWatcher(models.BuildDataSource(db), awsService, models.BatchDataSource(db))

	err := watcher.FindAFI(context.Background(), 100)
	if err != nil {
		exitWithErr(err)
	}
}

func checkHours() {
	log.Printf("checking deployments")
	err := billing_hours.CheckUserHours(models.SubscriptionDataSource(db), models.DeploymentDataSource(db), deploy)
	if err != nil {
		exitWithErr(err)
	}
}

func updateDebits() {
	log.Printf("updating user debits")
	err := billing_hours.UpdateDebits(models.UserBalanceDataSource(db), models.DeploymentDataSource(db), time.Now())
	if err != nil {
		exitWithErr(err)
	}
}

func exitWithErr(err interface{}) {
	log.Println(err)
	os.Exit(1)
}
