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

func FindAFI(d models.BuildRepo, awsService aws.Service, batch models.BatchRepo) error {
	//get list of builds waiting for AFI generation to finish
	buildswaitingonafis, err := d.GetBuildsWithStatus(creating_statuses, 100)
	if err != nil {
		return err
	}
	log.Printf("Looking up %d builds", len(buildswaitingonafis))

	if len(buildswaitingonafis) == 0 {
		return nil
	}
	//get the status of the associated AFIs
	statuses, err := awsService.DescribeAFIStatus(context.Background(), buildswaitingonafis)
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
			switch status {
			case "available":
				event = &models.BatchJobEvent{
					Timestamp: time.Now(),
					Status:    models.StatusCompleted,
					Message:   models.StatusCompleted,
					Code:      0,
				}
			case "failed":
				event = &models.BatchJobEvent{
					Timestamp: time.Now(),
					Status:    models.StatusErrored,
					Message:   models.StatusErrored,
					Code:      0,
				}
			default:
			}

			if event != nil {
				err := batch.AddEvent(build.BatchJob, *event)
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
