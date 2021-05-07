package postgres

import (
	"context"
	"io/ioutil"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/contiamo/go-base/v3/pkg/config"
	"github.com/contiamo/go-base/v3/pkg/data/managers"
	dbtest "github.com/contiamo/go-base/v3/pkg/db/test"
	"github.com/contiamo/go-base/v3/pkg/queue"
	"github.com/lib/pq"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestFinish(t *testing.T) {
	verifyLeak(t)

	logrus.SetOutput(ioutil.Discard)
	defer logrus.SetOutput(os.Stdout)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	name, db := dbtest.GetDatabase(t)
	defer db.Close()
	require.NoError(t, SetupTables(ctx, db, nil))

	taskID := uuid.NewV4().String()
	connStr := "user=contiamo_test password=localdev sslmode=disable dbname=" + name
	dbListener := pq.NewListener(
		connStr,
		10*time.Second,
		time.Minute,
		func(ev pq.ListenerEventType, err error) {
			if err != nil {
				logrus.Error(err)
			}
		},
	)

	defer dbListener.Close()

	q := NewDequeuer(db, dbListener, config.Queue{
		HeartbeatTTL:  10 * time.Second,
		PollFrequency: 50 * time.Millisecond,
	})

	builder := squirrel.StatementBuilder.
		PlaceholderFormat(squirrel.Dollar).
		RunWith(db)

	_, err := builder.
		Insert("tasks").
		Columns(
			"task_id",
			"queue",
			"type",
			"spec",
			"progress",
			"status",
		).
		Values(
			taskID,
			queueID1,
			"test",
			emptyJSON,
			emptyJSON,
			queue.Running,
		).
		ExecContext(ctx)

	t.Run("sets the progress, finished status and finished timestamp", func(t *testing.T) {
		err = q.Finish(ctx, taskID, progress)
		require.NoError(t, err)
		dbtest.EqualCount(t, db, 1, "tasks", squirrel.And{
			squirrel.Eq{
				"task_id":  taskID,
				"status":   queue.Finished,
				"progress": progress,
			},
			squirrel.NotEq{"finished_at": nil},
		})
	})
}

func TestFail(t *testing.T) {
	verifyLeak(t)

	logrus.SetOutput(ioutil.Discard)
	defer logrus.SetOutput(os.Stdout)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	name, db := dbtest.GetDatabase(t)
	defer db.Close()
	require.NoError(t, SetupTables(ctx, db, nil))

	taskID := uuid.NewV4().String()

	connStr := "user=contiamo_test password=localdev sslmode=disable dbname=" + name
	dbListener := pq.NewListener(
		connStr,
		10*time.Second,
		time.Minute,
		func(ev pq.ListenerEventType, err error) {
			if err != nil {
				logrus.Error(err)
			}
		},
	)
	defer dbListener.Close()

	q := NewDequeuer(db, dbListener, config.Queue{
		HeartbeatTTL:  10 * time.Second,
		PollFrequency: 50 * time.Millisecond,
	})

	builder := squirrel.StatementBuilder.
		PlaceholderFormat(squirrel.Dollar).
		RunWith(db)

	_, err := builder.
		Insert("tasks").
		Columns(
			"task_id",
			"queue",
			"type",
			"spec",
			"progress",
			"status",
		).
		Values(
			taskID,
			queueID1,
			"test",
			emptyJSON,
			emptyJSON,
			queue.Running,
		).
		ExecContext(ctx)

	t.Run("sets the progress, failed status and finished timestamp", func(t *testing.T) {
		err = q.Fail(ctx, taskID, progress)
		require.NoError(t, err)
		dbtest.EqualCount(t, db, 1, "tasks", squirrel.And{
			squirrel.Eq{
				"task_id":  taskID,
				"status":   queue.Failed,
				"progress": progress,
			},
			squirrel.NotEq{"finished_at": nil},
		})
	})
}

func TestHeartbeat(t *testing.T) {
	verifyLeak(t)

	logrus.SetOutput(ioutil.Discard)
	defer logrus.SetOutput(os.Stdout)

	waitingUUID := uuid.NewV4().String()
	runningUUID := uuid.NewV4().String()
	finishedUUID := uuid.NewV4().String()
	cancelledUUID := uuid.NewV4().String()
	failedUUID := uuid.NewV4().String()

	cases := []struct {
		name     string
		taskID   string
		progress queue.Progress
		expError string
	}{
		{
			name:     "returns an error when the task is not found",
			taskID:   uuid.NewV4().String(),
			progress: progress,
			expError: queue.ErrTaskNotFound.Error(),
		},
		{
			name:     "update un-finished task",
			taskID:   runningUUID,
			progress: progress,
		},
		{
			name:     "update task to finished",
			taskID:   runningUUID,
			progress: progress,
		},
		{
			name:     "update finished task throws an error",
			taskID:   finishedUUID,
			progress: progress,
			expError: queue.ErrTaskFinished.Error(),
		},
		{
			name:     "update canceled task throws an error",
			taskID:   cancelledUUID,
			progress: progress,
			expError: queue.ErrTaskCancelled.Error(),
		},
		{
			name:     "update waiting task throws an error",
			taskID:   waitingUUID,
			progress: progress,
			expError: queue.ErrTaskNotRunning.Error(),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	name, db := dbtest.GetDatabase(t)
	defer db.Close()
	require.NoError(t, SetupTables(ctx, db, nil))

	connStr := "user=contiamo_test password=localdev sslmode=disable dbname=" + name
	dbListener := pq.NewListener(
		connStr,
		10*time.Second,
		time.Minute,
		func(ev pq.ListenerEventType, err error) {
			if err != nil {
				logrus.Error(err)
			}
		},
	)
	defer dbListener.Close()

	q := NewDequeuer(db, dbListener, config.Queue{
		HeartbeatTTL:  10 * time.Second,
		PollFrequency: 50 * time.Millisecond,
	})

	builder := squirrel.StatementBuilder.
		PlaceholderFormat(squirrel.Dollar).
		RunWith(db)

	_, err := builder.
		Insert("tasks").
		Columns(
			"task_id",
			"queue",
			"type",
			"spec",
			"progress",
			"status",
		).
		Values(
			runningUUID,
			queueID1,
			"test",
			emptyJSON,
			emptyJSON,
			queue.Running,
		).
		Values(
			waitingUUID,
			queueID1,
			"test",
			emptyJSON,
			emptyJSON,
			queue.Waiting,
		).
		Values(
			finishedUUID,
			queueID2,
			"test",
			emptyJSON,
			emptyJSON,
			queue.Finished,
		).
		Values(
			cancelledUUID,
			queueID2,
			"test",
			emptyJSON,
			emptyJSON,
			queue.Cancelled,
		).
		Values(
			failedUUID,
			queueID2,
			"test",
			emptyJSON,
			emptyJSON,
			queue.Running,
		).
		ExecContext(ctx)

	require.NoError(t, err)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := q.Heartbeat(ctx, tc.taskID, tc.progress)

			if tc.expError != "" {
				require.Error(t, err)
				require.Equal(t, tc.expError, err.Error())
				return
			}

			require.NoError(t, err)

			var (
				beat     *time.Time
				status   string
				progress []byte
			)
			err = managers.NewBaseManager(db, "").GetQueryBuilder().
				Select(
					"last_heartbeat_at",
					"status",
					"progress",
				).
				From("tasks").
				Where("task_id = ?", tc.taskID).
				QueryRow().
				Scan(&beat, &status, &progress)

			require.NoError(t, err)
			require.NotNil(t, beat)
			require.Equal(t, string(queue.Running), status)
			require.Equal(t, string(tc.progress), string(progress))
		})
	}
}

