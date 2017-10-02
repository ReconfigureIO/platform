package queue

import (
	"container/heap"
	"sync"
	"time"
)

var _ Queue = &memoryQueue{}

// memoryQueue is the implementation of Queue using
// container/heap as underlying priority queue.
type memoryQueue struct {
	queue      *priorityQueue
	dispatched map[*Job]struct{}
	jobType    string

	runner     JobRunner
	halt       chan struct{}
	concurrent int

	sync.RWMutex
}

// NewWithMemoryStore creates a new Queue using runtime memory as storage
// for queue state.
func NewWithMemoryStore(runner JobRunner, concurrent int, jobType string) Queue {
	return &memoryQueue{
		queue:      &priorityQueue{},
		runner:     runner,
		dispatched: make(map[*Job]struct{}),
		jobType:    jobType,
		concurrent: concurrent,
	}
}

func (q *memoryQueue) Push(j Job) {
	q.Lock()
	defer q.Unlock()

	heap.Push(q.queue, j)
}

func (q *memoryQueue) Start() {
	q.halt = make(chan struct{})

	stop := false
	go func() {
		<-q.halt
		stop = true
	}()

	for !stop {
		// TODO 1 second suffices for now,
		// it may change in the future.
		time.Sleep(time.Second * 1)

		q.RLock()
		toRun := q.concurrent - len(q.dispatched)
		length := q.queue.Len()
		q.RUnlock()
		for i := 0; i < toRun && i < length; i++ {
			// pop from priority queue and add
			// to dispatched jobs.
			q.Lock()
			job := heap.Pop(q.queue).(Job)
			q.dispatched[&job] = struct{}{}
			q.Unlock()

			go q.dispatch(&job)
		}

	}
}

func (q *memoryQueue) Halt() {
	close(q.halt)
}

func (q *memoryQueue) dispatch(job *Job) {
	// run job
	q.runner.Run(*job)

	// delete from dispatched jobs
	q.Lock()
	delete(q.dispatched, job)
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
