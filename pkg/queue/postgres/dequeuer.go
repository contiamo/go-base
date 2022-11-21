package postgres

import (
	"context"
	"database/sql"
	"math/rand"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/contiamo/go-base/v4/pkg/data/managers"
	"github.com/contiamo/go-base/v4/pkg/queue"
	"github.com/lib/pq"

	"github.com/contiamo/go-base/v4/pkg/config"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// NewDequeuer creates a new postgres queue dequeuer
func NewDequeuer(db *sql.DB, dbListener *pq.Listener, cfg config.Queue) queue.Dequeuer {
	return &dequeuer{
		BaseManager: managers.NewBaseManager(db, "PostgresDequeuer"),
		cfg:         cfg,
		listener:    dbListener,
	}
}

// NewDequeuerWithMetrics creates a new postgres queue dequeuer with the default metrics enabled
func NewDequeuerWithMetrics(db *sql.DB, dbListener *pq.Listener, cfg config.Queue) queue.Dequeuer {
	return queue.DequeuerWithMetrics(NewDequeuer(db, dbListener, cfg))
}

// dequeuer is a postgres-backed implementation of the queue Dequeuer
type dequeuer struct {
	cfg      config.Queue
	listener *pq.Listener
	managers.BaseManager
}

// Dequeue implements queue.Dequeue
// It's expected to block until work is available
// Allows only one task per queue to be worked on at once
// Returns the oldest task in the queue that hasn't started or has had a failure
func (q *dequeuer) Dequeue(ctx context.Context, queues ...string) (task *queue.Task, err error) {
	span, ctx := q.StartSpan(ctx, "Dequeue")
	defer func() {
		q.FinishSpan(span, err)
	}()

	span.SetTag("queues", queues)

	// if nil, nil is returned, no tasks are available so we will continue to call the db
	// until a task or error is received
	task, err = q.attemptDequeue(ctx, queues...)

	if task == nil && err == nil {
		err = q.listener.Listen("task_update")
		if err != nil {
			return nil, err
		}
		defer func() {
			unlistenError := q.listener.Unlisten("task_update")
			if unlistenError != nil && err == nil {
				err = unlistenError
			}
		}()
		ticker := time.NewTicker(q.cfg.PollFrequency)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-ticker.C:
				logrus.Debug("attempt dequeue because of ticker")
				task, err = q.attemptDequeue(ctx, queues...)
				if task != nil || err != nil {
					return task, err
				}
				err = q.listener.Ping()
				if err != nil {
					return nil, err
				}
			case <-q.listener.Notify:
				logrus.Debug("attempt dequeue because of notification")
				task, err = q.attemptDequeue(ctx, queues...)
				if task != nil || err != nil {
					return task, err
				}
				err = q.listener.Ping()
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return task, err
}

func (q *dequeuer) attemptDequeue(ctx context.Context, queues ...string) (task *queue.Task, err error) {
	span, ctx := q.StartSpan(ctx, "attemptDequeue")
	defer func() {
		q.FinishSpan(span, err)
	}()

	now := time.Now()
	doubleTTL := now.Add(-2 * q.cfg.HeartbeatTTL) // assume a task has failed if there's no heartbeat after twice the ttl

	builder, tx, err := q.GetTxQueryBuilder(ctx, nil)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}

		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			err = errors.Wrap(err, rollbackErr.Error())
		}
	}()

	// if zero length q is passed in, get a list of all queues with unfinished tasks
	if len(queues) == 0 {
		queues, err = q.generateUnfinishedQueueList(ctx)
		if err != nil {
			return nil, err
		}
	}

	inflight, err := q.generateInflightQueueList(ctx, doubleTTL)
	if err != nil {
		return nil, err
	}

	processableQs := processableQueues(inflight, queues)

	span.SetTag("queue.inflightQueues", len(inflight))
	span.SetTag("queue.potentialQueues", len(processableQs))

	// no tasks available
	if len(processableQs) == 0 {
		return nil, nil
	}

	// choose a random queue
	//nolint: gosec // this random value is not involved in any security related logic
	n := rand.Intn(len(processableQs))
	queueName := processableQs[n]
	logrus.Debugf("choosing random queue `%s`", queueName)

	// find the oldest non-started or failed task
	row := builder.
		Select(q.columns()...).
		From("tasks").
		Where(squirrel.Or{
			squirrel.Eq{"queue": queueName, "started_at": nil},
			squirrel.And{
				squirrel.Eq{"queue": queueName, "finished_at": nil},
				squirrel.Lt{"last_heartbeat_at": doubleTTL},
			},
		}).
		OrderBy("created_at DESC").
		Limit(1).
		Suffix("FOR UPDATE").
		QueryRowContext(ctx)

	task, err = q.scan(row)
	if err == sql.ErrNoRows {
		logrus.Debugf("queue `%s` does not have an available task", queueName)
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	span.SetTag("task.timeSpentWaiting", now.Sub(task.CreatedAt))

	// update started_at time
	_, err = builder.
		Update("tasks").
		Set("started_at", now).
		Set("last_heartbeat_at", now).
		Set("status", queue.Running).
		Where("task_id=?", task.ID).
		ExecContext(ctx)
	if err != nil {
		return nil, err
	}

	task.StartedAt = &now
	task.LastHeartbeatAt = &now
	return task, err
}

