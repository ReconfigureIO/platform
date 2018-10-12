package main

// defaultNMaxInFlight is the default per-q max in flight if the queue is
// not specified to the newJobQueueSemaphores constructor.
const defaultNMaxInFlight = 1

// Q names a queue.
type Q string

type jobQueueSemaphores struct {
	nMaxInFlight map[Q]int

	// fifo of work waiting to enter the running state.
	// queues map[Q]chan func()
	// a semaphore per channel.
	running map[Q]chan struct{}

	// enqueueCalls is used internally to process enqueue() calls.
	enqueueCalls chan enqueueCall
}

type enqueueCall struct {
	q   Q
	run func()
}

// newJobQueueSemaphores constructs a semaphore per job queue, which allows
// running at most n-jobs-in-flight. If a Q is not specified during
// construction, 1 job is Max in flight for that Q.
func newJobQueueSemaphores(nMaxInFlight map[Q]int) jobQueueSemaphores {
	s := jobQueueSemaphores{
		nMaxInFlight: nMaxInFlight,
		running:      map[Q]chan struct{}{},
		enqueueCalls: make(chan enqueueCall, 16),
	}

	go s.run()

	return s
}

func (s *jobQueueSemaphores) run() {
	for c := range s.enqueueCalls {
		s.enqueue(c.q, c.run)
	}
}

func (s *jobQueueSemaphores) enqueue(q Q, run func()) {
	runningChan, ok := s.running[q]
	if !ok {
		s.initQueue(q)
		runningChan = s.running[q]
	}

	go func() {
		runningChan <- struct{}{}
		defer func() { <-runningChan }()

		run()
	}()
}

func (s *jobQueueSemaphores) initQueue(q Q) {
	maxInFlight := s.qToNMaxInFlight(q)
	s.running[q] = make(chan struct{}, maxInFlight)
}

func (s *jobQueueSemaphores) qToNMaxInFlight(q Q) int {
	if n, ok := s.nMaxInFlight[q]; ok {
		return n
	}
	return defaultNMaxInFlight
}

// Enqueue calls run() when a slot becomes free in the given q.
// run() must block until the resources it uses are free.
// Enqueue does not normally block.
func (s *jobQueueSemaphores) Enqueue(q Q, run func()) {
	s.enqueueCalls <- enqueueCall{q, run}
}
