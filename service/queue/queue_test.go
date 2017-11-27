// +build integration

package queue

import (
	"log"
	"os"
	"sync"
	"testing"
	"time"

	_ "github.com/ReconfigureIO/platform/models"
	"github.com/jinzhu/gorm"
)

func connectDB() *gorm.DB {
	gormConnDets := os.Getenv("DATABASE_URL")
	db, err := gorm.Open("postgres", gormConnDets)
	if err != nil {
		log.Fatal(err)
	}
	db.LogMode(true)
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
		pollInterval: time.Second * 1,
	}

	for _, job := range jobs {
		queue.Push(job)
	}
	go queue.Start()

	for i := 0; i < 10; i++ {
		time.Sleep(time.Second * 1)
		if len(runner.dispatched) >= 5 {
			queue.Halt()
			break
		}
	}

	for _, job := range jobs {
		if _, ok := runner.dispatched[job.ID]; !ok {
			t.Errorf("Job %s not dispatched", job.ID)
		}
	}
}

type fakeRunner struct {
	dispatched map[string]struct{}
	sync.Mutex
}

func (f *fakeRunner) Run(job Job) {
	log.Println("starting", job.ID)

	f.Lock()
	defer f.Unlock()
	if f.dispatched == nil {
		f.dispatched = make(map[string]struct{})
	}
	f.dispatched[job.ID] = struct{}{}
}

func (f fakeRunner) Stop(job Job) {
	log.Println("stopping", job.ID)
}
