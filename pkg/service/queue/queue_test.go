// +build integration

package queue

import (
	"log"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/ReconfigureIO/platform/pkg/models"
	"github.com/jinzhu/gorm"
)

func connectDB() *gorm.DB {
	gormConnDets := os.Getenv("DATABASE_URL")
	db, err := gorm.Open("postgres", gormConnDets)
	if err != nil {
		log.Fatal(err)
	}
	return db
}

var jobs = []Job{
	Job{ID: "1", Weight: 4},
	Job{ID: "2", Weight: 1},
	Job{ID: "3", Weight: 5},
	Job{ID: "4", Weight: 3},
	Job{ID: "5", Weight: 2},
}

func TestDBQueue(t *testing.T) {
	runner := &fakeRunner{}
	var queue = &dbQueue{
		jobType:      "deployment",
		runner:       runner,
		concurrent:   2,
		service:      QueueService{db: connectDB()},
		pollInterval: 10 * time.Millisecond,
		halt:         make(chan struct{}),
	}

	for _, job := range jobs {
		queue.Push(job)
	}
	go queue.Start()

	for i := 0; i < 100; i++ {
		time.Sleep(100 * time.Millisecond)
		if atomic.LoadUint64(&runner.nDispatched) >= 5 {
			queue.Halt()
			break
		}
	}

	for _, job := range jobs {
		if _, ok := runner.dispatched.Load(job.ID); !ok {
			t.Errorf("Job %s not dispatched", job.ID)
		}
	}
}

type fakeRunner struct {
	dispatched  sync.Map
	nDispatched uint64
}

func (f *fakeRunner) Run(job Job) {
	log.Println("starting", job.ID)

	f.dispatched.Store(job.ID, struct{}{})
	atomic.AddUint64(&f.nDispatched, 1)
}

func (f fakeRunner) Stop(job Job) {
	log.Println("stopping", job.ID)
}
