package queue

import (
	"log"
	"time"

	"github.com/ReconfigureIO/platform/handlers/api"
	"github.com/ReconfigureIO/platform/models"
)

var _ Queue = &dbQueue{}

type dbQueue struct {
	jobType    string
	runner     JobRunner
	concurrent int

	halt chan struct{}
}

// NewWithDBStore creates a new queue using database as storage for queue state.
func NewWithDBStore(runner JobRunner, concurrent int, jobType string) Queue {
	return &dbQueue{
		jobType:    jobType,
		runner:     runner,
		concurrent: concurrent,
	}
}

func (d *dbQueue) Push(job Job) {
	api.Queue{}.Push(d.jobType, job.ID, job.Weight)
}

func (d *dbQueue) Start() {
	d.halt = make(chan struct{})
	stop := false

	go func() {
		<-d.halt
		stop = true
	}()

	qapi := api.Queue{}

	for !stop {
		time.Sleep(time.Second * 10)
		dispatched, err := qapi.Count(d.jobType, models.StatusStarted)
		if err != nil {
			log.Println(err)
			continue
		}
		toRun := d.concurrent - dispatched
		if toRun > 0 {
			jobs, err := qapi.Fetch(d.jobType, toRun)
			if err != nil {
				log.Println(err)
				continue
			}

			for _, jobID := range jobs {
				go d.dispatch(jobID)
			}

		}
	}
}

func (d *dbQueue) Halt() {
	close(d.halt)
}

func (d *dbQueue) dispatch(jobID string) {
	// run job
	qapi := api.Queue{}
	err := qapi.Update(d.jobType, jobID, models.StatusStarted)
	if err != nil {
		log.Println(err)
		return
	}

	d.runner.Run(Job{ID: jobID})

	err = qapi.Update(d.jobType, jobID, models.StatusCompleted)
	if err != nil {
		log.Println(err)
	}
}
