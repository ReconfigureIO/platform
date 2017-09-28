package queue

import (
	"container/heap"
	"testing"
)

func TestPriorityQueue(t *testing.T) {
	var queue = &priorityQueue{
		Job{Weight: 3},
		Job{Weight: 1},
		Job{Weight: 5},
		Job{Weight: 3},
		Job{Weight: 2},
	}
	heap.Init(queue)
	expected := []int{
		5, 3, 3, 2, 1,
	}
	for i := 0; i < len(expected); i++ {
		job := heap.Pop(queue).(Job)
		if job.Weight != expected[i] {
			t.Errorf("Expected %d found %d", expected[i], job.Weight)
		}
	}
}
