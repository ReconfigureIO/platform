package queue

import (
	"log"
	"sync"
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
	once sync.Once
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
	d.service.Push(d.jobType, job.ID, job.Weight)
}

func (d *dbQueue) Start() {
	d.halt = make(chan struct{})

	ticker := time.NewTicker(time.Second * 1)

	// pick jobs in limbo and re-queue them.
	d.once.Do(func() {
		stuckJobs, err := d.service.FetchWithStatus(d.jobType, models.StatusStarted)
		if err != nil {
			log.Println(err)
			return
		}
		for _, sj := range stuckJobs {
			err := d.service.Update(d.jobType, sj, models.StatusQueued)
			if err != nil {
				log.Println(err)
			}
		}
	})

loop:
	for {
		select {
		case <-d.halt:
			break loop
		case <-ticker.C:
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
