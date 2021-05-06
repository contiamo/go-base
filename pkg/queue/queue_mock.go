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
	FailErr      error

	EnqueueCount   int
	DequeueCount   int
	HeartbeatCount int
	FinishCount    int
	FailCount      int
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

// Heartbeat implements queue Manager for testing
func (q *QueueMock) Heartbeat(ctx context.Context, taskID string, progress Progress) error {
	q.HeartbeatCount = q.HeartbeatCount + 1
	return q.HeartbeatErr
}

// Finish implements queue Manager for testing
func (q *QueueMock) Finish(ctx context.Context, taskID string, progress Progress) error {
	q.FinishCount = q.FinishCount + 1
	return q.FinishErr
}

// Fail implements queue Manager for testing
func (q *QueueMock) Fail(ctx context.Context, taskID string, progress Progress) error {
	q.FailCount = q.FailCount + 1
	return q.FailErr
}
