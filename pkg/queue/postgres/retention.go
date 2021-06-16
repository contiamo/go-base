package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/Masterminds/squirrel"
	cdb "github.com/contiamo/go-base/v4/pkg/db"
	"github.com/contiamo/go-base/v4/pkg/queue"
	"github.com/contiamo/go-base/v4/pkg/queue/handlers"
	cvalidation "github.com/contiamo/go-base/v4/pkg/validation"
	"github.com/opentracing/opentracing-go"
	uuid "github.com/satori/go.uuid"
)

const (
	// MaintenanceTaskQueue task queue name used for all the periodic maintenance jobs.
	// These are internal queue internal tasks
	MaintenanceTaskQueue string = "queue-maintenance"

	// RetentionTask is finished task cleanup type
	RetentionTask queue.TaskType = "retention"
)

// RetentionTaskSpec defines a SQL task to remove completed tasks that match given criteria.
type RetentionTaskSpec struct {
	handlers.SQLExecTaskSpec
	QueueName string           `json:"queueName"`
	TaskType  queue.TaskType   `json:"taskType"`
	Status    queue.TaskStatus `json:"status"`
	Age       time.Duration    `json:"age"`
}

// NewRetentionHandler creates a task handler that will clean up old finished tasks
func NewRetentionHandler(db *sql.DB) queue.TaskHandler {
	return handlers.NewSQLTaskHandler("RetentionHandler", db)
}

// AssertRetentionSchedule creates a new queue retention tasks for the supplied queue, finished tasks matching
// the supplied parameters will be deleted
func AssertRetentionSchedule(ctx context.Context, db *sql.DB, queueName string, taskType queue.TaskType, status queue.TaskStatus, age time.Duration) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "AssertRetentionSchedule")
	span.SetTag("pkg.name", "postgres")

	spec := createRetentionSpec(queueName, taskType, status, age)

	return AssertRetentionScheduleWithSpec(ctx, db, spec)
}

// AssertRetentionScheduleWithSpec ensures that a queue retention tasks exists with the provided task spec.
// The task is scheduled for a random minute throughout the hour. Use AssertRetentionSchedule to create simple
// retention tasks based on queue, type, and age. Callers should only use AssertRetentionScheduleWithSpec when
// the retention policy by age is not accurate enough for their needs.
//
// An upsert pattern is used to ensure that this retention task is scheduled exactly once.
func AssertRetentionScheduleWithSpec(ctx context.Context, db *sql.DB, spec RetentionTaskSpec) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "AssertRetentionScheduleWithSpec")
	span.SetTag("pkg.name", "postgres")

	specBytes, err := json.Marshal(spec)
	if err != nil {
		return fmt.Errorf("can not build retention task spec: %w", err)
	}

	// randomly distribute the retention tasks throughout the hour
	// nolint: gosec // this random value is not involved in any security related logic
	when := rand.Intn(60)
	retentionSchedule := queue.TaskScheduleRequest{
		TaskBase: queue.TaskBase{
			Queue: MaintenanceTaskQueue,
			Type:  RetentionTask,
			Spec:  specBytes,
		},
		CronSchedule: fmt.Sprintf("%d * * * *", when), // every hour at minute "when"
	}

	err = cvalidation.CronTab(retentionSchedule.CronSchedule)
	if err != nil {
		return err
	}

	builder := squirrel.StatementBuilder.
		PlaceholderFormat(squirrel.Dollar).
		RunWith(cdb.WrapWithTracing(db))

	scheduleID := uuid.NewV4().String()
	now := time.Now()
	q := builder.Insert("schedules").
		Columns(
			"schedule_id",
			"task_queue",
			"task_type",
			"task_spec",
			"cron_schedule",
			"next_execution_time",
		).
		Values(
			scheduleID,
			retentionSchedule.Queue,
			retentionSchedule.Type,
			retentionSchedule.Spec,
			retentionSchedule.CronSchedule,
			now, // the schedule will enqueue the task immediately
		).Suffix(`
			ON CONFLICT (task_queue,task_type,(task_spec->>'queueName'),(task_spec->>'taskType'),(task_spec->>'status')) WHERE task_type='retention'
			DO UPDATE SET
				updated_at=?,
				next_execution_time=?,
				task_spec=EXCLUDED.task_spec,
				cron_schedule=EXCLUDED.cron_schedule
		`, now, now)

	res, err := q.ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("can not upsert retention schedule: %w", err)
	}

	updated, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("can not determine the number of rows affected: %w", err)
	}

	span.SetTag("affected", updated)

	return nil
}

//createRetentionSpec builds the task retention job spec. It is split out to simplify test setup
func createRetentionSpec(queueName string, taskType queue.TaskType, status queue.TaskStatus, age time.Duration) RetentionTaskSpec {
	spec := RetentionTaskSpec{
		QueueName: queueName,
		TaskType:  taskType,
		Status:    status,
		Age:       age,
	}

	// use separate WHERE statements to make the order deterministic
	deletionSQL := squirrel.Delete(TasksTable).
		Where(squirrel.Eq{"status": status}).
		Where(
			// note that using this comparison allows us to use the index on
			// finished_at, if yo use `age(now(), finished_at)`, this can not use the index
			fmt.Sprintf("finished_at <= now() - interval '%f minutes'", age.Minutes()),
		)

	if queueName != "" {
		deletionSQL = deletionSQL.Where(squirrel.Eq{"queue": queueName})
	}

	if taskType != "" {
		deletionSQL = deletionSQL.Where(squirrel.Eq{"type": taskType})
	}

	spec.SQL = squirrel.DebugSqlizer(deletionSQL)

	return spec
}
