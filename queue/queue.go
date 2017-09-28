package queue

import (
	"container/heap"
	"sync"
	"time"
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
}

// JobRunner manage jobs in the queue.
type JobRunner interface {
	Run(Job)
	Stop(Job)
}

// Job is queue entry.
type Job struct {
	Id     string
	Meta   map[string]interface{}
	Weight int
}

var _ Queue = &queueImpl{}

// queueImpl is the implementation of Queue using
// container/heap as underlying priority queue.
type queueImpl struct {
	queue      priorityQueue
	dispatched map[*Job]struct{}

	runner     JobRunner
	halt       chan struct{}
	concurrent int

	sync.RWMutex
}

// New creates a new Queue.
func New(runner JobRunner, concurrent int) Queue {
	return &queueImpl{
		queue:      priorityQueue{},
		runner:     runner,
		concurrent: concurrent,
	}
}

func (q *queueImpl) Push(j Job) {
	q.Lock()
	defer q.Unlock()

	q.queue.Push(j)
}

func (q *queueImpl) Start() {
	q.halt = make(chan struct{})

	stop := false
	go func() {
		<-q.halt
		stop = true
	}()

	for !stop {
		// TODO 5 second suffices for now,
		// it may change in the future.
		time.Sleep(time.Second * 5)

		q.RLock()
		toRun := q.concurrent - len(q.dispatched)
		q.RUnlock()
		for i := 0; i < toRun; i++ {
			go q.dispatch()
		}

	}
}

func (q *queueImpl) Halt() {
	close(q.halt)
}

func (q *queueImpl) dispatch() {
	// pop from priority queue and add
	// to dispatched jobs.
	q.Lock()
	job := q.queue.Pop().(Job)
	q.dispatched[&job] = struct{}{}
	q.Unlock()

	// run job
	q.runner.Run(job)

	// delete from dispatched jobs
	q.Lock()
	delete(q.dispatched, &job)
	q.Unlock()
}

var _ heap.Interface = &priorityQueue{}

type priorityQueue []Job

func (q priorityQueue) Len() int            { return len(q) }
func (q priorityQueue) Swap(i, j int)       { q[i], q[j] = q[j], q[i] }
func (q priorityQueue) Less(i, j int) bool  { return q[j].Weight < q[i].Weight }
func (q *priorityQueue) Push(x interface{}) { q.push(x) }
func (q *priorityQueue) Pop() interface{}   { return q.pop() }
func (q *priorityQueue) push(x interface{}) {
	entry, ok := x.(Job)
	if !ok {
		return
	}
	*q = append([]Job{entry}, (*q)...)
}
func (q *priorityQueue) pop() interface{} {
	l := len(*q)
	if l == 0 {
		return nil
	}
	entry := (*q)[l-1]
	*q = (*q)[:l-1]
	return entry
}
