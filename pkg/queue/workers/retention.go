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
	"github.com/contiamo/go-base/pkg/tracing"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	// MaintenanceTaskQueue task queue name used for all the periodic maintenance jobs.
	// These are internal queue internal tasks
	MaintenanceTaskQueue string = "queue-maintenance"

	// RetentionTask is finished task cleanup type
	RetentionTask queue.TaskType = "retention"
)

var (
	errSerializingHearbeat = errors.New("failed to serialize progress payload while sending heartbeat")
)

type retentionProgress struct {
	// Duration of the HTTP request in milliseconds
	Duration *int64 `json:"duration,omitempty"`
	// RowsAffected
	RowsAffected *int64 `json:"rowsAffected,omitempty"`
	// ErrorMessage contains an error message string if it occurs during the update process
	ErrorMessage *string `json:"errorMessage,omitempty"`
}

type retentionSpec struct {
	// QueueName determine which queue the retention policy applies to
	QueueName string `json:"queueName"`
	// TaskType determines which task type the retention policy applies to
	TaskType queue.TaskType `json:"taskType"`
	// Status is the required status the task must have to be deleted
	Status queue.TaskStatus `json:"status"`
	// Age is the time since finish when the task will be considered for deletion
	Age time.Duration `json:"age"`
	// SQL is the actual sql that will be run
	SQL string `json:"sql"`
}

// NewRetentionHandler creates a task handler that will clean up old finished tasks
func NewRetentionHandler(db *sql.DB) TaskHandler {
	return &retentionHandler{
		Tracer: tracing.NewTracer("handlers", "RetentionHandler"),
		db:     db,
	}
}

type retentionHandler struct {
	tracing.Tracer
	db *sql.DB
}

func (h *retentionHandler) Process(ctx context.Context, task queue.Task, heartbeats chan<- queue.Progress) (err error) {
	span, ctx := h.StartSpan(ctx, "Process")
	defer func() {
		h.FinishSpan(span, err)
	}()
	span.SetTag("task.id", task.ID)
	span.SetTag("task.queue", task.Queue)
	span.SetTag("task.type", task.Type)
	span.SetTag("task.spec", string(task.Spec))

	log := logrus.WithContext(ctx).WithField("type", task.Type)
	log.Debug("starting retention task")

	var progress retentionProgress
	defer func() {
		// we check for errSerializingHearbeat so we don't cause
		// a recursion call
		if err == nil || err == errSerializingHearbeat {
			return
		}
		message := err.Error()
		progress.ErrorMessage = &message
		_ = sendRetentionProgress(progress, heartbeats)
	}()

	spec := retentionSpec{}
	err = json.Unmarshal(task.Spec, &spec)
	if err != nil {
		return fmt.Errorf("can not parse the retention task spec %w", err)
	}

	span.SetTag("spec.queue", spec.QueueName)
	span.SetTag("spec.taskType", spec.TaskType)
	span.SetTag("spec.status", spec.Status)
	span.SetTag("spec.age", spec.Age.String())

	// initial heartbeat
	err = sendRetentionProgress(progress, heartbeats)
	if err != nil {
		return err
	}

	now := time.Now()
	result, err := h.db.ExecContext(ctx, spec.SQL)
	duration := time.Since(now).Milliseconds()
	progress.Duration = &duration

	if err != nil {
		return fmt.Errorf("retention execution failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("can not get rows affected: %w", err)
	}

	progress.RowsAffected = &rowsAffected
	err = sendRetentionProgress(progress, heartbeats)
	if err != nil {
		return err
	}

	log.WithField("rowCount", rowsAffected).Debug("retention finished")
	return nil
}

func sendRetentionProgress(progress retentionProgress, heartbeats chan<- queue.Progress) (err error) {
	logrus.
		WithField("method", "sendRetentionProgress").
		Debugf("%+v", progress)

	bytes, err := json.Marshal(progress)
	if err != nil {
		logrus.Error(err)
		return errSerializingHearbeat
	}

	heartbeats <- bytes
	return nil
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
func createRetentionSpec(queueName string, taskType queue.TaskType, status queue.TaskStatus, filter squirrel.Sqlizer, age time.Duration) retentionSpec {
	spec := retentionSpec{
		QueueName: queueName,
		TaskType:  taskType,
		Status:    status,
		Age:       age,
		SQL:       "",
	}

	// use sepqrate WHERE statements to make the order deterministic
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
