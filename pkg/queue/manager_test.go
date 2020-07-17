package queue

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestQueueMetrics(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	SetupTaskQueueMetrics("test")
	SetupSchedulerMetrics("test")

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