func TestDequeueTicker(t *testing.T) {
	verifyLeak(t)

	logrus.SetOutput(ioutil.Discard)
	defer logrus.SetOutput(os.Stdout)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tests := []struct {
		name  string
		queue string
		wait  time.Duration // pause before inserting a task into the db
	}{
		{
			name:  "available task no blocking",
			queue: queueID1,
			wait:  0,
		},
		{
			name:  "available task blocking",
			queue: queueID2,
			wait:  150 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, db := dbtest.GetDatabase(t)
			defer db.Close()
			require.NoError(t, SetupTables(ctx, db, nil))

			connStr := "user=contiamo_test password=localdev sslmode=disable dbname=" + name
			dbListener := pq.NewListener(
				connStr,
				10*time.Second,
				time.Minute,
				func(ev pq.ListenerEventType, err error) {
					if err != nil {
						logrus.Error(err)
					}
				},
			)
			defer dbListener.Close()

			q := NewDequeuer(db, dbListener, config.Queue{
				HeartbeatTTL:  10 * time.Second,
				PollFrequency: 50 * time.Millisecond,
			})

			go func() {
				time.Sleep(tt.wait)
				builder := squirrel.StatementBuilder.
					PlaceholderFormat(squirrel.Dollar).
					RunWith(db)

				_, err := builder.Insert("tasks").
					Columns(
						"task_id",
						"queue",
						"type",
						"spec",
						"progress",
						"status",
						"created_at",
					).
					Values(
						uuid.NewV4(),
						tt.queue,
						"test",
						emptyJSON,
						emptyJSON,
						queue.Waiting,
						time.Now(),
					).
					ExecContext(ctx)

				require.NoError(t, err)
			}()

			// to ensure that we have the insert in the db, wait for 20 ms before calling Dequeue
			time.Sleep(20 * time.Millisecond)

			task, err := q.Dequeue(ctx, tt.queue)
			require.NoError(t, err)
			require.NotNil(t, task)
		})
	}
}

