package workers

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/contiamo/go-base/v2/pkg/queue"

	"github.com/contiamo/go-base/v2/pkg/tracing"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type queueEvent struct {
	task *queue.Task
	err  error
}

// NewTaskWorker creates a new Task Worker instance
func NewTaskWorker(dequeuer queue.Dequeuer, handler queue.TaskHandler) queue.Worker {
	return &taskWorker{
		Tracer:   tracing.NewTracer("workers", "TaskWorker"),
		dequeuer: dequeuer,
		handler:  handler,
		maxWait:  1 * time.Minute,
	}
}

type taskWorker struct {
	tracing.Tracer

	handler  queue.TaskHandler
	dequeuer queue.Dequeuer
	maxWait  time.Duration

	queue.Worker
}

func (w *taskWorker) iteration(ctx context.Context, tracer opentracing.Tracer, ticker *time.Ticker) (err error) {
	logrus.Debug("starting work attempt")

	span := tracer.StartSpan("iteration")
	ctx, cancel := context.WithCancel(opentracing.ContextWithSpan(ctx, span))
	defer func() {
		// stop any ongoing dequeue attempt or work
		cancel()
		w.FinishSpan(span, err)
	}()

	events := w.startDequeue(ctx)
	TaskQueueMetrics.WorkerWaiting.Inc()
	logrus.Debug("waiting to dequeue a task")

	select {
	case <-ctx.Done():
		err = ctx.Err()
		if err != nil {
			return err
		}
		span.SetTag("cancelled", "true")
		logrus.Debug("worker cancelled")
	case <-ticker.C:
		// set a ticker so that the tracing/logging isn't too long, otherwise
		// we can end up with spans that are hours long
		span.SetTag("skipped", "true")
		logrus.Debug("work loop skipped/reset")
		return nil
	case event := <-events:
		err = w.verifyQueueEvent(event)
		if err != nil {
			logrus.Errorf("error dequeueing task: %s", err.Error())
			TaskQueueMetrics.QueueErrors.Inc()
			return nil
		}

		return w.handleTask(ctx, *event.task)
	}

	return nil
}

// handleTask is responsible for actually calling the handler.Process method.  This method includes
// the standardized logic need for metrics and handling cancellation errors
func (w *taskWorker) handleTask(ctx context.Context, task queue.Task) (err error) {
	span, ctx := w.StartSpan(ctx, "handleTask")
	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		cancel()
		w.FinishSpan(span, err)

	}()

	log := logrus.WithField("worker", "handleTask")

	l := prometheus.Labels{"queue": task.Queue, "type": task.Type.String()}
	timer := prometheus.NewTimer(TaskQueueMetrics.TaskDuration.With(l))
	defer timer.ObserveDuration()

	TaskQueueMetrics.WorkerWorking.With(l).Inc()
	TaskQueueMetrics.WorkerTask.With(l).Inc()

	TaskQueueMetrics.WorkerWorkingGauge.With(l).Inc()
	defer TaskQueueMetrics.WorkerWorkingGauge.With(l).Dec()

	heartbeats := make(chan queue.Progress)
	done := make(chan struct{})

	var workErr error
	go func() {
		defer close(done)
		// handler.Process is responsible for closing the heartbeats channel
		// if `Process` returns an error it means the task failed
		workErr = w.handler.Process(ctx, task, heartbeats)
		if workErr != nil {
			TaskQueueMetrics.WorkerErrors.With(l).Inc()
		}
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

	return w.dequeuer.Finish(ctx, task.ID, progress)
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

func (w *taskWorker) Work(ctx context.Context) (err error) {
	logrus.Debug("starting task worker")
	TaskQueueMetrics.WorkerGauge.Inc()
	defer TaskQueueMetrics.WorkerGauge.Dec()

	tracer := opentracing.GlobalTracer()

	ticker := time.NewTicker(w.maxWait)
	defer ticker.Stop()

	// while ctx is not cancelled or interrupted
	for err == nil {
		err = w.iteration(ctx, tracer, ticker)
	}

	return err
}

// verifyQueueEvent is a helper to detect if the queue emitted an error
func (w *taskWorker) verifyQueueEvent(event *queueEvent) error {
	if event.err != nil {
		return event.err
	}

	if event.task == nil {
		return errors.New("unexpected empty task")
	}
	return nil
}

// dequeue wraps the Dequeue in channels to make the channel select easier
func (w *taskWorker) startDequeue(ctx context.Context) <-chan *queueEvent {
	queueEvents := make(chan *queueEvent, 1)
	go func() {
		timer := prometheus.NewTimer(TaskQueueMetrics.DequeueDuration)
		defer timer.ObserveDuration()

		t, err := w.dequeuer.Dequeue(ctx)

		if t != nil {
			l := prometheus.Labels{"queue": t.Queue, "type": t.Type.String()}
			TaskQueueMetrics.TaskWaiting.With(l).Observe(time.Since(t.CreatedAt).Seconds())
		}

		queueEvents <- &queueEvent{task: t, err: err}
	}()

	return queueEvents
}
