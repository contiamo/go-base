package workers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/contiamo/go-base/pkg/queue"
	"github.com/contiamo/go-base/pkg/tracing"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	errSerializingHearbeat = errors.New("failed to serialize progress payload while sending heartbeat")
)

// SQLTaskProgress contains the generic progress information for a sql task
type SQLTaskProgress struct {
	// Duration of the HTTP request in milliseconds
	Duration *int64 `json:"duration,omitempty"`
	// RowsAffected
	RowsAffected *int64 `json:"rowsAffected,omitempty"`
	// ErrorMessage contains an error message string if it occurs during the update process
	ErrorMessage *string `json:"errorMessage,omitempty"`
}

// SQLExecTaskSpec  defines a task that simply executes a single SQL statement. This can
// be used for simple CRON cleanup tasks, for example.
type SQLExecTaskSpec struct {
	// SQL is the actual sql that will be run
	SQL string `json:"sql"`
}

type sqlTaskHandler struct {
	tracing.Tracer
	db *sql.DB
}

// NewSQLTaskHandler creates a sqlTaskHandler handler instance with the given tracing name
func NewSQLTaskHandler(name string, db *sql.DB) TaskHandler {
	return &sqlTaskHandler{
		Tracer: tracing.NewTracer("handlers", name),
		db:     db,
	}
}

func (h *sqlTaskHandler) Process(ctx context.Context, task queue.Task, heartbeats chan<- queue.Progress) (err error) {
	span, ctx := h.StartSpan(ctx, "Process")
	defer func() {
		h.FinishSpan(span, err)
		close(heartbeats)
	}()
	span.SetTag("task.id", task.ID)
	span.SetTag("task.queue", task.Queue)
	span.SetTag("task.type", task.Type)

	log := logrus.WithContext(ctx).
		WithField("id", task.ID).
		WithField("type", task.Type).
		WithField("queue", task.Queue)
	log.Debug("starting sql task")

	var progress SQLTaskProgress
	defer func() {
		// we check for errSerializingHearbeat so we don't cause
		// a recursion call
		if err == nil || err == errSerializingHearbeat {
			return
		}
		message := err.Error()
		progress.ErrorMessage = &message
		_ = sendSQLTaskProgress(progress, heartbeats)
	}()

	// initial heartbeat
	err = sendSQLTaskProgress(progress, heartbeats)
	if err != nil {
		return err
	}

	spec := SQLExecTaskSpec{}
	err = json.Unmarshal(task.Spec, &spec)
	if err != nil {
		return fmt.Errorf("can not parse the sql task spec %w", err)
	}

	span.SetTag("task.spec.sql", spec.SQL)
	log.Debug(spec.SQL)

	now := time.Now()
	result, err := h.db.ExecContext(ctx, spec.SQL)
	duration := time.Since(now).Milliseconds()
	progress.Duration = &duration

	if err != nil {
		return fmt.Errorf("sql execution failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("can not get rows affected: %w", err)
	}

	progress.RowsAffected = &rowsAffected
	err = sendSQLTaskProgress(progress, heartbeats)
	if err != nil {
		return err
	}

	log.WithField("rowCount", rowsAffected).Debug("finished")
	return nil
}

func sendSQLTaskProgress(progress SQLTaskProgress, heartbeats chan<- queue.Progress) (err error) {
	logrus.
		WithField("method", "sendSQLTaskProgress").
		Debugf("%+v", progress)

	bytes, err := json.Marshal(progress)
	if err != nil {
		logrus.Error(err)
		return errSerializingHearbeat
	}

	heartbeats <- bytes
	return nil
}
