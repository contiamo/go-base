package workers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/contiamo/go-base/v4/pkg/queue"

	"github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/contiamo/go-base/v4/pkg/http/middlewares/authorization"
)

func TestTaskWorkerMetrics(t *testing.T) {
	defer goleak.VerifyNone(t)

	logrus.SetOutput(ioutil.Discard)
	defer logrus.SetOutput(os.Stdout)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	qCh := make(chan *queue.Task)
	q := &mockQueue{queue: qCh}

	handler := queue.TaskHandlerFunc(func(ctx context.Context, task queue.Task, heartbeats chan<- queue.Progress) error {
		defer close(heartbeats)

		require.NotNil(t, task)
		require.Equal(t, "testTask", task.ID)
		require.Equal(t, "testQueue", task.Queue)

		select {
		case <-ctx.Done():
			return nil
		case <-time.NewTimer(time.Second).C:
			// mimic a successful processing
			heartbeats <- queue.Progress{}
		}

		return nil
	})
	w := NewTaskWorker(q, handler)

	t.Run("worker count starts at 0", func(t *testing.T) {
		require.Equal(t, float64(0), testutil.ToFloat64(queue.TaskWorkerMetrics.ActiveGauge))
	})
	go func() {
		err := w.Work(ctx)
		if err != nil {
			logrus.Debug(err)
		}
	}()

	metricSleep := 50 * time.Millisecond
	time.Sleep(metricSleep)
	testTask := &queue.Task{TaskBase: queue.TaskBase{Queue: "testQueue"}, ID: "testTask"}
	t.Run("calling work inc the worker", func(t *testing.T) {
		require.Equal(t, float64(1), testutil.ToFloat64(queue.TaskWorkerMetrics.ActiveGauge))
		require.Equal(t, float64(1), testutil.ToFloat64(queue.TaskWorkerMetrics.WorkingGauge))
	})

	qCh <- testTask
	time.Sleep(metricSleep)
	t.Run("active worker count inc after enqueueing a task", func(t *testing.T) {
		require.Equal(t, float64(1), testutil.ToFloat64(queue.TaskWorkerMetrics.WorkingGauge))
	})

	cancel()
	time.Sleep(metricSleep)
	t.Run("worker count returns to 0 when worker is canceled", func(t *testing.T) {
		require.Equal(t, float64(0), testutil.ToFloat64(queue.TaskWorkerMetrics.WorkingGauge))
	})
}

