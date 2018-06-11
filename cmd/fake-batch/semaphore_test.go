package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type nAllowed int32

func TestJobQueueSemaphores(t *testing.T) {
	// Check that JobQueueSemaphores works with various values of nAllowed.

	for _, v := range []nAllowed{1, 2, 10} {
		t.Run(
			fmt.Sprintf("nAllowed=%d", v),
			v.run,
		)
	}
}

func TestJobQueueSemaphoresFIFO(t *testing.T) {
	t.Parallel()

	for i := 0; i < 20; i++ {
		t.Run(fmt.Sprintf("paralleltest-%d", i), testJobQueueSemaphoresFIFOOne)
	}
}

func testJobQueueSemaphoresFIFOOne(t *testing.T) {
	t.Parallel()

	// Check that jobs run in the order they are given; assuming there is a long
	// enough gap between them being scheduled. NOTE: This test is racy and
	// could fail under adverse conditions. It has been carefully stress tested
	// with thousands of runs to make sure that it doesn't give false positives,
	// but you could encounter them if you are unlucky.

	jq := newJobQueueSemaphores(map[Q]int{
		"testqueuename": 1,
	})

	const n = 100
	var iPrime int
	var wg sync.WaitGroup

	for i := 0; i < n; i++ {
		i := i

		wg.Add(1)
		jq.Schedule("testqueuename", func() {
			defer wg.Done()

			if i != iPrime {
				t.Fatalf("i != iPrime: %d != %d", i, iPrime)
			}

			iPrime++

			// Job takes 1ms to run.
			time.Sleep(1000 * time.Microsecond)
		})

		// Schedule jobs at 0.5ms intervals, twice the rate they complete at, so
		// that they "pile up" in the queue.
		time.Sleep(500 * time.Microsecond)
	}

	wg.Wait()
}

func (nAllowed nAllowed) run(t *testing.T) {
	t.Parallel()

	// Check that we can schedule 10,000 jobs, with nAllowed max in flight, and
	// that this maximum is not exceeded. Also implicitly checks that all 10,000 run.
	const nSchedule = 10000

	jq := newJobQueueSemaphores(map[Q]int{
		"testqueuename": int(nAllowed),
	})

	var (
		nInFlight     int32
		highWaterMark int32
	)

	var wg sync.WaitGroup

	for i := 0; i < nSchedule; i++ {
		wg.Add(1)
		jq.Schedule("testqueuename", func() {
			defer wg.Done()

			v := atomic.AddInt32(&nInFlight, 1)
			if v > int32(nAllowed) {
				t.Errorf("max in flight exceeded: %v in flight", v)
			}

			vMax := atomic.LoadInt32(&highWaterMark)
			for v > vMax && !atomic.CompareAndSwapInt32(&highWaterMark, vMax, v) {
				vMax = atomic.LoadInt32(&highWaterMark)
			}

			// Required to allow other stuff to get scheduled.
			time.Sleep(1 * time.Microsecond)

			v = atomic.AddInt32(&nInFlight, -1)
			if v < 0 {
				t.Errorf("nInFlight negative: %d", v)
			}
		})
	}

	wg.Wait()

	if highWaterMark != int32(nAllowed) {
		t.Logf("high water mark did not reach nAllowed: %d != %d",
			highWaterMark, nAllowed)
	}
}
