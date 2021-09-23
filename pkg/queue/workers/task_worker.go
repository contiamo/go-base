package workers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
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
	// HeartbeatTimeout is returned from the worker when the task heartbeat is too slow,
	// tasks must heartbeat every 15s or else the worker abandons he task. The task is
	// not marked as a failure, but can be restarted by another worker.
	HeartbeatTimeout = fmt.Errorf("task timeout")
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

type taskDone struct {
	err            error
	fromProcessing bool
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

	log := logrus.WithContext(ctx).
		WithField("worker", "handleTask").
		WithField("queue", task.Queue)

	heartbeats := make(chan queue.Progress)

	// use cleanup to signal to the hearbeat loop that Processing is done to
	// avoid any leaks and to handle edge cases where heatbeats doesn't close
	// or the heartbeats are very slow.
	cleanup := make(chan struct{}, 1)
	defer close(cleanup)

	processDone := make(chan taskDone)
	go func() {
		defer close(processDone)
		// handler.Process is responsible for closing the heartbeats channel
		// if `Process` returns an error it means the task failed
		err := w.handler.Process(ctx, task, heartbeats)

		processDone <- taskDone{err: err, fromProcessing: true}
	}()

	heartbeatsDone := make(chan taskDone)
	progress := queue.Progress("{}") // empty progress by default
	go func() {
		defer close(heartbeatsDone)

		ttl := time.NewTimer(heartbeatTTL)
		for {
			select {
			case <-ttl.C:
				processDone <- taskDone{err: HeartbeatTimeout, fromProcessing: false}
				return
			case <-cleanup:
				return

			// use a temporary p instead of shadowing progress directory so that
			// we don't accidentally nil the progress when the heartbeats closes
			case p, ok := <-heartbeats:
				if !ok {
					log.Debug("closed heartbeats")
					return
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
						log.Error(hrtErr)
						// finished/canceled errors are not considered event errors, stop and return nil
						return
					default:
						// fatal error, time to stop
						heartbeatsDone <- taskDone{err: hrtErr, fromProcessing: false}
						return
					}
				}

				if !ttl.Stop() {
					// ttl has fired and we need to drain the channel
					<-ttl.C
				}
				ttl.Reset(heartbeatTTL)
			}
		}
	}()

	// use separate channels and merge to avoid pushes to closed channels
	done := merge(processDone, heartbeatsDone)
	defer func() {
		// drain done to avoid leaks
		for range done {
		}
	}()

	workErr := <-done
	// there is a potential race condition at the end of the Process(). The implementation may do something like
	//
	// 		heartbeats <- progress
	// 		return err
	//
	// We now have a small race between the Process goroutine that pushed `err` into the done channel
	// and the heartbeat goroutine that is processing `progress`. This small sleep  should help capture that final progress
	// item most of the time, if there is one.
	time.Sleep(time.Millisecond)
	cleanup <- struct{}{}

	if workErr.err != nil && workErr.fromProcessing {
		// we must try to put the error message in the latest version of progress
		// empty progress (no heartbeats) is also fine
		span.SetTag("workErr", workErr)
		queue.TaskWorkerMetrics.ProcessingErrorsCounter.With(labels).Inc()

		progress = w.setError(progress, workErr.err)
		return w.dequeuer.Fail(ctx, task.ID, progress)
	}

	// non-processing error, usually a DB or serialization failure
	if workErr.err != nil {
		return workErr.err
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

func merge(cs ...<-chan taskDone) <-chan taskDone {
	out := make(chan taskDone)
	var wg sync.WaitGroup
	wg.Add(len(cs))
	for _, c := range cs {
		go func(c <-chan taskDone) {
			for v := range c {
				out <- v
			}
			wg.Done()
		}(c)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}