func TestTaskWorkerWork(t *testing.T) {
	defer goleak.VerifyNone(t)

	logrus.SetOutput(ioutil.Discard)
	defer logrus.SetOutput(os.Stdout)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	errorCounter := queue.TaskWorkerMetrics.ProcessingErrorsCounter.With(prometheus.Labels{"queue": "testQueue", "type": "testType"})
	require.Equal(t, float64(0), testutil.ToFloat64(errorCounter))

	taskTimeoutStatus := `{"error":"task timeout"}`
	t.Run("worker handles multiple tasks without stopping", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		qCh := make(chan *queue.Task, 2)
		q := &mockQueue{queue: qCh}
		seenTasks := []string{}

		testTaskOne := &queue.Task{
			TaskBase: queue.TaskBase{
				Queue: "testQueue",
				Type:  "testType",
			},
			ID: "testTask1",
		}
		testTaskTwo := &queue.Task{
			TaskBase: queue.TaskBase{
				Queue: "testQueue",
				Type:  "testType",
			},
			ID: "testTask2",
		}
		qCh <- testTaskOne
		qCh <- testTaskTwo

		handler := queue.TaskHandlerFunc(func(ctx context.Context, task queue.Task, heartbeats chan<- queue.Progress) error {
			defer close(heartbeats)
			seenTasks = append(seenTasks, task.ID)
			return nil
		})

		w := NewTaskWorker(q, handler)

		done := make(chan error)
		go func() {
			done <- w.Work(ctx)
		}()

		time.Sleep(5 * time.Millisecond)
		cancel()
		<-done
		require.Equal(t, []string{testTaskOne.ID, testTaskTwo.ID}, seenTasks)
	})

	t.Run("worker sets the error to the progress if handler returns an error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		qCh := make(chan *queue.Task, 1)
		q := &mockQueue{queue: qCh}

		testTask := &queue.Task{
			TaskBase: queue.TaskBase{
				Queue: "testQueue",
				Type:  "testType",
			},
			ID: "testTask",
		}
		qCh <- testTask

		handler := queue.TaskHandlerFunc(func(ctx context.Context, task queue.Task, heartbeats chan<- queue.Progress) error {
			defer close(heartbeats)
			return errors.New("some serious error")
		})

		w := NewTaskWorker(q, handler)

		done := make(chan error)
		go func() {
			done <- w.Work(ctx)
		}()

		time.Sleep(3 * time.Millisecond)
		cancel()
		err := <-done
		// we should get the context error from the Work thread because the dequeue
		// is not a fatal error, but we should see the dequeue error in the logs
		require.EqualError(t, err, "context canceled")

		expStatus := `{"error":"some serious error"}`
		require.Equal(t, []queue.Progress{queue.Progress(expStatus)}, q.fails)

		require.Equal(t, float64(1), testutil.ToFloat64(errorCounter))
	})

	t.Run("worker sets the error to the latest progress if handler returns an error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		qCh := make(chan *queue.Task, 1)
		q := &mockQueue{queue: qCh}

		testTask := &queue.Task{
			TaskBase: queue.TaskBase{
				Queue: "testQueue",
				Type:  "testType",
			},
			ID: "testTask",
		}
		qCh <- testTask

		handler := queue.TaskHandlerFunc(func(ctx context.Context, task queue.Task, heartbeats chan<- queue.Progress) error {
			defer close(heartbeats)
			heartbeats <- queue.Progress(`{"some":"text"}`)
			return errors.New("some serious error")
		})

		w := NewTaskWorker(q, handler)

		done := make(chan error)
		go func() {
			done <- w.Work(ctx)
		}()

		time.Sleep(3 * time.Millisecond)
		cancel()
		err := <-done
		// we should get the context error from the Work thread because the dequeue
		// is not a fatal error, but we should see the dequeue error in the logs
		require.EqualError(t, err, "context canceled")

		expStatus := `{"error":"some serious error","some":"text"}`
		require.Equal(t, []queue.Progress{queue.Progress(expStatus)}, q.fails)
		require.Equal(t, float64(2), testutil.ToFloat64(errorCounter))
	})

	t.Run("worker does not stop and logs the error when queue returns a dequeue error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		var logs bytes.Buffer
		logrus.SetOutput(&logs)

		qCh := make(chan *queue.Task, 1)
		q := &mockQueue{
			queue:      qCh,
			dequeueErr: errors.New("can not dequeue"),
		}

		testTask := &queue.Task{
			TaskBase: queue.TaskBase{
				Queue: "testQueue",
				Type:  "testType",
			},
			ID: "testTask",
		}
		qCh <- testTask

		handler := queue.TaskHandlerFunc(func(ctx context.Context, task queue.Task, heartbeats chan<- queue.Progress) error {
			defer close(heartbeats)

			require.NotNil(t, task)
			require.Equal(t, testTask.ID, task.ID)
			require.Equal(t, testTask.Queue, task.Queue)

			select {
			case <-ctx.Done():
				return nil
			case <-time.NewTimer(time.Second).C:
				// mimic a successful processing
				heartbeats <- queue.Progress{}
			}

			return nil
		})

		w := NewTaskWorker(q, handler)

		done := make(chan error)
		go func() {
			done <- w.Work(ctx)
		}()

		time.Sleep(3 * time.Millisecond)
		cancel()
		err := <-done
		// we should get the context error from the Work thread because the dequeue
		// is not a fatal error, but we should see the dequeue error in the logs
		require.EqualError(t, err, "context canceled")
		require.Contains(t, logs.String(), "can not dequeue")

		t.Run("should not increment worker error during queue errors", func(t *testing.T) {
			require.Equal(t, float64(2), testutil.ToFloat64(errorCounter))
		})
	})

	t.Run("worker logs the error when heartbeat returns a non-terminal error", func(t *testing.T) {
		var logs bytes.Buffer
		logrus.SetOutput(&logs)

		qCh := make(chan *queue.Task, 1)
		testTask := &queue.Task{TaskBase: queue.TaskBase{Queue: "testQueue"}, ID: "testTask"}
		handler := queue.TaskHandlerFunc(func(ctx context.Context, task queue.Task, heartbeats chan<- queue.Progress) error {
			defer close(heartbeats)

			require.NotNil(t, task)
			require.Equal(t, testTask.ID, task.ID)
			require.Equal(t, testTask.Queue, task.Queue)

			select {
			case <-ctx.Done():
				return nil
			case <-time.NewTimer(2 * time.Millisecond).C:
				// mimic a successful processing
				heartbeats <- queue.Progress{}
			}

			return nil
		})

		allowedFinishedErrors := []error{
			queue.ErrTaskCancelled,
			queue.ErrTaskFinished,
			queue.ErrTaskNotFound,
			queue.ErrTaskNotRunning,
		}

		for _, err := range allowedFinishedErrors {
			t.Run(fmt.Sprintf("%s does not kill worker", err), func(t *testing.T) {
				logs.Reset()
				qCh <- testTask

				ctx, cancel := context.WithCancel(ctx)

				w := NewTaskWorker(&mockQueue{queue: qCh, heartbeatErr: err}, handler)

				done := make(chan error)
				go func() {
					done <- w.Work(ctx)
				}()

				time.Sleep(5 * time.Millisecond)
				cancel()
				err := <-done
				// we should get the context error from the Work thread because the dequeue
				// is not a fatal error, but we should see the dequeue error in the logs
				require.EqualError(t, err, "context canceled")
				require.Contains(t, logs.String(), err.Error())
			})
		}

		t.Run("should not increment worker error during heartbeat errors", func(t *testing.T) {
			require.Equal(t, float64(2), testutil.ToFloat64(errorCounter))
		})
	})
	t.Run("worker stops if process stops without closing the heartbeats", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		qCh := make(chan *queue.Task, 1)
		q := &mockQueue{queue: qCh}

		testTask := &queue.Task{
			TaskBase: queue.TaskBase{
				Queue: "testQueue",
				Type:  "testType",
			},
			ID: "testTask",
		}
		qCh <- testTask

		handler := queue.TaskHandlerFunc(func(ctx context.Context, task queue.Task, heartbeats chan<- queue.Progress) error {
			// explicitly don't close the heartbeats
			// defer close(heartbeats)
			// time.Sleep(time.Second)
			return errors.New("heartbeats not closed error")
		})

		w := NewTaskWorker(q, handler)

		done := make(chan error)
		go func() {
			done <- w.Work(ctx)
		}()

		// give time for the long task and then stop the worker
		time.Sleep(100 * time.Millisecond)
		cancel()
		<-done
		expStatus := `{"error":"heartbeats not closed error"}`
		require.Equal(t, float64(3), testutil.ToFloat64(errorCounter))
		require.Equal(t, []queue.Progress{queue.Progress(expStatus)}, q.fails)
	})

	t.Run("slow heartbeats triggers an error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		qCh := make(chan *queue.Task, 1)
		q := &mockQueue{queue: qCh}

		testTask := &queue.Task{
			TaskBase: queue.TaskBase{
				Queue: "testQueue",
				Type:  "testType",
			},
			ID: "testTask",
		}
		qCh <- testTask

		opts := Options{HeartbeatTTL: time.Second}
		slow := opts.HeartbeatTTL + time.Second
		handler := queue.TaskHandlerFunc(func(ctx context.Context, task queue.Task, heartbeats chan<- queue.Progress) error {
			time.Sleep(slow)
			return errors.New("some serious error")
		})

		w := NewWorker(q, handler, opts)

		done := make(chan error)
		go func() {
			done <- w.Work(ctx)
		}()

		// give time for the long task and then stop the worker
		time.Sleep(slow + 100*time.Millisecond)
		cancel()
		err := <-done
		require.EqualError(t, err, "context canceled")

		require.Equal(t, 0, len(q.heartbeats))
		require.Equal(t, 1, len(q.fails))
		require.Equal(t, 0, len(q.finishes))
		require.Equal(t, float64(4), testutil.ToFloat64(errorCounter))

		require.Equal(t, []queue.Progress{queue.Progress(taskTimeoutStatus)}, q.fails)
	})

	t.Run("long exponential backoff eventually stops the task", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		qCh := make(chan *queue.Task, 1)
		q := &mockQueue{queue: qCh}

		testTask := &queue.Task{
			TaskBase: queue.TaskBase{
				Queue: "testQueue",
				Type:  "testType",
			},
			ID: "testTask",
		}
		qCh <- testTask

		var backoffErr error
		handler := queue.TaskHandlerFunc(func(ctx context.Context, task queue.Task, heartbeats chan<- queue.Progress) error {
			defer close(heartbeats)
			plan := backoff.WithMaxRetries(
				backoff.WithContext(backoff.NewExponentialBackOff(), ctx),
				10,
			)
			// every attempt will trigger a retry
			backoffErr = backoff.Retry(func() error {
				time.Sleep(time.Second)
				return fmt.Errorf("retryable error")
			}, plan)

			return errors.New("some serious error")
		})

		opts := Options{HeartbeatTTL: 3 * time.Second}
		w := NewWorker(q, handler, opts)

		done := make(chan error)
		go func() {
			done <- w.Work(ctx)
		}()

		// give time for the long task and then stop the worker
		time.Sleep(opts.HeartbeatTTL + 3*time.Millisecond)
		cancel()
		err := <-done
		require.EqualError(t, err, "context canceled")
		require.EqualError(t, backoffErr, "context canceled", "handler backoff should fail because the task context is canceled by the worker")

		require.Equal(t, 0, len(q.heartbeats))
		require.Equal(t, 1, len(q.fails))
		require.Equal(t, 0, len(q.finishes))
		require.Equal(t, float64(5), testutil.ToFloat64(errorCounter))

		require.Equal(t, []queue.Progress{queue.Progress(taskTimeoutStatus)}, q.fails)
	})
}

type mockQueue struct {
	queue        chan *queue.Task
	dequeueErr   error
	heartbeatErr error
	finishErr    error
	failErr      error
	heartbeats   []queue.Progress
	finishes     []queue.Progress
	fails        []queue.Progress
}

func (q *mockQueue) Enqueue(ctx context.Context, task queue.Task, claims authorization.Claims) error {
	q.queue <- &task
	return nil
}

func (q *mockQueue) Dequeue(ctx context.Context, queue ...string) (*queue.Task, error) {
	if q.dequeueErr != nil {
		return nil, q.dequeueErr
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case t := <-q.queue:
		return t, nil
	}
}

func (q *mockQueue) Heartbeat(ctx context.Context, taskID string, metadata queue.Progress) error {
	q.heartbeats = append(q.heartbeats, metadata)
	return q.heartbeatErr
}

func (q *mockQueue) Finish(ctx context.Context, taskID string, metadata queue.Progress) error {
	q.finishes = append(q.finishes, metadata)
	return q.finishErr
}

func (q *mockQueue) Fail(ctx context.Context, taskID string, metadata queue.Progress) error {
	q.fails = append(q.fails, metadata)
	return q.failErr
}
