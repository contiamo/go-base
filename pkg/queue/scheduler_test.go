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
