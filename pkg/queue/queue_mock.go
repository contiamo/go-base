package queue

import (
	"context"
)

// QueueMock implements queue with very simple counters for testing
type QueueMock struct {
	Queue          chan *Task
	DequeueErr     error
	HeartbeatErr   error
	FinishErr      error
	EnqueueCount   int
	DequeueCount   int
	HeartbeatCount int
	FinishCount    int
}

// Enqueue implements queue Manager for testing
func (q *QueueMock) Enqueue(ctx context.Context, task TaskEnqueueRequest) error {
	q.EnqueueCount = q.EnqueueCount + 1
	q.Queue <- &Task{TaskBase: task.TaskBase}
	return nil
}

// Dequeue implements queue Manager for testing
func (q *QueueMock) Dequeue(ctx context.Context, queue ...string) (*Task, error) {
	q.DequeueCount = q.DequeueCount + 1
	if q.DequeueErr != nil {
		return nil, q.DequeueErr
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case t := <-q.Queue:
		return t, nil
	}

}
