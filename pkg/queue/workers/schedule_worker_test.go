package workers

import (
	"context"
	"testing"
	"time"

	"github.com/Masterminds/squirrel"

	dbtest "github.com/contiamo/go-base/pkg/db/test"
	"github.com/contiamo/go-base/pkg/queue"
	"github.com/contiamo/go-base/pkg/queue/postgres"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestScheduleTask(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Takes a schedule from the queue and enqueues tasks", func(t *testing.T) {
		_, db := dbtest.GetDatabase(t)
		defer db.Close()
		require.NoError(t, postgres.Setup(ctx, db, nil))

		scheduleID1 := uuid.NewV4().String()
		scheduleID2 := uuid.NewV4().String()
		now := time.Now().Add(-1 * time.Minute).Round(time.Second)
		taskQueue1 := "queue1"
		taskQueue2 := "queue2"
		taskType := "type"
		taskSpec1 := []byte(`{"field": "value1"}`)
		taskSpec2 := []byte(`{"field": "value2"}`)

		builder := squirrel.StatementBuilder.
			PlaceholderFormat(squirrel.Dollar).
			RunWith(db)

		_, err := builder.
			Insert("schedules").
			Columns(
				"schedule_id",
				"task_queue",
				"task_type",
				"task_spec",
				"cron_schedule",
				"next_execution_time",
				"created_at",
				"updated_at",
			).
			Values(
				scheduleID1,
				taskQueue1,
				taskType,
				taskSpec1,
				"@weekly",
				now.Add(-2*time.Minute), // this must be executed instantly
				now.Add(-1*time.Minute),
				now.Add(-1*time.Minute),
			).
			Values(
				scheduleID2,
				taskQueue2,
				taskType,
				taskSpec2,
				"",
				now.Add(-1*time.Minute), // this must be executed after the first
				now.Add(-1*time.Minute),
				now.Add(-1*time.Minute),
			).
			ExecContext(ctx)
		require.NoError(t, err)

		qm := &queueMock{}
		w := newScheduleWorker(db, qm, time.Second)
		err = w.scheduleTask(ctx)
		require.NoError(t, err)
		err = w.scheduleTask(ctx)
		require.NoError(t, err)

		require.Len(t, qm.q, 2)

		task1 := qm.q[0]
		require.Equal(t, taskQueue1, task1.Queue)
		require.Equal(t, taskType, task1.Type.String())
		require.Equal(t, string(taskSpec1), string(task1.Spec))

		task2 := qm.q[1]
		require.Equal(t, taskQueue2, task2.Queue)
		require.Equal(t, taskType, task2.Type.String())
		require.Equal(t, string(taskSpec2), string(task2.Spec))

		// checking that the execution time has changed
		dbtest.EqualCount(t, db, 0, "schedules", squirrel.Eq{
			"next_execution_time": now,
		})

		// these should stay zero because they are not incremented in the scheduleTask function
		require.Equal(t, float64(0), testutil.ToFloat64(TaskSchedulingMetrics.WorkerGauge))
		require.Equal(t, float64(0), testutil.ToFloat64(TaskSchedulingMetrics.WorkerWorkingGauge))
		require.Equal(t, float64(0), testutil.ToFloat64(TaskSchedulingMetrics.WorkerWaiting))
		require.Equal(t, float64(0), testutil.ToFloat64(TaskSchedulingMetrics.WorkerWorking))
		require.Equal(t, float64(0), testutil.ToFloat64(TaskSchedulingMetrics.WorkerErrors))
		// this should increase because the task is scheduled
		promLabels1 := prometheus.Labels{"queue": task1.Queue, "type": task1.Type.String()}
		require.Equal(t, float64(1), testutil.ToFloat64(TaskSchedulingMetrics.WorkerTaskScheduled.With(promLabels1)))
		promLabels2 := prometheus.Labels{"queue": task2.Queue, "type": task2.Type.String()}
		require.Equal(t, float64(1), testutil.ToFloat64(TaskSchedulingMetrics.WorkerTaskScheduled.With(promLabels2)))
	})

	t.Run("Returns ErrScheduleQueueIsEmpty if there is no task to schedule", func(t *testing.T) {
		_, db := dbtest.GetDatabase(t)
		defer db.Close()
		require.NoError(t, postgres.Setup(ctx, db, nil))

		qm := &queueMock{}
		w := newScheduleWorker(db, qm, time.Second)
		err := w.scheduleTask(ctx)
		require.Error(t, err)
		require.Equal(t, ErrScheduleQueueIsEmpty, err)
		require.Len(t, qm.q, 0)
	})
}

func TestMetrics(t *testing.T) {
	defer goleak.VerifyNone(t)

	// these should be zero in the beginning of the test
	require.Equal(t, float64(0), testutil.ToFloat64(TaskSchedulingMetrics.WorkerGauge))
	require.Equal(t, float64(0), testutil.ToFloat64(TaskSchedulingMetrics.WorkerWorkingGauge))
	require.Equal(t, float64(0), testutil.ToFloat64(TaskSchedulingMetrics.WorkerWaiting))
	require.Equal(t, float64(0), testutil.ToFloat64(TaskSchedulingMetrics.WorkerWorking))

	_, db := dbtest.GetDatabase(t)
	defer db.Close()
	require.NoError(t, postgres.Setup(context.TODO(), db, nil))

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// the interval must be greater than the context cancellation time
	w := newScheduleWorker(db, &queueMock{}, 750*time.Millisecond)
	go func() {
		_ = w.Work(ctx)
	}()
	<-ctx.Done()
	<-time.After(100 * time.Millisecond) // let the work to shutdown

	// these gauges should get back to zero
	require.Equal(t, float64(0), testutil.ToFloat64(TaskSchedulingMetrics.WorkerGauge))
	require.Equal(t, float64(0), testutil.ToFloat64(TaskSchedulingMetrics.WorkerWorkingGauge))
	// because context interruption is an error
	require.Equal(t, float64(0), testutil.ToFloat64(TaskSchedulingMetrics.WorkerErrors))
	// these counters should increase
	// the first iteration has time to succeed, the second waiting is interrupted
	require.Equal(t, float64(2), testutil.ToFloat64(TaskSchedulingMetrics.WorkerWaiting))
	// the first iteration starts immediately and the second after the interval tick
	require.Equal(t, float64(2), testutil.ToFloat64(TaskSchedulingMetrics.WorkerWorking))
}

type queueMock struct {
	enqueue error
	q       []queue.TaskEnqueueRequest
}

func (q *queueMock) Enqueue(ctx context.Context, task queue.TaskEnqueueRequest) error {
	q.q = append(q.q, task)
	return q.enqueue
}

func (q *queueMock) Dequeue(ctx context.Context, queue ...string) (*queue.Task, error) {
	return nil, nil
}

func (q *queueMock) Heartbeat(ctx context.Context, taskID string, progress queue.Progress) error {
	return nil
}

func (q *queueMock) Finish(ctx context.Context, taskID string, progress queue.Progress) error {
	return nil
}
