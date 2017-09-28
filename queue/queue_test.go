package queue

import (
	"container/heap"
	"log"
	"sync"
	"testing"
	"time"
)

var jobs = []Job{
	Job{Id: "1", Weight: 4},
	Job{Id: "2", Weight: 1},
	Job{Id: "3", Weight: 5},
	Job{Id: "4", Weight: 3},
	Job{Id: "5", Weight: 2},
}

func TestPriorityQueue(t *testing.T) {
	var queue = priorityQueue(jobs)
	heap.Init(&queue)
	expected := []int{
		5, 4, 3, 2, 1,
	}
	for i := 0; i < len(expected); i++ {
		job := heap.Pop(&queue).(Job)
		if job.Weight != expected[i] {
			t.Errorf("Expected %d found %d", expected[i], job.Weight)
		}
	}
}

func TestQueue(t *testing.T) {
	runner := &fakeRunner{}
	var queue = New(runner, 2)
	for _, job := range jobs {
		queue.Push(job)
	}
	go queue.Start()
	for {
		time.Sleep(time.Second * 1)
		if len(runner.dispatched) >= 5 {
			queue.Halt()
			break
		}
	}

	for _, job := range jobs {
		if _, ok := runner.dispatched[job.Id]; !ok {
			t.Errorf("Job %s not dispatched", job.Id)
		}
	}
}

type fakeRunner struct {
	dispatched map[string]struct{}
	sync.Mutex
}

func (f *fakeRunner) Run(job Job) {
	log.Println("starting", job.Id)
	time.Sleep(time.Second * 2)

	f.Lock()
	defer f.Unlock()
	if f.dispatched == nil {
		f.dispatched = make(map[string]struct{})
	}
	f.dispatched[job.Id] = struct{}{}
}

func (f fakeRunner) Stop(job Job) {
	log.Println("stopping", job.Id)
}
