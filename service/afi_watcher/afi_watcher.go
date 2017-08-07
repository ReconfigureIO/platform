package afi_watcher

import (
	"context"
	"log"
	"time"

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/aws"
)

var (
	creating_statuses = []string{models.StatusCreatingImage}
)

type AFIWatcher struct {
	d          models.BuildRepo
	awsService aws.Service
	batch      models.BatchRepo
}

func NewAFIWatcher(d models.BuildRepo, awsService aws.Service, batch models.BatchRepo) *AFIWatcher {
	w := AFIWatcher{
		d:          d,
		awsService: awsService,
		batch:      batch,
	}
	return &w
}

func (watcher *AFIWatcher) FindAFI(ctx context.Context, limit int) error {
	//get list of builds waiting for AFI generation to finish
	buildswaitingonafis, err := watcher.d.GetBuildsWithStatus(creating_statuses, limit)
	if err != nil {
		return err
	}
	log.Printf("Looking up %d builds", len(buildswaitingonafis))

	if len(buildswaitingonafis) == 0 {
		return nil
	}

	//get the status of the associated AFIs
	statuses, err := watcher.awsService.DescribeAFIStatus(ctx, buildswaitingonafis)
	if err != nil {
		return err
	}

	log.Printf("statuses of %v", statuses)
	afigenerated := 0

	//for each build check associated AFI, if done, post event
	for _, build := range buildswaitingonafis {
		status, found := statuses[build.FPGAImage]
		if found {
			var event *models.BatchJobEvent
			switch status.Status {
			case "available":
				event = &models.BatchJobEvent{
					Timestamp: status.UpdatedAt,
					Status:    models.StatusCompleted,
					Message:   models.StatusCompleted,
					Code:      0,
				}
			case "failed":
				event = &models.BatchJobEvent{
					Timestamp: status.UpdatedAt,
					Status:    models.StatusErrored,
					Message:   models.StatusErrored,
					Code:      0,
				}
			default:
			}

			if event != nil {
				err := watcher.batch.AddEvent(build.BatchJob, *event)
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
