package queue

import (
	"log"
	"time"

	"github.com/ReconfigureIO/platform/models"
	"github.com/jinzhu/gorm"
)

var _ Queue = &dbQueue{}

type dbQueue struct {
	jobType    string
	runner     JobRunner
	concurrent int
	service    QueueService

	halt chan struct{}
}

// NewWithDBStore creates a new queue using database as storage for queue state.
func NewWithDBStore(db *gorm.DB, runner JobRunner, concurrent int, jobType string) Queue {
	return &dbQueue{
		jobType:    jobType,
		runner:     runner,
		concurrent: concurrent,
		service:    QueueService{db: db},
	}
}

func (d *dbQueue) Push(job Job) {
	d.service.Push(d.jobType, job)
}

func (d *dbQueue) CountUserJobsInStatus(user models.User, status string) (int, error) {
	count, err := d.service.CountUserJobsInStatus(d.jobType, user, status)
	return count, err
}

func (d *dbQueue) Start() {
	d.halt = make(chan struct{})
	stop := false

	go func() {
		<-d.halt
		stop = true
	}()

	for !stop {
		time.Sleep(time.Second * 10)
		dispatched, err := d.service.Count(d.jobType, models.StatusStarted)
		if err != nil {
			log.Println(err)
			continue
		}
		toRun := d.concurrent - dispatched
		if toRun > 0 {
			jobs, err := d.service.Fetch(d.jobType, toRun)
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
	err := d.service.Update(d.jobType, jobID, models.StatusStarted)
	if err != nil {
		log.Println(err)
		return
	}

	d.runner.Run(Job{ID: jobID})

	err = d.service.Update(d.jobType, jobID, models.StatusCompleted)
	if err != nil {
		log.Println(err)
	}
}
