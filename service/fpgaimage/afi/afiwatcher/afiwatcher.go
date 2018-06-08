package afiwatcher

import (
	"context"

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/fpgaimage"
	log "github.com/sirupsen/logrus"
)

var (
	statusCreating = []string{models.StatusCreatingImage}
)

// AFIWatcher considers builds which aren't yet finished, and looks for their AFIs.
// When it finds them, the build is marked as completed or errored.
type AFIWatcher struct {
	BatchRepo        models.BatchRepo
	BuildRepo        models.BuildRepo
	FPGAImageService fpgaimage.Service
}

// FindAFI searches for 'limit' builds which are in the StatusCreatingImage state,
// and searches for their AFIs. When found, the build is marked as completed or errored.
func (watcher *AFIWatcher) FindAFI(ctx context.Context, limit int) error {
	// get list of builds waiting for AFI generation to finish
	buildsWaitingOnAFIs, err := watcher.BuildRepo.GetBuildsWithStatus(statusCreating, limit)
	if err != nil {
		return err
	}
	log.Printf("Looking up %d builds", len(buildsWaitingOnAFIs))

	if len(buildsWaitingOnAFIs) == 0 {
		return nil
	}

	// get the status of the associated AFIs
	statuses, err := watcher.FPGAImageService.DescribeAFIStatus(ctx, buildsWaitingOnAFIs)
	if err != nil {
		return err
	}

	log.Printf("statuses of %v", statuses)
	afiGenerated := 0

	// for each build check associated AFI, if done, post event
	for _, build := range buildsWaitingOnAFIs {
		status, found := statuses[build.FPGAImage]
		if !found {
			continue
		}

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

		afiGenerated++
	}

	log.Printf("%d builds have finished generating AFIs", afiGenerated)
	return nil
}
