package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/contiamo/go-base/pkg/queue/handlers"
	"go.uber.org/goleak"

	"github.com/Masterminds/squirrel"
	dbtest "github.com/contiamo/go-base/pkg/db/test"
	"github.com/contiamo/go-base/pkg/queue"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestRetentionHandler(t *testing.T) {
	defer goleak.VerifyNone(t)

	logrus.SetOutput(ioutil.Discard)
	defer logrus.SetOutput(os.Stdout)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, db := dbtest.GetDatabase(t)
	defer db.Close()
	require.NoError(t, SetupTables(ctx, db, nil))

	// Start test setup
	now := time.Now()
	fiveMinutesAgo := now.Add(-5 * time.Minute)
	tenMinutesAgo := now.Add(-10 * time.Minute)
	twoWeeksAgo := now.Add(-14 * 24 * time.Hour)

	waitingTask := queue.Task{
		TaskBase: queue.TaskBase{
			Queue: "standard",
			Type:  queue.TaskType("simple"),
			Spec:  emptyJSON,
		},
		ID:        uuid.NewV4().String(),
		CreatedAt: now.Add(-5 * time.Minute),
		Status:    queue.Waiting,
	}
	require.NoError(t, insertTestTask(ctx, db, &waitingTask))

	runningTask := queue.Task{
		TaskBase: queue.TaskBase{
			Queue: "standard",
			Type:  queue.TaskType("simple"),
			Spec:  emptyJSON,
		},
		ID:              uuid.NewV4().String(),
		CreatedAt:       now.Add(-5 * time.Minute),
		StartedAt:       &now,
		LastHeartbeatAt: &now,
		Status:          queue.Running,
	}
	require.NoError(t, insertTestTask(ctx, db, &runningTask))

	finishedTask := queue.Task{
		TaskBase: queue.TaskBase{
			Queue: "standard",
			Type:  queue.TaskType("simple"),
			Spec:  emptyJSON,
		},
		ID:              uuid.NewV4().String(),
		CreatedAt:       now.Add(-60 * time.Minute),
		StartedAt:       &tenMinutesAgo,
		UpdatedAt:       fiveMinutesAgo,
		FinishedAt:      &fiveMinutesAgo,
		LastHeartbeatAt: &fiveMinutesAgo,
		Status:          queue.Finished,
	}
	require.NoError(t, insertTestTask(ctx, db, &finishedTask))

	twoWeeksAgoPlus10 := twoWeeksAgo.Add(10 * time.Minute)
	toBeRemoved := queue.Task{
		TaskBase: queue.TaskBase{
			Queue: "standard",
			Type:  queue.TaskType("simple"),
			Spec:  emptyJSON,
		},
		ID:              uuid.NewV4().String(),
		CreatedAt:       twoWeeksAgo.Add(-10 * time.Minute),
		StartedAt:       &twoWeeksAgo,
		UpdatedAt:       twoWeeksAgoPlus10,
		FinishedAt:      &twoWeeksAgoPlus10,
		LastHeartbeatAt: &twoWeeksAgoPlus10,
		Status:          queue.Finished,
	}
	require.NoError(t, insertTestTask(ctx, db, &toBeRemoved))
	// Test db is now prepared

	// Now test the handler
	sevenDays := 7 * 24 * time.Hour
	spec := createRetentionSpec("standard", "", queue.Finished, sevenDays)
	specBytes, err := json.Marshal(spec)
	require.NoError(t, err)

	retentionTask := queue.Task{
		TaskBase: queue.TaskBase{
			Queue: MaintenanceTaskQueue,
			Type:  RetentionTask,
			Spec:  specBytes,
		},
		ID:        uuid.NewV4().String(),
		CreatedAt: now,
	}

	heartbeats := make(chan queue.Progress, 10)

	seenBeats := []handlers.SQLTaskProgress{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for hb := range heartbeats {
			progress := handlers.SQLTaskProgress{}
			err = json.Unmarshal(hb, &progress)
			require.NoError(t, err, "can't parse progress")
			seenBeats = append(seenBeats, progress)
		}
	}()

	handler := NewRetentionHandler(db)
	err = handler.Process(ctx, retentionTask, heartbeats)
	require.NoError(t, err, "processing error")

	dbtest.EqualCount(t, db, 0, TasksTable, squirrel.Eq{
		"task_id": toBeRemoved.ID,
	}, "found toBeRemoved when it should be deleted")
	dbtest.EqualCount(t, db, 1, TasksTable, squirrel.Eq{
		"task_id": finishedTask.ID,
	}, "can not find finishedTask")
	dbtest.EqualCount(t, db, 1, TasksTable, squirrel.Eq{
		"task_id": runningTask.ID,
	}, "can not find runningTask")
	dbtest.EqualCount(t, db, 1, TasksTable, squirrel.Eq{
		"task_id": waitingTask.ID,
	}, "can not find waitingTask")

	dbtest.EqualCount(t, db, 3, TasksTable, nil, "incorrect task count")

	<-done

	require.Equal(t, handlers.SQLTaskProgress{}, seenBeats[0])

	lastBeat := seenBeats[len(seenBeats)-1]

	require.Nil(t, lastBeat.ErrorMessage)
	require.NotNil(t, lastBeat.Duration)
	require.NotNil(t, lastBeat.RowsAffected)
	require.Equal(t, int64(1), *lastBeat.RowsAffected, "%+v", seenBeats)
}

