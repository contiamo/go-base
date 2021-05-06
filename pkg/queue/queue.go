package queue

import (
	"context"
	"errors"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	// ErrTaskCancelled is used to notify the worker from the Heartbeat that the task
	// was canceled by the user and it needs to stop working on it.
	ErrTaskCancelled = errors.New("task has been canceled")
	// ErrTaskFinished is used to notify the worker from the Heartbeat that the task has
	// already finished and can't be worked on
	ErrTaskFinished = errors.New("task has finished")
	// ErrTaskNotRunning is used to notify the worker from the Heartbeat that the task has
	// not started yet and can't be worked on
	ErrTaskNotRunning = errors.New("task has not started yet")
	// ErrTaskNotFound is used to notify the worker from the Heartbeat that the task does not
	// exist
	ErrTaskNotFound = errors.New("task not found")
	// ErrTaskFailed indicates that a task has failed and should not be restarted
	ErrTaskFailed = errors.New("task failed")
	// ErrTaskQueueNotSpecified indicates that the given task cannot be enqueued because
	// the queue name is empty
	ErrTaskQueueNotSpecified = errors.New("task queue name cannot be blank")
	// ErrTaskTypeNotSpecified indicates that the given task cannot be enqueued because
	// the task type is empty
	ErrTaskTypeNotSpecified = errors.New("task type cannot be blank")
)

// Queuer is a write-only interface for the queue
type Queuer interface {
	// Enqueue adds a task to the provided queue within the Task object
	Enqueue(ctx context.Context, task TaskEnqueueRequest) error
}

// Dequeuer is a read only interface for the queue
type Dequeuer interface {
	// Dequeue removes a task for processing from the provided queue
	// It's expected to block until work is available
	//
	// For zero-length queue parameter, dequeue should return a task from a random queue;
	// otherwise dequeue should return a task from the supplied list of queues.
	Dequeue(ctx context.Context, queue ...string) (*Task, error)

	// Heartbeat updates the task to ensure that it's still being processed by a worker.
	//
	// Once a task has begun processing, 'Heartbeat' should be called at known
	// intervals to update the 'Progress' and 'LastHeartbeat' fields.
	// This provides a way to determine if the worker is still processing.
	Heartbeat(ctx context.Context, taskID string, progress Progress) error

	// Finish marks the task as completed
	// It's recommended to update the progress to a final state to ensure there is
	// no ambiguity in whether or not the task is complete
	Finish(ctx context.Context, taskID string, progress Progress) error

	// Finish marks the task as failed
	// It's recommended to update the progress to a final state.
	Fail(ctx context.Context, taskID string, progress Progress) error
}

type queuerWithMetrics struct {
	q Queuer
}

// QueuerWithMetrics returns q wrapped with the standard metrics implementation
func QueuerWithMetrics(q Queuer) Queuer {
	return &queuerWithMetrics{q}
}

func (q *queuerWithMetrics) Enqueue(ctx context.Context, task TaskEnqueueRequest) error {
	TaskQueueMetrics.TaskCounter.With(prometheus.Labels{"queue": task.Queue, "type": task.Type.String()}).Inc()
	timer := prometheus.NewTimer(TaskQueueMetrics.EnqueueDuration.With(prometheus.Labels{"queue": task.Queue}))
	defer timer.ObserveDuration()

	return q.q.Enqueue(ctx, task)
}

type dequeuerWithMetrics struct {
	q Dequeuer
}

// DequeuerWithMetrics returns q wrapped with the standard metrics implementation
func DequeuerWithMetrics(q Dequeuer) Dequeuer {
	return &dequeuerWithMetrics{q}
}

func (q *dequeuerWithMetrics) Dequeue(ctx context.Context, queue ...string) (*Task, error) {
	return q.q.Dequeue(ctx, queue...)
}

func (q *dequeuerWithMetrics) Heartbeat(ctx context.Context, taskID string, progress Progress) error {
	return q.q.Heartbeat(ctx, taskID, progress)
}

func (q *dequeuerWithMetrics) Finish(ctx context.Context, taskID string, progress Progress) error {
	return q.q.Finish(ctx, taskID, progress)
}

func (q *dequeuerWithMetrics) Fail(ctx context.Context, taskID string, progress Progress) error {
	return q.q.Fail(ctx, taskID, progress)
}
