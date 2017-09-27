package queue

import (
	"container/heap"
)

// Queue is a job queue.
type Queue interface {
	// Push adds an entry to the queue.
	Push(Job)
	// Start starts and monitors the queue.
	// All popped jobs are passed to the job runner to run.
	// This blocks forever.
	Start()
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

// queueImpl is the implementation of Queue using
// container/heap as underlying priority queue.
type queueImpl struct {
	queue      priorityQueue
	runner     JobRunner
	concurrent int
}

func New(runner JobRunner, concurrent int) Queue {
	return &queueImpl{
		queue:      priorityQueue{},
		runner:     nil,
		concurrent: concurrent,
	}
}

func (q *queueImpl) Push(j Job) {
	q.queue.Push(j)
}
func (q *queueImpl) Start() {

}

var _ heap.Interface = priorityQueue{}

type priorityQueue []Job

func (q priorityQueue) Len() int           { return len(q) }
func (q priorityQueue) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }
func (q priorityQueue) Less(i, j int) bool { return q[i].Weight < q[j].Weight }
func (q priorityQueue) Push(x interface{}) { q.push(x) }
func (q priorityQueue) Pop() interface{}   { return q.pop() }
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
