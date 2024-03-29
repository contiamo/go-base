package workers

import (
	"context"
	"testing"
	"time"

	"github.com/Masterminds/squirrel"

	dbtest "github.com/contiamo/go-base/v4/pkg/db/test"
	"github.com/contiamo/go-base/v4/pkg/queue"
	"github.com/contiamo/go-base/v4/pkg/queue/postgres"
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
		require.NoError(t, postgres.SetupTables(ctx, db, nil))

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
				"",                      // empty value is a one-time job
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

		time.Sleep(time.Second)
		err = w.scheduleTask(ctx)
		require.NoError(t, err)

		time.Sleep(time.Second)
		err = w.scheduleTask(ctx)
		require.Equal(t, err, ErrScheduleQueueIsEmpty)

		time.Sleep(time.Second)
		err = w.scheduleTask(ctx)
		require.Equal(t, err, ErrScheduleQueueIsEmpty)

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
		dbtest.EqualCount(t, db, 0, "schedules", squirrel.LtOrEq{
			"next_execution_time": now,
		})

		dbtest.EqualCount(t, db, 1, "schedules", squirrel.And{
			squirrel.Gt{"next_execution_time": now},
			squirrel.Eq{"schedule_id": scheduleID1},
		})

		dbtest.EqualCount(t, db, 1, "schedules", squirrel.Eq{
			"next_execution_time": nil,
		})

		// these should stay zero because they are not incremented in the scheduleTask function
		require.Equal(t, float64(0), testutil.ToFloat64(queue.ScheduleWorkerMetrics.ActiveGauge))
		require.Equal(t, float64(0), testutil.ToFloat64(queue.ScheduleWorkerMetrics.WorkingGauge))
		require.Equal(t, float64(0), testutil.ToFloat64(queue.ScheduleWorkerMetrics.WaitingGauge))
		require.Equal(t, float64(0), testutil.ToFloat64(queue.ScheduleWorkerMetrics.DequeueingGauge))
		require.Equal(t, float64(0), testutil.ToFloat64(queue.ScheduleWorkerMetrics.ErrorsCounter))
		// this should increase because the task is scheduled
		promLabels1 := prometheus.Labels{"queue": task1.Queue, "type": task1.Type.String()}
		require.Equal(t, float64(1), testutil.ToFloat64(queue.ScheduleWorkerMetrics.ProcessedCounter.With(promLabels1)))
		promLabels2 := prometheus.Labels{"queue": task2.Queue, "type": task2.Type.String()}
		require.Equal(t, float64(1), testutil.ToFloat64(queue.ScheduleWorkerMetrics.ProcessedCounter.With(promLabels2)))
	})

	t.Run("Returns ErrScheduleQueueIsEmpty if there is no task to schedule", func(t *testing.T) {
		_, db := dbtest.GetDatabase(t)
		defer db.Close()
		require.NoError(t, postgres.SetupTables(ctx, db, nil))

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
	require.Equal(t, float64(0), testutil.ToFloat64(queue.ScheduleWorkerMetrics.ActiveGauge))
	require.Equal(t, float64(0), testutil.ToFloat64(queue.ScheduleWorkerMetrics.WorkingGauge))
	require.Equal(t, float64(0), testutil.ToFloat64(queue.ScheduleWorkerMetrics.WaitingGauge))
	require.Equal(t, float64(0), testutil.ToFloat64(queue.ScheduleWorkerMetrics.DequeueingGauge))

	_, db := dbtest.GetDatabase(t)
	defer db.Close()
	require.NoError(t, postgres.SetupTables(context.TODO(), db, nil))

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
	require.Equal(t, float64(0), testutil.ToFloat64(queue.ScheduleWorkerMetrics.ActiveGauge))
	require.Equal(t, float64(0), testutil.ToFloat64(queue.ScheduleWorkerMetrics.WorkingGauge))
	require.Equal(t, float64(0), testutil.ToFloat64(queue.ScheduleWorkerMetrics.WaitingGauge))
	require.Equal(t, float64(0), testutil.ToFloat64(queue.ScheduleWorkerMetrics.DequeueingGauge))
	// because context interruption is an error
	require.Equal(t, float64(0), testutil.ToFloat64(queue.ScheduleWorkerMetrics.ErrorsCounter))
	// these counters should increase
	cancel()
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
