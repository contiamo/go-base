package workers

import (
	"context"
	"database/sql"
	"time"

	"github.com/Masterminds/squirrel"
	cdb "github.com/contiamo/go-base/v2/pkg/db"
	"github.com/contiamo/go-base/v2/pkg/queue"
	"github.com/contiamo/go-base/v2/pkg/tracing"
	cvalidation "github.com/contiamo/go-base/v2/pkg/validation"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/robfig/cron"
	"github.com/sirupsen/logrus"
)

var (
	// ErrScheduleQueueIsEmpty occurs when there is no schedule with
	// the `next_execution_time` in the past
	ErrScheduleQueueIsEmpty = errors.New("nothing to schedule")
)

// NewScheduleWorker creates a new task scheduling worker
func NewScheduleWorker(db *sql.DB, queue queue.Queuer, interval time.Duration) queue.Worker {
	return newScheduleWorker(db, queue, interval)
}

func newScheduleWorker(db *sql.DB, queue queue.Queuer, interval time.Duration) *scheduleWorker {
	return &scheduleWorker{
		Tracer:   tracing.NewTracer("workers", "ScheduleWorker"),
		queue:    queue,
		interval: interval,
		db:       db,
	}
}

type scheduleWorker struct {
	tracing.Tracer
	db       *sql.DB
	queue    queue.Queuer
	interval time.Duration
}

// Work starts an infinite loop with work iterations and waits the given
// amount of time between iterations
func (w *scheduleWorker) Work(ctx context.Context) (err error) {
	tracer := opentracing.GlobalTracer()
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	queue.ScheduleWorkerMetrics.ActiveGauge.Inc()
	defer queue.ScheduleWorkerMetrics.ActiveGauge.Dec()

	// the error in the iteration should not stop the work
	// it's logged by the Tracer interface, so we don't have to handle it here
	// since the ticker delivers the first tick after the interval we need to run it for the
	// first time out of the loop
	e := w.iteration(ctx, tracer)
	if e != nil {
		logrus.Error(e)
	}

	logrus.Debug("starting task scheduling loop...")
	// while ctx is not cancelled or interrupted
	for {
		queue.ScheduleWorkerMetrics.WaitingGauge.Inc()

		select {
		case <-ctx.Done():
			queue.ScheduleWorkerMetrics.WaitingGauge.Dec()
			logrus.Debug("scheduling loop is interrupted")
			return ctx.Err()
		case <-ticker.C:
			queue.ScheduleWorkerMetrics.WaitingGauge.Dec()
			e := w.iteration(ctx, tracer)
			if e != nil {
				logrus.Error(e)
			}
		}
	}
}

