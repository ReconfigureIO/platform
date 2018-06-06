package main

import (
	"context"
	"os"
	"time"

	"github.com/ReconfigureIO/platform/config"
	"github.com/ReconfigureIO/platform/handlers/api"
	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/aws"
	"github.com/ReconfigureIO/platform/service/billing_hours"
	"github.com/ReconfigureIO/platform/service/cw_id_watcher"
	"github.com/ReconfigureIO/platform/service/deployment"
	"github.com/ReconfigureIO/platform/service/fpgaimage/afi/afiwatcher"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/robfig/cron"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	stripe "github.com/stripe/stripe-go"
)

var (
	deploy     deployment.Service
	awsService aws.Service

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
	stripe.Key = conf.StripeKey

	err = config.SetupLogging(version, conf)
	if err != nil {
		log.Fatal(err)
	}

	deploy = deployment.New(conf.Reco.Deploy)
	awsService = aws.New(conf.Reco.AWS)

	db = config.SetupDB(conf)
	api.DB(db)
}

// add commands to root command
func main() {
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
	schedule(5*time.Minute, getBatchJobLogNames)
	schedule(time.Minute, terminateDeployments)
	schedule(time.Minute, checkHours)
	schedule(time.Minute, findDeploymentIPs)

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
		log.WithError(err).Error("Errored while marking deployments as terminated")
	}
}

func findDeploymentIPs() {
	log.Printf("finding the IPs of deployments")
	d := models.DeploymentDataSource(db)
	ctx := context.Background()

	err := deployment.NewInstances(d, deploy).FindIPs(ctx)

	if err != nil {
		log.WithError(err).Error("Errored while finding deployment IPs")
	}
}

func generatedAFIs() {
	log.Printf("checking afis")
	watcher := afiwatcher.AFIWatcher{
		BatchRepo:           models.BatchDataSource(db),
		BuildRepo:           models.BuildDataSource(db),
		DescribeAFIStatuser: awsService,
	}

	err := watcher.FindAFI(context.Background(), 100)
	if err != nil {
		log.WithError(err).Error("Errored while checking for generated AFIs")
	}
}

func getBatchJobLogNames() {
	log.Printf("Getting log names")
	watcher := cw_id_watcher.NewLogWatcher(awsService, models.BatchDataSource(db))

	// find batch jobs that've become active in the last hour
	sinceTime := time.Now().Add(-1 * time.Hour)
	err := watcher.FindLogNames(context.Background(), 100, sinceTime)
	if err != nil {
		log.WithError(err).Error("Errored while reading batch job log names")
	}
}

func checkHours() {
	log.Printf("checking for users exceeding their subscription hours")
	err := billing_hours.CheckUserHours(models.SubscriptionDataSource(db), models.DeploymentDataSource(db), deploy)
	if err != nil {
		log.WithError(err).Error("Errored while checking users have not exceeded their hour allowances")
	}
}

func exitWithErr(err interface{}) {
	log.Println(err)
	os.Exit(1)
}