func (q *dequeuer) Heartbeat(ctx context.Context, taskID string, progress queue.Progress) (err error) {
	span, ctx := q.StartSpan(ctx, "Heartbeat")
	defer func() {
		q.FinishSpan(span, err)
	}()
	span.SetTag("task.ID", taskID)
	span.SetTag("progress", string(progress))
	err = q.updateProgress(ctx, taskID, progress, false, false)

	return err
}

func (q *dequeuer) Finish(ctx context.Context, taskID string, progress queue.Progress) (err error) {
	span, ctx := q.StartSpan(ctx, "Finish")
	defer func() {
		q.FinishSpan(span, err)
	}()

	span.SetTag("task.ID", taskID)
	span.SetTag("progress", string(progress))

	err = q.updateProgress(ctx, taskID, progress, true, false)

	return err
}

func (q *dequeuer) Fail(ctx context.Context, taskID string, progress queue.Progress) (err error) {
	span, ctx := q.StartSpan(ctx, "Fail")
	defer func() {
		q.FinishSpan(span, err)
	}()

	span.SetTag("task.ID", taskID)
	span.SetTag("progress", string(progress))

	err = q.updateProgress(ctx, taskID, progress, true, true)

	return err
}

func (q *dequeuer) updateProgress(ctx context.Context, taskID string, progress queue.Progress, isFinal, isFailed bool) (err error) {
	span, ctx := q.StartSpan(ctx, "updateProgress")
	defer func() {
		q.FinishSpan(span, err)
	}()

	// debug log the progress update because we had a really bad bug were we tried to write an
	// empty progress. This debug statement may be the other content we get in on prem
	logrus.
		WithField("method", "updateProgress").
		WithField("task", taskID).
		WithField("isFinal", isFinal).
		WithField("isFailed", isFailed).
		Debug(string(progress))

	// ensure that progress is always a valid JSON object
	if len(progress) == 0 {
		progress = emptyJSON
	}

	builder, tx, err := q.GetTxQueryBuilder(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}

		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			err = errors.Wrap(err, rollbackErr.Error())
		}
	}()

	now := time.Now()

	var status queue.TaskStatus
	err = builder.
		Select("status").
		From("tasks").
		Where("task_id = ?", taskID).
		Suffix("FOR UPDATE").
		QueryRowContext(ctx).
		Scan(&status)

	if err == sql.ErrNoRows {
		return queue.ErrTaskNotFound
	}
	if err != nil {
		return err
	}

	// we should proceed ONLY if the task is actually running
	switch status {
	case queue.Waiting:
		return queue.ErrTaskNotRunning
	case queue.Cancelled:
		return queue.ErrTaskCancelled
	case queue.Finished:
		return queue.ErrTaskFinished
	}

	stmt := builder.
		Update("tasks").
		Set("last_heartbeat_at", now).
		Set("progress", progress).
		Where("task_id = ?", taskID)

	switch {
	case isFinal && isFailed:
		stmt = stmt.Set("finished_at", now).Set("status", queue.Failed)
	case isFinal && !isFailed:
		stmt = stmt.Set("finished_at", now).Set("status", queue.Finished)
	case !isFinal && isFailed:
		stmt = stmt.Set("status", queue.Failed)
	case !isFinal && !isFailed:
		stmt = stmt.Set("status", queue.Running)
	}

	_, err = stmt.ExecContext(ctx)

	return err
}