func insertTestTask(ctx context.Context, db *sql.DB, task *queue.Task) error {
	_, err := squirrel.StatementBuilder.
		PlaceholderFormat(squirrel.Dollar).
		RunWith(db).
		Insert("tasks").
		Columns(
			"task_id",
			"queue",
			"type",
			"spec",
			"status",
			"progress",
			"created_at",
			"updated_at",
			"started_at",
			"finished_at",
			"last_heartbeat_at",
		).
		Values(
			task.ID,
			task.Queue,
			task.Type,
			task.Spec,
			task.Status,
			emptyJSON,
			task.CreatedAt,
			task.UpdatedAt,
			task.StartedAt,
			task.FinishedAt,
			task.LastHeartbeatAt,
		).
		ExecContext(ctx)
	return err
}

func TestAssertRetentionSchedule(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, db := dbtest.GetDatabase(t)
	defer db.Close()
	require.NoError(t, SetupTables(ctx, db, nil))

	cases := []struct {
		name        string
		queueName   string
		taskType    queue.TaskType
		status      queue.TaskStatus
		age         time.Duration
		expectedSQL string
	}{
		{
			name:        "retention wtih all filter values specified is successful",
			queueName:   "basic",
			taskType:    queue.TaskType("throw-away"),
			status:      queue.Finished,
			age:         time.Minute,
			expectedSQL: `DELETE FROM tasks WHERE status = 'finished' AND finished_at <= now() - interval '1.000000 minutes' AND queue = 'basic' AND type = 'throw-away'`,
		},
		{
			name:        "one week retention without any extra filters is successful",
			queueName:   "advanced",
			taskType:    queue.TaskType("throw-away-2"),
			status:      queue.Finished,
			age:         7 * 24 * time.Hour,
			expectedSQL: `DELETE FROM tasks WHERE status = 'finished' AND finished_at <= now() - interval '10080.000000 minutes' AND queue = 'advanced' AND type = 'throw-away-2'`,
		},
		{
			name:        "one month retention without any extra filters is successful",
			queueName:   "super",
			taskType:    queue.TaskType("throw-away-3"),
			status:      queue.Finished,
			age:         30 * 24 * time.Hour,
			expectedSQL: `DELETE FROM tasks WHERE status = 'finished' AND finished_at <= now() - interval '43200.000000 minutes' AND queue = 'super' AND type = 'throw-away-3'`,
		},
		{
			name:        "retention for _any_ finished task, without queue or task type restriction",
			status:      queue.Finished,
			age:         time.Minute,
			expectedSQL: `DELETE FROM tasks WHERE status = 'finished' AND finished_at <= now() - interval '1.000000 minutes'`,
		},
		{
			name:        "retention for _any_ failed task in a specific queue",
			queueName:   "justme",
			status:      queue.Failed,
			age:         time.Minute,
			expectedSQL: `DELETE FROM tasks WHERE status = 'failed' AND finished_at <= now() - interval '1.000000 minutes' AND queue = 'justme'`,
		},
		{
			name:        "retention based on task type and status",
			taskType:    "justme",
			status:      queue.Failed,
			age:         time.Minute,
			expectedSQL: `DELETE FROM tasks WHERE status = 'failed' AND finished_at <= now() - interval '1.000000 minutes' AND type = 'justme'`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			err := AssertRetentionSchedule(ctx, db, tc.queueName, tc.taskType, tc.status, tc.age)
			require.NoError(t, err, "unexpected assert error")

			dbtest.EqualCount(t, db, 1, "schedules", squirrel.And{
				squirrel.Eq{
					"task_queue":              MaintenanceTaskQueue,
					"task_type":               RetentionTask,
					"task_spec->>'queueName'": tc.queueName,
					"task_spec->>'taskType'":  tc.taskType,
					"task_spec->>'status'":    tc.status,
					"task_spec->>'age'":       tc.age,
				},
				squirrel.Expr(`coalesce(task_spec->>'sql','') != ''`),
				squirrel.Expr(`cron_schedule ~ '\d{1,2} * * * *'`),
			}, "unique retention task not found")
		})
	}

	t.Run("can reschedule a retention task with a new age policy", func(t *testing.T) {
		queueName := "super_important"
		taskType := queue.TaskType("update-test")
		status := queue.Finished

		err := AssertRetentionSchedule(ctx, db, queueName, taskType, status, 5*time.Minute)
		require.NoError(t, err, "unexpected assert error")

		dbtest.EqualCount(t, db, 1, "schedules", squirrel.And{
			squirrel.Eq{
				"task_queue":              MaintenanceTaskQueue,
				"task_type":               RetentionTask,
				"task_spec->>'queueName'": queueName,
				"task_spec->>'taskType'":  taskType,
				"task_spec->>'status'":    status,
				"task_spec->>'age'":       5 * time.Minute,
			},
			squirrel.Expr(`coalesce(task_spec->>'sql','') != ''`),
			squirrel.Expr(`cron_schedule ~ '\d{1,2} * * * *'`),
		}, "initial retention task not found")

		err = AssertRetentionSchedule(ctx, db, queueName, taskType, status, 10*time.Minute)
		require.NoError(t, err, "unexpected assert error")

		dbtest.EqualCount(t, db, 1, "schedules", squirrel.And{
			squirrel.Eq{
				"task_queue":              MaintenanceTaskQueue,
				"task_type":               RetentionTask,
				"task_spec->>'queueName'": queueName,
				"task_spec->>'taskType'":  taskType,
				"task_spec->>'status'":    status,
				"task_spec->>'age'":       10 * time.Minute,
			},
			squirrel.Expr(`coalesce(task_spec->>'sql','') != ''`),
			squirrel.Expr(`cron_schedule ~ '\d{1,2} * * * *'`),
		}, "updated task with new age parameter not found")
	})
}
