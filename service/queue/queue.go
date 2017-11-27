package queue

import (
	"github.com/ReconfigureIO/platform/models"
)

// Queue is a job queue.
type Queue interface {
	// Push adds an entry to the queue.
	Push(Job)
	// Start starts and monitors the queue.
	// Jobs are dispatched to the job runner accordingly.
	// This blocks forever.
	Start()
	// Halt stops the queue from dispatching jobs to
	// job runner.
	// Halt should only be called after the Queue has been
	// started. i.e. Start has been previously called.
	Halt()
	// CountUserJobsInstatus counts the amount of user jobs
	// in a status.
	CountUserJobsInStatus(user models.User, status string) (int, error)
}

// JobRunner manage jobs in the queue.
type JobRunner interface {
	// Run runs the job.
	// This function is expected to block until the job has finished running.
	// The queue treats the job as done and move on to the next
	// job on return of this function.
	Run(Job)
	// Stop stops the job.
	Stop(Job)
}

// NewJobRunner creates a new JobRunner with start and stop functions.
func NewJobRunner(run, stop func(Job)) JobRunner {
	return jobRunner{
		run:  run,
		stop: stop,
	}
}

type jobRunner struct{ run, stop func(j Job) }

func (r jobRunner) Run(j Job)  { r.run(j) }
func (r jobRunner) Stop(j Job) { r.stop(j) }

// Job is queue entry.
type Job struct {
	ID     string
	Weight int
	User   models.User
}
