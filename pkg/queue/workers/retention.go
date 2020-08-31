package workers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/contiamo/go-base/pkg/queue"
	"github.com/contiamo/go-base/pkg/queue/postgres"
)

const (
	// MaintenanceTaskQueue task queue name used for all the periodic maintenance jobs.
	// These are internal queue internal tasks
	MaintenanceTaskQueue string = "queue-maintenance"

	// RetentionTask is finished task cleanup type
	RetentionTask queue.TaskType = "retention"
)

// NewRetentionHandler creates a task handler that will clean up old finished tasks
func NewRetentionHandler(db *sql.DB) TaskHandler {
	return NewSQLTaskHandler("RetentionHandler", db)
}

// AssertRetentionSchedule creates a new queue retention tasks for the supplied queue, finished tasks matching
// the supplied parameters will be deleted
func AssertRetentionSchedule(ctx context.Context, scheduler queue.Scheduler, queueName string, taskType queue.TaskType, status queue.TaskStatus, filter squirrel.Sqlizer, age time.Duration) error {
	spec := createRetentionSpec(queueName, taskType, status, filter, age)
	specBytes, err := json.Marshal(spec)
	if err != nil {
		return fmt.Errorf("can not build retention task spec: %w", err)
	}
	// randomly distribute the retention tasks throughout the hour
	when := rand.Intn(60)
	retentionSchedule := queue.TaskScheduleRequest{
		TaskBase: queue.TaskBase{
			Queue: MaintenanceTaskQueue,
			Type:  RetentionTask,
			Spec:  specBytes,
		},
		CronSchedule: fmt.Sprintf("%d * * * *", when), // every hour at minute "when"
	}

	return scheduler.AssertSchedule(ctx, retentionSchedule)
}

//createRetentionSpec builds the task retention job spec. It is split out to simplify test setup
func createRetentionSpec(queueName string, taskType queue.TaskType, status queue.TaskStatus, filter squirrel.Sqlizer, age time.Duration) SQLExecTaskSpec {
	spec := SQLExecTaskSpec{
		SQL: "",
	}

	// use separate WHERE statements to make the order deterministic
	deletionSQL := squirrel.Delete(postgres.TasksTable).
		Where(squirrel.Eq{"status": status}).
		Where(
			// note that using this comparision allows us to use the index on
			// finished_at, if yo use `age(now(), finished_at)`, this can not use the index
			fmt.Sprintf("finished_at <= now() - interval '%f minutes'", age.Minutes()),
		)

	if queueName != "" {
		deletionSQL = deletionSQL.Where(squirrel.Eq{"queue": queueName})
	}

	if taskType != "" {
		deletionSQL = deletionSQL.Where(squirrel.Eq{"type": taskType})
	}

	if filter != nil {
		deletionSQL = deletionSQL.Where(filter)
	}

	spec.SQL = squirrel.DebugSqlizer(deletionSQL)

	return spec
}