func (q *dequeuer) columns() []string {
	return []string{
		"task_id",
		"queue",
		"type",
		"spec",
		"status",
		"progress",
		"created_at",
		"updated_at",
		"started_at",
		"finished_at",
		"last_heartbeat_at",
	}
}

func (q *dequeuer) scan(row squirrel.RowScanner) (task *queue.Task, err error) {
	var t queue.Task
	err = row.Scan(
		&t.ID,
		&t.Queue,
		&t.Type,
		&t.Spec,
		&t.Status,
		&t.Progress,
		&t.CreatedAt,
		&t.UpdatedAt,
		&t.StartedAt,
		&t.FinishedAt,
		&t.LastHeartbeatAt,
	)
	if err == nil {
		task = &t
		// ensure that we always have at least an empty json object
		if len(task.Progress) == 0 {
			task.Progress = emptyJSON
		}
		if len(task.Spec) == 0 {
			task.Spec = emptyJSON
		}
	}

	return task, err
}

// generateUnfinishedQueueList returns a distinct list of queues with unfinished tasks
func (q *dequeuer) generateUnfinishedQueueList(ctx context.Context) (queueList []string, err error) {
	span, ctx := q.StartSpan(ctx, "generateUnfinishedQueueList")
	defer func() {
		q.FinishSpan(span, err)
	}()

	where := squirrel.Eq{
		"finished_at": nil,
	}

	return q.generateQueueList(ctx, where)
}

// generateInflightQueueList returns a distinct list of queues with tasks that are currently inflight
func (q *dequeuer) generateInflightQueueList(ctx context.Context, beforeDoubleTTL time.Time) (queueList []string, err error) {
	span, ctx := q.StartSpan(ctx, "generateInflightQueueList")
	defer func() {
		q.FinishSpan(span, err)
	}()

	where := squirrel.And{
		squirrel.Gt{"last_heartbeat_at": beforeDoubleTTL},
		squirrel.Eq{"finished_at": nil},
	}

	return q.generateQueueList(ctx, where)
}

// generateQueueList returns a distinct list of queues
func (q *dequeuer) generateQueueList(ctx context.Context, where squirrel.Sqlizer) (queueList []string, err error) {
	span, ctx := q.StartSpan(ctx, "generateQueueList")
	defer func() {
		q.FinishSpan(span, err)
	}()

	rows, err := q.GetQueryBuilder().
		Select("queue").
		Distinct().
		From("tasks").
		Where(where).
		QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var q string
		err = rows.Scan(&q)
		if err != nil {
			return nil, err
		}

		queueList = append(queueList, q)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	span.SetTag("queue.list", queueList)

	return queueList, nil
}

// processableQueues removes all inflight items from potential items
func processableQueues(inflight, potential []string) []string {
	inflightMap := make(map[string]bool, len(inflight))
	for _, s := range inflight {
		inflightMap[s] = true
	}
	var processableQs []string
	for _, s := range potential {
		_, ok := inflightMap[s]
		if !ok {
			processableQs = append(processableQs, s)
		}
	}

	return processableQs
}
