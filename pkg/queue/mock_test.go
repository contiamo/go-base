package queue

import (
	"context"

	cdb "github.com/contiamo/go-base/pkg/db"
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

// Heartbeat implements queue Manager for testing
func (q *QueueMock) Heartbeat(ctx context.Context, taskID string, progress Progress) error {
	q.HeartbeatCount = q.HeartbeatCount + 1
	return q.HeartbeatErr
}

// Finish implements queue Manager for testing
func (q *QueueMock) Finish(ctx context.Context, taskID string, progress Progress, isFailed bool) error {
	q.FinishCount = q.FinishCount + 1
	return q.FinishErr
}

// SchedulerMock implement scheduler with very simple counters for testing
type SchedulerMock struct {
	ScheduleErr   error
	ScheduleCount int

	EnsureError error
	EnsureCount int
}

func (s *SchedulerMock) Schedule(ctx context.Context, builder cdb.SQLBuilder, task TaskScheduleRequest) (err error) {
	s.ScheduleCount = s.ScheduleCount + 1
	return s.ScheduleErr
}

func (s *SchedulerMock) EnsureSchedule(ctx context.Context, builder cdb.SQLBuilder, task TaskScheduleRequest) error {
	s.EnsureCount = s.EnsureCount + 1
	return s.EnsureError
}
