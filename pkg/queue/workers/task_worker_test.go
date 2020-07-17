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

	"github.com/contiamo/go-base/pkg/queue"

	"github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/contiamo/go-base/pkg/http/middlewares/authorization"
)

func Test_WorkerMetrics(t *testing.T) {
	defer goleak.VerifyNone(t)

	logrus.SetOutput(ioutil.Discard)
	defer logrus.SetOutput(os.Stdout)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	qCh := make(chan *queue.Task)
	q := &mockQueue{queue: qCh}

	handler := TaskHandlerFunc(func(ctx context.Context, task queue.Task, heartbeats chan<- queue.Progress) error {
		defer close(heartbeats)

		require.NotNil(t, task)
		require.Equal(t, "testTask", task.ID)
		require.Equal(t, "testQueue", task.Queue)

		select {
		case <-ctx.Done():
			return nil
		case <-time.NewTimer(time.Second).C:
			// mimic a successfull processing
			heartbeats <- queue.Progress{}
		}

		return nil
	})
	w := NewTaskWorker(q, handler)

	t.Run("worker count starts at 0", func(t *testing.T) {
		require.Equal(t, float64(0), testutil.ToFloat64(TaskQueueMetrics.WorkerGauge))
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
	promLabels := prometheus.Labels{"queue": testTask.Queue, "type": testTask.Type.String()}
	t.Run("calling work inc the worker", func(t *testing.T) {
		require.Equal(t, float64(1), testutil.ToFloat64(TaskQueueMetrics.WorkerGauge))
		require.Equal(t, float64(1), testutil.ToFloat64(TaskQueueMetrics.WorkerWaiting))
		require.Equal(t, float64(0), testutil.ToFloat64(TaskQueueMetrics.WorkerWorkingGauge.With(promLabels)))
		require.Equal(t, float64(0), testutil.ToFloat64(TaskQueueMetrics.WorkerWorking.With(promLabels)))
	})

	qCh <- testTask
	time.Sleep(metricSleep)
	t.Run("active worker count inc after enqueueing a task", func(t *testing.T) {
		require.Equal(t, float64(1), testutil.ToFloat64(TaskQueueMetrics.WorkerWorking.With(promLabels)))
		require.Equal(t, float64(1), testutil.ToFloat64(TaskQueueMetrics.WorkerWorkingGauge.With(promLabels)))
	})

	cancel()
	time.Sleep(metricSleep)
	t.Run("worker count returns to 0 when worker is cancelled", func(t *testing.T) {
		require.Equal(t, float64(0), testutil.ToFloat64(TaskQueueMetrics.WorkerGauge))
	})

}

func Test_WorkerHeartbeatUnknownErrorIsReturned(t *testing.T) {
	defer goleak.VerifyNone(t)

	logrus.SetOutput(ioutil.Discard)
	defer logrus.SetOutput(os.Stdout)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	qCh := make(chan *queue.Task, 1)
	q := &mockQueue{queue: qCh, heartbeatErr: errors.New("can not hearbeat")}

	testTask := &queue.Task{TaskBase: queue.TaskBase{Queue: "testQueue"}, ID: "testTask"}
	qCh <- testTask

	handler := TaskHandlerFunc(func(ctx context.Context, task queue.Task, heartbeats chan<- queue.Progress) error {
		defer close(heartbeats)

		require.NotNil(t, task)
		require.Equal(t, testTask.ID, task.ID)
		require.Equal(t, testTask.Queue, task.Queue)

		select {
		case <-ctx.Done():
			return nil
		case <-time.NewTimer(time.Second).C:
			// mimic a successfull processing
			heartbeats <- queue.Progress{}
		}

		return nil
	})

	w := NewTaskWorker(q, handler)
	err := w.Work(ctx)
	require.EqualError(t, err, "can not hearbeat")
}

func Test_WorkerDequeueErrorIsNotReturned(t *testing.T) {
	defer goleak.VerifyNone(t)

	logrus.SetOutput(ioutil.Discard)
	defer logrus.SetOutput(os.Stdout)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var logs bytes.Buffer
	logrus.SetOutput(&logs)

	qCh := make(chan *queue.Task, 1)
	q := &mockQueue{queue: qCh, dequeueErr: errors.New("can not dequeue")}

	testTask := &queue.Task{TaskBase: queue.TaskBase{Queue: "testQueue"}, ID: "testTask"}
	qCh <- testTask

	handler := TaskHandlerFunc(func(ctx context.Context, task queue.Task, heartbeats chan<- queue.Progress) error {
		defer close(heartbeats)

		require.NotNil(t, task)
		require.Equal(t, testTask.ID, task.ID)
		require.Equal(t, testTask.Queue, task.Queue)

		select {
		case <-ctx.Done():
			return nil
		case <-time.NewTimer(time.Second).C:
			// mimic a successfull processing
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
}

func Test_WorkerFindsFinishedTask(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var logs bytes.Buffer
	logrus.SetOutput(&logs)

	qCh := make(chan *queue.Task, 1)
	testTask := &queue.Task{TaskBase: queue.TaskBase{Queue: "testQueue"}, ID: "testTask"}
	handler := TaskHandlerFunc(func(ctx context.Context, task queue.Task, heartbeats chan<- queue.Progress) error {
		defer close(heartbeats)

		require.NotNil(t, task)
		require.Equal(t, testTask.ID, task.ID)
		require.Equal(t, testTask.Queue, task.Queue)

		select {
		case <-ctx.Done():
			return nil
		case <-time.NewTimer(2 * time.Millisecond).C:
			// mimic a successfull processing
			heartbeats <- queue.Progress{}
		}

		return nil
	})

	allowedFinishedErrors := []error{
		queue.ErrTaskCancelled, queue.ErrTaskFinished, queue.ErrTaskNotFound, queue.ErrTaskNotRunning,
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
}

type mockQueue struct {
	queue        chan *queue.Task
	dequeueErr   error
	heartbeatErr error
	finishErr    error
	failErr      error
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
	return q.heartbeatErr
}

func (q *mockQueue) Finish(ctx context.Context, taskID string, metadata queue.Progress) error {
	return q.finishErr
}

func (q *mockQueue) Fail(ctx context.Context, taskID string, metadata queue.Progress) error {
	return q.failErr
}
