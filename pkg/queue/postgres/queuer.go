package postgres

import (
	"context"
	"database/sql"

	"github.com/contiamo/go-base/v4/pkg/data/managers"
	"github.com/contiamo/go-base/v4/pkg/queue"
	uuid "github.com/satori/go.uuid"
)

var emptyJSON = []byte("{}")

// NewQueuer creates a new postgres queue queuer
func NewQueuer(db *sql.DB) queue.Queuer {
	return &queuer{
		BaseManager: managers.NewBaseManager(db, "PostgresQueuer"),
	}
}

// NewQueuerWithMetrics creates a new postgres queue queuer with metrics enabled
func NewQueuerWithMetrics(db *sql.DB) queue.Queuer {
	return queue.QueuerWithMetrics(NewQueuer(db))
}

// pgQueue is a postgres backed implementation of the queue manager
type queuer struct {
	managers.BaseManager
}

// Enqueue implements queue.Enqueue
//
// `task` must contain a Queue name and task type.
func (q *queuer) Enqueue(ctx context.Context, task queue.TaskEnqueueRequest) (err error) {
	span, ctx := q.StartSpan(ctx, "Enqueue")
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

	taskID := uuid.NewV4().String()
	span.SetTag("task.queue", task.Queue)
	span.SetTag("task.type", task.Type)
	span.SetTag("task.spec", string(task.Spec))
	span.SetTag("task.references", task.References)

	refColumns, refValues := task.References.GetNamesAndValues()

	_, err = q.GetQueryBuilder().
		Insert("tasks").
		Columns(
			append(
				refColumns,
				"task_id",
				"queue",
				"type",
				"spec",
				"status",
				"progress",
			)...,
		).
		Values(
			append(
				refValues,
				taskID,
				task.Queue,
				task.Type,
				task.Spec,
				queue.Waiting,
				emptyJSON,
			)...,
		).
		ExecContext(ctx)
	if err != nil {
		return err
	}

	return nil
}
