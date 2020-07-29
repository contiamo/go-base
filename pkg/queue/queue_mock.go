package queue

import (
	"context"
)

// QueueMock implements queue with very simple counters for testing
type QueueMock struct {
	Queue chan Task

	EnqueueErr   error
	DequeueErr   error
	HeartbeatErr error
	FinishErr    error

	EnqueueCount   int
	DequeueCount   int
	HeartbeatCount int
	FinishCount    int
}

// Enqueue implements queue Manager for testing
func (q *QueueMock) Enqueue(ctx context.Context, task TaskEnqueueRequest) error {
	q.EnqueueCount = q.EnqueueCount + 1
	q.Queue <- Task{TaskBase: task.TaskBase}
	return q.EnqueueErr
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
		return &t, q.DequeueErr
	}

}

func (q *QueueMock) Heartbeat(ctx context.Context, taskID string, progress Progress) error {
	q.HeartbeatCount = q.HeartbeatCount + 1
	return q.HeartbeatErr
}

func (q *QueueMock) Finish(ctx context.Context, taskID string, progress Progress) error {
	q.FinishCount = q.FinishCount + 1
	return q.FinishErr
}
