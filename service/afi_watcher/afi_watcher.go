package afi_watcher

import (
	"github.com/ReconfigureIO/platform/handlers/api"
	"github.com/ReconfigureIO/platform/models"
)

func FindAfi(d models.BuildRepo, awsService aws, batch api.BatchService) error {
	//get list of builds waiting for AFI generation to finish
	buildswaitingonafis, err := d.GetBuildsWithStatus([]string{models.StatusCreatingImage}, 100)
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
		status, found := statuses[build.FPGAImage.AFIID]
		if found {
			switch status {
			case "available":
				event := models.PostBatchEvent{
					Status:  models.StatusCompleted,
					Message: models.StatusCompleted,
					Code:    0,
				}
			case "failed":
				event := models.PostBatchEvent{
					Status:  models.StatusErrored,
					Message: models.StatusErrored,
					Code:    0,
				}
			default:
			}
			_, err := batch.AddEvent(&build.BatchJob, event)
			if err != nil {
				return err
			}
			afigenerated += 1
		}
	}

	log.Printf("%d builds have finished generating AFIs", afigenerated)
	return nil

}
