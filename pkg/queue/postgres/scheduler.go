package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/Masterminds/squirrel"
	cdb "github.com/contiamo/go-base/pkg/db"
	"github.com/contiamo/go-base/pkg/tracing"
	cvalidation "github.com/contiamo/go-base/pkg/validation"
	"github.com/contiamo/go-base/pkg/queue"
	uuid "github.com/satori/go.uuid"
)

// NewScheduler creates a new postgres task scheduler
func NewScheduler(db *sql.DB) queue.Scheduler {
	return &scheduler{
		Tracer: tracing.NewTracer("queue", "PostgresScheduler"),
	}
}

// NewScheduler creates a new postgres task scheduler with metrics enabled
func NewSchedulerWithMetrics(db *sql.DB) queue.Scheduler {
	return queue.SchedulerWithMetrics(NewScheduler(db))
}

// scheduler is a postgres backed implementation of the task scheduler
type scheduler struct {
	tracing.Tracer
}

func (q *scheduler) Schedule(ctx context.Context, builder cdb.SQLBuilder, task queue.TaskScheduleRequest) (err error) {
	span, ctx := q.StartSpan(ctx, "Schedule")
	defer func() {
		q.FinishSpan(span, err)
	}()

	if task.Queue == "" {
		return queue.ErrTaskQueueNotSpecified
	}

	if task.Type == "" {
		return queue.ErrTaskTypeNotSpecified
	}

	if task.Spec == nil {
		task.Spec = emptyJSON
	}

	err = cvalidation.CronTab(task.CronSchedule)
	if err != nil {
		return err
	}

	scheduleID := uuid.NewV4().String()

	span.SetTag("schedule.id", scheduleID)
	span.SetTag("schedule.cron", task.CronSchedule)
	span.SetTag("schedule.references", task.References)
	span.SetTag("task.queue", task.Queue)
	span.SetTag("task.type", task.Type)
	span.SetTag("task.spec", string(task.Spec))

	refColumns, refValues := task.References.GetNamesAndValues()

	_, err = builder.
		Insert("schedules").
		Columns(
			append(
				refColumns,
				"schedule_id",
				"task_queue",
				"task_type",
				"task_spec",
				"cron_schedule",
				"next_execution_time",
			)...,
		).
		Values(
			append(
				refValues,
				scheduleID,
				task.Queue,
				task.Type,
				task.Spec,
				task.CronSchedule,
				time.Now(), // the schedule will enqueue the task immediately
			)...,
		).
		ExecContext(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (q *scheduler) EnsureSchedule(ctx context.Context, builder cdb.SQLBuilder, task queue.TaskScheduleRequest) (err error) {
	span, ctx := q.StartSpan(ctx, "EnsureSchedule")
	defer func() {
		q.FinishSpan(span, err)
	}()

	if task.Queue == "" {
		return queue.ErrTaskQueueNotSpecified
	}

	if task.Type == "" {
		return queue.ErrTaskTypeNotSpecified
	}

	span.SetTag("schedule.references", task.References)
	span.SetTag("task.queue", task.Queue)
	span.SetTag("task.type", task.Type)

	refColumns, refValues := task.References.GetNamesAndValues()

	query := builder.
		Select("1").
		From("schedules").
		Limit(1).
		Where(squirrel.Eq{"task_queue": task.Queue}).
		Where(squirrel.Eq{"task_type": task.Type}).
		Where(squirrel.Eq{"task_spec": []byte(task.Spec)})

	for idx, col := range refColumns {
		query = query.Where(squirrel.Eq{col: refValues[idx]})
	}

	var exists int
	err = query.ScanContext(ctx, &exists)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	// will only non-zero if err is nil
	if exists != 0 {
		return nil
	}

	// either value was 0 or err == sql.ErrorNoRows
	return queue.ErrNotScheduled
}
