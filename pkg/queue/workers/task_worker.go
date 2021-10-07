package workers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/contiamo/go-base/v4/pkg/queue"

	"github.com/contiamo/go-base/v4/pkg/tracing"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

var (
	maxSpanDuration = 1 * time.Minute
	heartbeatTTL    = 15 * time.Second
	// ErrHeartbeatTimeout is returned from the worker when the task heartbeat is too slow,
	// tasks must heartbeat every 15s or else the worker abandons he task. The task is
	// not marked as a failure, but can be restarted by another worker.
	ErrHeartbeatTimeout = fmt.Errorf("task timeout")
)

type Options struct {
	HeartbeatTTL time.Duration
}

// NewTaskWorker creates a new Task Worker instance, the worker will enforce a default
// heartbeat ttl of 15 seconds.
//
// Deprecated: Use NewWorker instead.
func NewTaskWorker(dequeuer queue.Dequeuer, handler queue.TaskHandler) queue.Worker {
	return NewWorker(dequeuer, handler, Options{HeartbeatTTL: heartbeatTTL})
}

// NewWorker creates a task Worker instance with the specified options.
func NewWorker(dequeuer queue.Dequeuer, handler queue.TaskHandler, opts Options) queue.Worker {
	return &taskWorker{
		Tracer:   tracing.NewTracer("workers", "TaskWorker"),
		dequeuer: dequeuer,
		handler:  handler,
		ttl:      opts.HeartbeatTTL,
	}
}

type taskWorker struct {
	tracing.Tracer

	handler  queue.TaskHandler
	dequeuer queue.Dequeuer

	queue.Worker

	ttl time.Duration
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
	// while ctx is not canceled or interrupted
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
		// check if the iteration was canceled
		case <-ctx.Done():
			logrus.Debug("task processing iteration is interrupted")
			return ctx.Err()
		case <-timer.C:
			// we should not hold tracing spans open for more then maxSpanDuration
			// so, we just restart the span with a new iteration
			return nil
		default:
			logrus.Debug("trying to find a task to process...")

			task, err := w.tryDequeueTask(ctx)
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
			return w.handleTask(ctx, *task)
		}
	}
}

func (w *taskWorker) tryDequeueTask(ctx context.Context) (task *queue.Task, err error) {
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
	}()

	timer := prometheus.NewTimer(queue.TaskWorkerMetrics.ProcessingDuration)
	defer timer.ObserveDuration()

	logger := logrus.WithContext(ctx).
		WithField("worker", "handleTask").
		WithField("queue", task.Queue)

	heartbeats := make(chan queue.Progress)
	processDone := make(chan error, 1)

	go func() {
		// handle panics because we force close the heartbeats if the beats are too slow
		defer func() {
			if r := recover(); r != nil {
				logger.Errorf("Recovered in task handler.Process: %v", r)
			}
		}()
		// force cleanup heartbeats, this might cause a panic ....
		defer close(heartbeats)
		defer close(processDone)
		// handler.Process is responsible for closing the heartbeats channel
		// if `Process` returns an error it means the task failed
		processDone <- w.handler.Process(ctx, task, heartbeats)
	}()

	// block while we process the heartbeats
	progress, err := w.processHeartbeats(ctx, task, heartbeats)
	if err == ErrHeartbeatTimeout {
		// we must try to put the error message in the latest version of progress
		// empty progress (no heartbeats) is also fine
		span.SetTag("err", err)
		queue.TaskWorkerMetrics.ProcessingErrorsCounter.With(labels).Inc()

		progress = w.setError(progress, err)
		return w.dequeuer.Fail(ctx, task.ID, progress)
	}

	if err != nil {
		return err
	}

	// now wait for the worker processing error
	workErr := <-processDone
	if workErr != nil {
		// we must try to put the error message in the latest version of progress
		// empty progress (no heartbeats) is also fine
		span.SetTag("workErr", workErr)
		queue.TaskWorkerMetrics.ProcessingErrorsCounter.With(labels).Inc()

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

// processHeartbeats will synchronously process the heartbeats channel, saving the progress reports to the dequeuer.
// We moved this to a method because using returns is nicer than labels and break.
func (w *taskWorker) processHeartbeats(ctx context.Context, task queue.Task, heartbeats chan queue.Progress) (progress queue.Progress, err error) {
	progress = queue.Progress("{}") // empty progress by default
	ttl := time.NewTimer(w.ttl)
	defer ttl.Stop()

	logger := logrus.WithContext(ctx).
		WithField("worker", "processHeartbeats").
		WithField("queue", task.Queue)

	for {
		select {
		case <-ctx.Done():
			return progress, ctx.Err()
		case t := <-ttl.C:
			logrus.WithField("time", t).Error("heartbeat timeout")
			return progress, ErrHeartbeatTimeout
		// use a temporary p instead of shadowing progress directory so that
		// we don't accidentally nil the progress when the heartbeats closes
		case p, ok := <-heartbeats:
			if !ok {
				logger.Debug("closed heartbeats")
				return progress, nil
			}

			// record the latest valid progress
			progress = p
			hrtErr := w.dequeuer.Heartbeat(ctx, task.ID, progress)
			if hrtErr != nil {
				switch hrtErr {
				case queue.ErrTaskCancelled,
					queue.ErrTaskFinished,
					queue.ErrTaskNotFound,
					queue.ErrTaskNotRunning:
					logger.Error(hrtErr)
					// finished/canceled errors are not considered event errors, stop and return nil
					return progress, nil
				default:
					// fatal error, time to stop
					return progress, hrtErr
				}
			}
			if !ttl.Stop() {
				// ttl has fired and we need to drain the channel
				<-ttl.C
			}
			ttl.Reset(heartbeatTTL)
		}
	}
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