// iteration finds all the tasks that require to be scheduled as tasks and then returns
// `nil` if there is no error while doing so
// One iteration is dealing with multiple tasks scheduling until there is no task to
// schedule anymore, than it makes a break until the next `w.interval` tick
func (w *scheduleWorker) iteration(ctx context.Context, tracer opentracing.Tracer) (err error) {
	span := tracer.StartSpan("iteration")
	ctx, cancel := context.WithCancel(opentracing.ContextWithSpan(ctx, span))
	defer func() {
		cancel()
		w.FinishSpan(span, err)
		if err != nil {
			queue.ScheduleWorkerMetrics.ErrorsCounter.Inc()
		}
	}()

	queue.ScheduleWorkerMetrics.WorkingGauge.Inc()
	defer queue.ScheduleWorkerMetrics.WorkingGauge.Dec()

	logrus.Debug("starting task scheduling iteration...")
	for {
		// check if the iteration was cancelled
		err = ctx.Err()
		if err != nil {
			logrus.Debug("task scheduling iteration is interrupted")
			return err
		}

		logrus.Debug("trying to find a task to schedule...")
		err = w.scheduleTask(ctx)
		if err == ErrScheduleQueueIsEmpty {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func (w *scheduleWorker) scheduleTask(ctx context.Context) (err error) {
	span, ctx := w.StartSpan(ctx, "scheduleTask")
	defer func() {
		// this not really an error that we need to log
		// it's just to indicate the calling function to take a break
		// before it tries again
		if err == ErrScheduleQueueIsEmpty {
			w.FinishSpan(span, nil)
		} else {
			w.FinishSpan(span, err)
		}
	}()

	tx, err := w.db.BeginTx(ctx, nil)
	builder := squirrel.StatementBuilder.
		PlaceholderFormat(squirrel.Dollar).
		RunWith(cdb.WrapWithTracing(tx))
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

	var (
		scheduleID   string
		cronSchedule string
		taskQueue    string
		taskType     queue.TaskType
		specBytes    []byte
	)

	timer := prometheus.NewTimer(queue.ScheduleWorkerMetrics.DequeueingDuration)

	// this will lock the schedule row until the transaction is closed
	row := builder.
		Select(
			"schedule_id",
			"task_type",
			"task_queue",
			"task_spec",
			"cron_schedule",
		).
		From("schedules").
		Where(squirrel.LtOrEq{"next_execution_time": time.Now()}).
		OrderBy("next_execution_time").
		Limit(1).
		// Skipping locked rows provides an inconsistent view of the data,
		// so this is not suitable for general purpose work,
		// but can be used to avoid lock contention with multiple consumers
		// accessing a queue-like table.
		// https://devdocs.io/postgresql~10/sql-select#SQL-FOR-UPDATE-SHARE
		Suffix("FOR NO KEY UPDATE SKIP LOCKED").
		QueryRowContext(ctx)

	err = row.Scan(
		&scheduleID,
		&taskType,
		&taskQueue,
		&specBytes,
		&cronSchedule,
	)
	timer.ObserveDuration()

	if err == sql.ErrNoRows {
		return ErrScheduleQueueIsEmpty
	}
	if err != nil {
		queue.ScheduleWorkerMetrics.DequeueErrorCounter.Inc()
		return err
	}

	labels := prometheus.Labels{"queue": taskQueue, "type": taskType.String()}
	defer func() {
		if err != nil {
			queue.ScheduleWorkerMetrics.ProcessingErrorsCounter.With(labels).Inc()
		}
	}()
	timer = prometheus.NewTimer(queue.ScheduleWorkerMetrics.ProcessingDuration)
	defer timer.ObserveDuration()

	span.SetTag("schedule.id", scheduleID)
	span.SetTag("schedule.cron", cronSchedule)
	span.SetTag("task.type", taskType.String())
	span.SetTag("task.queue", taskQueue)
	span.SetTag("task.spec", string(specBytes))

	logrus := logrus.WithField("type", taskType).WithField("queue", taskQueue)

	logrus.Debug("adding the task to the queue...")
	task := queue.TaskEnqueueRequest{
		TaskBase: queue.TaskBase{
			Queue: taskQueue,
			Type:  taskType,
			Spec:  specBytes,
		},
		References: queue.References{
			"schedule_id": scheduleID,
		},
	}
	err = w.queue.Enqueue(ctx, task)
	if err != nil {
		return err
	}

	logrus.Debug("task has been scheduled successfully")

	logrus.Debug("calculating and updating the next execution time...")

	var nextExecution *time.Time
	if cronSchedule != "" {
		p := cron.NewParser(cvalidation.JobCronFormat)
		schedule, err := p.Parse(cronSchedule)
		if err != nil {
			return err
		}
		t := schedule.Next(time.Now())
		nextExecution = &t
	}

	_, err = builder.
		Update("schedules").
		Set("next_execution_time", nextExecution).
		Where(squirrel.Eq{
			"schedule_id": scheduleID,
		}).
		ExecContext(ctx)
	if err != nil {
		return err
	}
	logrus.Debug("the new execution time is set")

	queue.ScheduleWorkerMetrics.ProcessedCounter.With(labels).Inc()
	return nil
}
