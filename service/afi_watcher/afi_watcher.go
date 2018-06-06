package afi_watcher

import (
	"context"

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/aws"
	log "github.com/sirupsen/logrus"
)

var (
	creating_statuses = []string{models.StatusCreatingImage}
)

type AFIWatcher struct {
	BuildRepo  models.BuildRepo
	awsService aws.Service
	BatchRepo  models.BatchRepo
}

func NewAFIWatcher(d models.BuildRepo, awsService aws.Service, batch models.BatchRepo) *AFIWatcher {
	w := AFIWatcher{
		BuildRepo:  d,
		awsService: awsService,
		BatchRepo:  batch,
	}
	return &w
}

func (watcher *AFIWatcher) FindAFI(ctx context.Context, limit int) error {
	// get list of builds waiting for AFI generation to finish
	buildswaitingonafis, err := watcher.BuildRepo.GetBuildsWithStatus(creating_statuses, limit)
	if err != nil {
		return err
	}
	log.Printf("Looking up %d builds", len(buildswaitingonafis))

	if len(buildswaitingonafis) == 0 {
		return nil
	}

	// get the status of the associated AFIs
	statuses, err := watcher.awsService.DescribeAFIStatus(ctx, buildswaitingonafis)
	if err != nil {
		return err
	}

	log.Printf("statuses of %v", statuses)
	afigenerated := 0

	// for each build check associated AFI, if done, post event
	for _, build := range buildswaitingonafis {
		status, found := statuses[build.FPGAImage]
		if found {
			var event *models.BatchJobEvent
			switch status.Status {
			case "available":
				event = &models.BatchJobEvent{
					Timestamp: status.UpdatedAt,
					Status:    models.StatusCompleted,
					Message:   "Image generation succeeded",
					Code:      0,
				}
			case "failed":
				event = &models.BatchJobEvent{
					Timestamp: status.UpdatedAt,
					Status:    models.StatusErrored,
					Message:   "Image generation failed",
					Code:      0,
				}
			default:
			}

			if event != nil {
				err := watcher.BatchRepo.AddEvent(build.BatchJob, *event)
				if err != nil {
					return err
				}
			}

			afigenerated += 1
		}
	}

	log.Printf("%d builds have finished generating AFIs", afigenerated)
	return nil
}
