// +simulation integration

package models

import (
	"testing"
	"time"

	"github.com/jinzhu/gorm"
)

// genSimulation generates a mock simulation.
// if duration > 0, the mock simulation will be in TERMINATED
// status and have a duration of `duration`.
func genSimulation(userID string, duration time.Duration) Simulation {
	var start = (time.Unix(0, 0)).Add(time.Hour)
	simulation := Simulation{
		Project: Project{
			UserID: userID,
		},
		BatchJob: BatchJob{
			Events: []BatchJobEvent{
				BatchJobEvent{
					Status:    "STARTED",
					Timestamp: start,
				},
			},
		},
	}
	if duration > 0 {
		simulation.BatchJob.Events = append(simulation.BatchJob.Events, BatchJobEvent{
			Status:    "TERMINATED",
			Timestamp: start.Add(duration),
		})
	}
	return simulation
}

func TestActiveSimulations(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		simulationData := simulationRepo{db}

		user := User{}

		simulations := []Simulation{
			genSimulation(user.ID, time.Hour),
			genSimulation(user.ID, 0),
			genSimulation(user.ID, 0),
			genSimulation(user.ID, time.Hour*2),
		}

		for i := range simulations {
			db.Create(&(simulations[i]))
		}

		activeSimulations, err := simulationData.ActiveSimulations(user)
		if err != nil {
			t.Error(err)
			return
		}
		if l := len(activeSimulations); l != 2 {
			t.Errorf("Expected %v found %v", 2, l)
		}
	})
}