func TestDequeue(t *testing.T) {
	verifyLeak(t)

	logrus.SetOutput(ioutil.Discard)
	defer logrus.SetOutput(os.Stdout)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	now := time.Now()
	old := time.Now().Add(-10 * time.Minute)

	cases := []struct {
		name     string
		queue    string
		tasks    []queue.Task
		expError bool
		expTask  bool
	}{
		{
			name:  "queue has no tasks in db",
			queue: queueID1,
		},
		{
			name:    "queue has one task waiting",
			queue:   queueID1,
			tasks:   []queue.Task{{}},
			expTask: true,
		},
		{
			name:  "queue has one working task and no others available",
			queue: queueID1,
			tasks: []queue.Task{{StartedAt: &now, LastHeartbeatAt: &now}},
		},

		{
			name:  "queue has one working task and one available",
			queue: queueID1,
			tasks: []queue.Task{
				{StartedAt: &now, LastHeartbeatAt: &now},
				{},
			},
		},
		{
			name:    "queue has one failed task",
			queue:   queueID1,
			tasks:   []queue.Task{{StartedAt: &old, LastHeartbeatAt: &old}},
			expTask: true,
		},
		{
			name:  "queue has one finished task",
			queue: queueID1,
			tasks: []queue.Task{{StartedAt: &old, LastHeartbeatAt: &old, FinishedAt: &old}},
		},
		{
			name:  "queue has one finished task and one running task",
			queue: queueID1,
			tasks: []queue.Task{
				{StartedAt: &old, LastHeartbeatAt: &old, FinishedAt: &old},
				{StartedAt: &now, LastHeartbeatAt: &now},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			name, db := dbtest.GetDatabase(t)
			defer db.Close()
			require.NoError(t, SetupTables(ctx, db, nil))

			connStr := "user=contiamo_test password=localdev sslmode=disable dbname=" + name
			dbListener := pq.NewListener(
				connStr,
				10*time.Second,
				time.Minute,
				func(ev pq.ListenerEventType, err error) {
					if err != nil {
						logrus.Error(err)
					}
				},
			)
			defer dbListener.Close()

			q := NewDequeuer(db, dbListener, config.Queue{
				HeartbeatTTL:  10 * time.Second,
				PollFrequency: 50 * time.Millisecond,
			})

			builder := squirrel.StatementBuilder.
				PlaceholderFormat(squirrel.Dollar).
				RunWith(db)

			for _, task := range tc.tasks {
				_, err := builder.Insert("tasks").
					Columns(
						"task_id",
						"queue",
						"type",
						"spec",
						"progress",
						"status",
						"started_at",
						"last_heartbeat_at",
						"finished_at").
					Values(
						uuid.NewV4(),
						tc.queue,
						"test",
						emptyJSON,
						emptyJSON,
						queue.Waiting,
						task.StartedAt,
						task.LastHeartbeatAt,
						task.FinishedAt).
					ExecContext(ctx)

				require.NoError(t, err)
			}

			task, err := q.(*dequeuer).attemptDequeue(ctx, tc.queue)
			if tc.expTask {
				require.NotNil(t, task)
			} else {
				require.Nil(t, task)
			}
			if tc.expError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestProcessableQueues(t *testing.T) {
	verifyLeak(t)

	tests := []struct {
		name      string
		inflight  []string
		potential []string
		wanted    []string
	}{
		{
			name:      "empty inflight; empty potential",
			inflight:  nil,
			potential: nil,
			wanted:    nil,
		},
		{
			name:      "empty inflight; some potential",
			inflight:  nil,
			potential: []string{"a", "b"},
			wanted:    []string{"a", "b"},
		},
		{
			name:      "some inflight; empty potential",
			inflight:  []string{"a", "b"},
			potential: nil,
			wanted:    nil,
		},
		{
			name:      "same inflight and potential",
			inflight:  []string{"a", "b"},
			potential: []string{"a", "b"},
			wanted:    nil,
		},
		{
			name:      "some similar elements in inflight and potential",
			inflight:  []string{"a", "b"},
			potential: []string{"a", "b", "c", "d"},
			wanted:    []string{"c", "d"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processableQueues(tt.inflight, tt.potential)
			require.Equal(t, tt.wanted, result)
		})
	}
}

func TestQueueList(t *testing.T) {
	verifyLeak(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	qs := []string{
		queueID1,
		queueID2,
		queueID3,
	}
	sort.Strings(qs)

	tests := []struct {
		name    string
		db      string
		queues  []string
		where   squirrel.Sqlizer
		expList []string
	}{
		{
			name: "no queues",
			db:   "postgres_queue_queue_list_no_queues",
		},
		{
			name:    "queues; no where statement",
			db:      "postgres_queue_queue_list_no_where_stmt",
			queues:  qs,
			expList: qs,
		},
		{
			name:    "queues; where statement",
			db:      "postgres_queue_queue_list_where_stmt",
			queues:  qs,
			where:   squirrel.Eq{"queue": qs[0]},
			expList: []string{qs[0]},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, db := dbtest.GetDatabase(t)
			defer db.Close()
			require.NoError(t, SetupTables(ctx, db, nil))

			q := NewDequeuer(
				db,
				nil, // listener is not needed for this test
				config.Queue{
					HeartbeatTTL:  10 * time.Second,
					PollFrequency: 50 * time.Millisecond,
				})

			pd, ok := q.(*dequeuer)
			require.True(t, ok)

			for _, queueName := range tt.queues {
				_, err := pd.GetQueryBuilder().
					Insert("tasks").
					Columns(
						"task_id",
						"queue",
						"type",
						"spec",
						"progress",
						"status",
					).
					Values(
						uuid.NewV4(),
						queueName,
						"test",
						emptyJSON,
						emptyJSON,
						queue.Waiting,
					).
					ExecContext(ctx)

				require.NoError(t, err)
			}

			qList, err := pd.generateQueueList(ctx, tt.where)
			sort.Strings(qList)
			require.NoError(t, err)
			require.Equal(t, tt.expList, qList)
		})
	}
}
