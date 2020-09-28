package workers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/contiamo/go-base/v2/pkg/queue"

	"github.com/contiamo/go-base/v2/pkg/tracing"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

var (
	maxSpanDuration = 1 * time.Minute
)

// NewTaskWorker creates a new Task Worker instance
func NewTaskWorker(dequeuer queue.Dequeuer, handler queue.TaskHandler) queue.Worker {
	return &taskWorker{
		Tracer:   tracing.NewTracer("workers", "TaskWorker"),
		dequeuer: dequeuer,
		handler:  handler,
	}
}

type taskWorker struct {
	tracing.Tracer

	handler  queue.TaskHandler
	dequeuer queue.Dequeuer

	queue.Worker
}

func (w *taskWorker) Work(ctx context.Context) (err error) {
	tracer := opentracing.GlobalTracer()
	queue.TaskWorkerMetrics.ActiveGauge.Inc()
	defer queue.TaskWorkerMetrics.ActiveGauge.Dec()

	// the error in the iteration should not stop the work
	// it's logged by the Tracer interface, so we don't have to handle it here
	// since the ticker delivers the first tick after the interval we need to run it for the
	// first time out of the loop
	e := w.iteration(ctx, tracer)
	if e != nil {
		logrus.Error(e)
	}

	logrus.Debug("starting task worker loop...")
	// while ctx is not cancelled or interrupted
	for {
		select {
		case <-ctx.Done():
			logrus.Debug("processing loop is interrupted")
			return ctx.Err()
		default:
			e := w.iteration(ctx, tracer)
			if e != nil {
				logrus.Error(e)
			}
		}
	}
}

func (w *taskWorker) iteration(ctx context.Context, tracer opentracing.Tracer) (err error) {
	span := tracer.StartSpan("iteration")
	ctx, cancel := context.WithCancel(opentracing.ContextWithSpan(ctx, span))
	defer func() {
		cancel()
		w.FinishSpan(span, err)
		if err != nil {
			queue.TaskWorkerMetrics.ErrorsCounter.Inc()
		}
	}()

	queue.TaskWorkerMetrics.WorkingGauge.Inc()
	defer queue.TaskWorkerMetrics.WorkingGauge.Dec()

	timer := time.NewTimer(maxSpanDuration)
	defer timer.Stop()

	logrus.Debug("starting work attempt...")

	for {
		select {
		// check if the iteration was cancelled
		case <-ctx.Done():
			logrus.Debug("task processing iteration is interrupted")
			return ctx.Err()
		case <-timer.C:
			// we should not hold tracing spans open for more then maxSpanDuration
			// so, we just restart the span with a new iteration
			return nil
		default:
			logrus.Debug("trying to find a task to process...")

			task, err := w.tryDequeueTask(ctx, tracer)
			// empty queue is not an error
			if err == sql.ErrNoRows {
				return nil
			}
			if err != nil {
				return err
			}

			if task == nil {
				return errors.New("task cannot be nil")
			}
			err = w.handleTask(ctx, *task)
			if err != nil {
				return err
			}
		}
	}
}

func (w *taskWorker) tryDequeueTask(ctx context.Context, tracer opentracing.Tracer) (task *queue.Task, err error) {
	span, ctx := w.StartSpan(ctx, "tryDequeueTask")
	defer func() {
		// this not really an error that we need to log
		// it's just to indicate the calling function to take a break
		// before it tries again
		if err == sql.ErrNoRows {
			w.FinishSpan(span, nil)
		} else {
			w.FinishSpan(span, err)
		}
	}()
	defer func() {
		if err != nil {
			queue.TaskWorkerMetrics.DequeueErrorCounter.Inc()
		}
	}()

	queue.TaskWorkerMetrics.DequeueingGauge.Inc()
	defer queue.TaskWorkerMetrics.DequeueingGauge.Dec()
	timer := prometheus.NewTimer(queue.TaskWorkerMetrics.DequeueingDuration)
	defer timer.ObserveDuration()

	return w.dequeuer.Dequeue(ctx)
}

// handleTask is responsible for actually calling the handler.Process method.  This method includes
// the standardized logic need for metrics and handling cancellation errors
func (w *taskWorker) handleTask(ctx context.Context, task queue.Task) (err error) {
	span, ctx := w.StartSpan(ctx, "handleTask")
	ctx, cancel := context.WithCancel(ctx)
	labels := prometheus.Labels{"queue": task.Queue, "type": task.Type.String()}
	defer func() {
		cancel()
		w.FinishSpan(span, err)
		if err != nil {
			queue.TaskWorkerMetrics.ProcessingErrorsCounter.With(labels).Inc()
		}
	}()

	timer := prometheus.NewTimer(queue.TaskWorkerMetrics.ProcessingDuration)
	defer timer.ObserveDuration()

	log := logrus.WithField("worker", "handleTask")

	heartbeats := make(chan queue.Progress)
	done := make(chan struct{})

	var workErr error
	go func() {
		defer close(done)
		// handler.Process is responsible for closing the heartbeats channel
		// if `Process` returns an error it means the task failed
		workErr = w.handler.Process(ctx, task, heartbeats)
	}()

	// assumes that the handler will close the heartbeats channel when if finishes/errors
	progress := queue.Progress("{}") // empty progress by default
	for progress = range heartbeats {
		hrtErr := w.dequeuer.Heartbeat(ctx, task.ID, progress)
		if hrtErr != nil {
			switch hrtErr {
			case queue.ErrTaskCancelled,
				queue.ErrTaskFinished,
				queue.ErrTaskNotFound,
				queue.ErrTaskNotRunning:
				log.Error(hrtErr)
				// finished/cancelled errors are not considered event errors, stop and return nil
				return nil
			default:
				return hrtErr
			}
		}
	}

	<-done

	if workErr != nil {
		// we must try to put the error message in the latest version of progress
		// empty progress (no heartbeats) is also fine
		progress = w.setError(progress, workErr)
		return w.dequeuer.Fail(ctx, task.ID, progress)
	}

	err = w.dequeuer.Finish(ctx, task.ID, progress)
	if err != nil {
		return err
	}

	queue.TaskWorkerMetrics.ProcessedCounter.With(labels).Inc()
	return nil
}

func (w *taskWorker) setError(progress queue.Progress, err error) queue.Progress {
	p := map[string]interface{}{}
	e := json.Unmarshal(progress, &p)
	if e != nil {
		logrus.
			WithError(e).
			Error("failed to put error message into the task progress")
		return progress
	}
	p["error"] = err.Error()
	bytes, e := json.Marshal(p)
	if e != nil {
		logrus.
			WithError(e).
			Error("failed to marshal updated task progress")
		return progress
	}
	return queue.Progress(bytes)
}
