package queue

import (
	"context"
	"errors"

	cdb "github.com/contiamo/go-base/v3/pkg/db"
	"github.com/prometheus/client_golang/prometheus"
)

// ErrNoteSchedule indicates the current scheduled task is has not been created yet. This should
// be returned from the EnsureSchedule method
var ErrNotScheduled = errors.New("Task not currently scheduled")

// Scheduler defines how one schedules a task
type Scheduler interface {
	// Schedule creates a cron schedule according to which the worker will enqueue
	// the given task.
	// `task.CronSchedule` cannot be blank.
	Schedule(ctx context.Context, builder cdb.SQLBuilder, task TaskScheduleRequest) error

	// EnsureSchedule checks if a task for with the given queue, type, and references
	// currently exists. An error ErrNotScheduled is returned if a current task cannot be found.
	// implementation and validation errors may also be returned and should be checked for.
	EnsureSchedule(ctx context.Context, builder cdb.SQLBuilder, task TaskScheduleRequest) (err error)
	// AssertSchedule makes sure a schedule with the given parameters exists, if it does not
	// this function will create one.
	AssertSchedule(ctx context.Context, task TaskScheduleRequest) (err error)
}

type schedulerWithMetrics struct {
	s Scheduler
}

// SchedulerWithMetrics returns q wrapped with the standard metrics implementation
func SchedulerWithMetrics(s Scheduler) Scheduler {
	return &schedulerWithMetrics{s}
}

func (s *schedulerWithMetrics) Schedule(ctx context.Context, builder cdb.SQLBuilder, task TaskScheduleRequest) (err error) {
	labels := prometheus.Labels{"queue": task.Queue, "type": task.Type.String()}
	SchedulerMetrics.ScheduleCounter.With(labels).Inc()
	defer func() {
		if err != nil {
			SchedulerMetrics.ErrorCounter.With(labels).Inc()
		}
	}()
	return s.s.Schedule(ctx, builder, task)
}

func (s *schedulerWithMetrics) EnsureSchedule(ctx context.Context, builder cdb.SQLBuilder, task TaskScheduleRequest) error {
	return s.s.EnsureSchedule(ctx, builder, task)
}
func (s *schedulerWithMetrics) AssertSchedule(ctx context.Context, task TaskScheduleRequest) error {
	return s.s.AssertSchedule(ctx, task)
}
