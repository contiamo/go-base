package queue

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestSwitchMetricsServiceName(t *testing.T) {
	t.Run("does not panic when try to register something that matches default", func(t *testing.T) {
		require.NotPanics(t, func() {
			SwitchMetricsServiceName(constLabels[serviceKey])
		})
	})

	t.Run("does not panic when try to register something else", func(t *testing.T) {
		require.NotPanics(t, func() {
			SwitchMetricsServiceName("test")
		})
	})

	t.Run("does not panic when try to register something twice", func(t *testing.T) {
		require.NotPanics(t, func() {
			SwitchMetricsServiceName("test")
			SwitchMetricsServiceName("test")
		})
	})
}

func TestQueueMetrics(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	numOfTasks := 2
	qCh := make(chan *Task, numOfTasks)
	rawQ := QueueMock{Queue: qCh}
	q := QueuerWithMetrics(&rawQ)

	testTask := TaskEnqueueRequest{TaskBase: TaskBase{Queue: "testQueue", Type: "tester"}}
	promLabels := prometheus.Labels{"queue": testTask.Queue, "type": testTask.Type.String()}

	t.Run("task count starts at 0", func(t *testing.T) {
		require.Equal(t, float64(0), testutil.ToFloat64(TaskQueueMetrics.TaskCounter.With(promLabels)))
	})

	for i := 0; i < numOfTasks; i++ {
		err := q.Enqueue(ctx, testTask)
		require.NoError(t, err)
	}

	time.Sleep(time.Millisecond)
	t.Run("task count inc after enqueueing a task", func(t *testing.T) {
		// compairing with the rawQ count ensures that the underlying queue was called by our wrapping method
		require.Equal(t, numOfTasks, rawQ.EnqueueCount)
		require.Equal(t, float64(rawQ.EnqueueCount), testutil.ToFloat64(TaskQueueMetrics.TaskCounter.With(promLabels)))
	})

	t.Run("metrics wrapped queue calls the internal queue method", func(t *testing.T) {
		actualTask := <-qCh
		require.Equal(t, testTask.Type, actualTask.Type)
		require.Equal(t, testTask.Queue, actualTask.Queue)
	})
}

func TestSchedulerMetrics(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rawScheduler := SchedulerMock{}
	s := SchedulerWithMetrics(&rawScheduler)

	testTask := TaskScheduleRequest{TaskBase: TaskBase{Queue: "testQueue", Type: "tester"}, CronSchedule: "@daily"}
	promLabels := prometheus.Labels{"queue": testTask.Queue, "type": testTask.Type.String()}

	t.Run("task and error count starts at 0", func(t *testing.T) {
		require.Equal(t, float64(0), testutil.ToFloat64(SchedulerMetrics.ScheduleCounter.With(promLabels)))
		require.Equal(t, float64(0), testutil.ToFloat64(SchedulerMetrics.ErrorCounter.With(promLabels)))
	})

	t.Run("schedule count inc after scheduling a task", func(t *testing.T) {
		err := s.Schedule(ctx, nil, testTask)
		require.NoError(t, err)
		// compairing with the rawQ count ensures that the underlying queue was called by our wrapping method
		time.Sleep(time.Millisecond)
		require.Equal(t, 1, rawScheduler.ScheduleCount)
		require.Equal(t, float64(rawScheduler.ScheduleCount), testutil.ToFloat64(SchedulerMetrics.ScheduleCounter.With(promLabels)))
		require.Equal(t, float64(0), testutil.ToFloat64(SchedulerMetrics.ErrorCounter.With(promLabels)))
	})

	t.Run("schedule count inc after scheduling a task", func(t *testing.T) {
		rawScheduler.ScheduleErr = errors.New("scheduler oops")
		err := s.Schedule(ctx, nil, testTask)
		require.EqualError(t, err, "scheduler oops")

		// compairing with the rawQ count ensures that the underlying queue was called by our wrapping method
		time.Sleep(time.Millisecond)
		require.Equal(t, 2, rawScheduler.ScheduleCount)
		require.Equal(t, float64(rawScheduler.ScheduleCount), testutil.ToFloat64(SchedulerMetrics.ScheduleCounter.With(promLabels)))
		require.Equal(t, float64(1), testutil.ToFloat64(SchedulerMetrics.ErrorCounter.With(promLabels)))
	})
}
