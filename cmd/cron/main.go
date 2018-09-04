package main

import (
	"context"
	"os"
	"time"

	"github.com/ReconfigureIO/platform/config"
	"github.com/ReconfigureIO/platform/handlers/api"
	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/batch/aws"
	"github.com/ReconfigureIO/platform/service/batch/aws/logs/cloudwatch"
	"github.com/ReconfigureIO/platform/service/billing_hours"
	"github.com/ReconfigureIO/platform/service/deployment"
	"github.com/ReconfigureIO/platform/service/fpgaimage/afi"
	"github.com/ReconfigureIO/platform/service/fpgaimage/afi/afiwatcher"
	awsaws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/batch"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/robfig/cron"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	stripe "github.com/stripe/stripe-go"
)

var (
	deploy      deployment.Service
	awsService  aws.Service
	batchClient *batch.Batch

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
	awsService = *aws.New(conf.Reco.AWS, &cloudwatch.Service{
		LogGroup: conf.Reco.AWS.LogGroup,
	})

	batchClient = batch.New(session.Must(session.NewSession(awsaws.NewConfig().WithRegion("us-east-1"))))

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
		BatchRepo:        models.BatchDataSource(db),
		BuildRepo:        models.BuildDataSource(db),
		FPGAImageService: &afi.Service{},
	}

	err := watcher.FindAFI(context.Background(), 100)
	if err != nil {
		log.WithError(err).Error("Errored while checking for generated AFIs")
	}
}

func getBatchJobLogNames() {
	log.Printf("Getting log names")
	watcher := NewLogWatcher(models.BatchDataSource(db), batchClient)

	// find batch jobs that've become active in the last hour
	sinceTime := time.Now().Add(-1 * time.Hour)
	err := watcher.FindLogNames(context.Background(), 100, sinceTime)
	if err != nil {
		log.WithError(err).Error("Errored while reading batch job log names")
	}
}

// checkBatchJobRunningStatus gets a list of all batch jobs we think are in a
// running state and then queries AWS Batch for the true state of these jobs.
// Jobs can transition from running states to errored states without sending an
// event to platform in some cases e.g. when the underlying instance terminates
func checkBatchJobRunningStatus() {
	//watcher := NewLogWatcher(models.BatchDataSource(db), batchClient)
	batchRepo := models.BatchDataSource(db)

	batchJobs, err := batchRepo.GetBatchJobsWithStatus([]string{models.StatusStarted}, 100)
	if err != nil {
		log.WithError(err).Error("Errored while finding batch jobs in a running state")
	}

	var batchJobIDs []string
	for _, batchJob := range batchJobs {
		batchJobIDs = append(batchJobIDs, batchJob.BatchID)
	}

	// err := watcher.FindLogNames(context.Background(), 100, sinceTime)
	cfg := batch.DescribeJobsInput{
		Jobs: awsaws.StringSlice(batchJobIDs),
	}

	results, err := batchClient.DescribeJobs(&cfg)
	if err != nil {
		log.WithError(err).Error("Errored while running AWS Batch DescribeJobs")
	}

	for _, job := range results.Jobs {
		if *job.Status == "FAILED" {
			for _, batchJob := range batchJobs {
				if batchJob.BatchID == *job.JobId {
					err = batchRepo.AddEvent(batchJob, models.BatchJobEvent{
						BatchJobID: batchJob.ID,
						Timestamp:  time.Unix(*job.StoppedAt, 0),
						Status:     models.StatusErrored,
						Message:    *job.StatusReason,
					})
					if err != nil {
						log.WithField("batch_job_id", batchJob.ID).WithError(err).Error("Errored while adding FAILED event")
					}
				}
			}
		}
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
