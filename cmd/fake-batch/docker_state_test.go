package main

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fortytw2/leaktest"
)

var numContainers = 100
var numFollowers = 100

func TestDockerState(t *testing.T) {
	defer leaktest.Check(t)()

	var wg sync.WaitGroup
	var (
		numSuccess, numFailed uint32
	)

	dockerState := NewDockerState()

	var runLogFollowers = func(id string) {
		for j := 0; j < numFollowers; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				cs, present := dockerState.Get(id)
				var success bool
				if present {
					success = cs.BecomeLogFollower()
					if success {
						defer cs.UnfollowLogs()
					}
				}

				if !present || !success {
					atomic.AddUint32(&numFailed, 1)
					return
				}

				atomic.AddUint32(&numSuccess, 1)
			}()
		}
	}

	var createContainer = func(id string) {
		cs := dockerState.Create(id)

		wg.Add(1)
		go func() {
			defer wg.Done()

			runLogFollowers(id)
		}()

		r := 1 + rand.Intn(100)
		time.Sleep(time.Duration(r) * time.Microsecond)

		cs.SetStarted()

		r = 1 + rand.Intn(20)
		time.Sleep(time.Duration(r) * time.Microsecond)
		dockerState.Delete(id)
	}

	for i := 0; i < numContainers; i++ {
		id := fmt.Sprint(i)

		wg.Add(1)
		go func() {
			defer wg.Done()

			createContainer(id)
		}()
	}

	wg.Wait()

	if len(dockerState.idToContainerState) != 0 {
		t.Errorf("Not all containerStates have been removed, expected 0, got: %v \n", len(dockerState.idToContainerState))
	}

	t.Logf("success = %d failed = %d", numSuccess, numFailed)
}
