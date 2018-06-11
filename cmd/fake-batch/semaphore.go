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

	// scheduleCalls is used internally to process scheduling events
	scheduleCalls chan scheduleCall
}

type scheduleCall struct {
	q   Q
	run func()
}

// newJobQueueSemaphores constructs a semaphore per job queue, which allows
// running at most n-jobs-in-flight. If a Q is not specified during
// construction, 1 job is Max in flight for that Q.
func newJobQueueSemaphores(nMaxInFlight map[Q]int) jobQueueSemaphores {
	s := jobQueueSemaphores{
		nMaxInFlight:  nMaxInFlight,
		running:       map[Q]chan struct{}{},
		scheduleCalls: make(chan scheduleCall, 16),
	}

	go s.run()

	return s
}

func (s *jobQueueSemaphores) run() {
	for c := range s.scheduleCalls {
		s.schedule(c.q, c.run)
	}
}

func (s *jobQueueSemaphores) schedule(q Q, run func()) {
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

// Schedule calls run() when a slot becomes free in the given q.
// run() must block until the resources it uses are free.
func (s *jobQueueSemaphores) Schedule(q Q, run func()) {
	s.scheduleCalls <- scheduleCall{q, run}
}
