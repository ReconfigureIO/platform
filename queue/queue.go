package queue

import (
	"container/heap"
	"time"
)

// Queue is a platform queue implementation.
type Queue interface {
	// Push pushes an entry to the top of the queue.
	Push(Entry)
	// Pop removes the last entry in the queue and returns it.
	Pop() Entry
	// Concurrent sets the number of jobs that can run concurrently.
	Concurrent(n int)
}

// Entry is queue entry.
type Entry struct {
	Id      string
	Meta    map[string]interface{}
	Timeout time.Duration
	Weight  int
	Execute func()
}

// queueImpl is the implementation of Queue using
// container/heap as underlying priority queue.
type queueImpl struct {
	queue      priorityQueue
	concurrent int
}

var _ heap.Interface = priorityQueue{}

type priorityQueue []Entry

func (q priorityQueue) Len() int           { return len(q) }
func (q priorityQueue) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }
func (q priorityQueue) Less(i, j int) bool { return q[i].Weight < q[j].Weight }
func (q priorityQueue) Push(x interface{}) { q.push(x) }
func (q priorityQueue) Pop() interface{}   { return q.pop() }
func (q *priorityQueue) push(x interface{}) {
	entry, ok := x.(Entry)
	if !ok {
		return
	}
	*q = append([]Entry{entry}, (*q)...)
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
