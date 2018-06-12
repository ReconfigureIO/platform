package main

// Q names a queue.
type Q string

type jobQueueSemaphores struct {
	nAllowedInFlight map[Q]int
	queues           map[Q]chan func()
	running          map[Q]chan struct{}
}

// newJobQueueSemaphores constructs a semaphore per job queue, which allows
// running at most n-jobs-in-flight. If a Q is not specified during
// construction, 1 job is allowed in flight for that Q.
func newJobQueueSemaphores(nAllowedInFlight map[Q]int) jobQueueSemaphores {
	s := jobQueueSemaphores{
		nAllowedInFlight: nAllowedInFlight,
	}

	return s
}

// Schedule calls run() when a slot becomes free in the given q.
// run() must block until the resources it uses are free.
func (s *jobQueueSemaphores) Schedule(q Q, run func()) {

}
