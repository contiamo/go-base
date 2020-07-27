package postgres

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/Masterminds/squirrel"
	dbtest "github.com/contiamo/go-base/pkg/db/test"
	"github.com/contiamo/go-base/pkg/queue"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestSchedule(t *testing.T) {
	logrus.SetOutput(ioutil.Discard)
	defer logrus.SetOutput(os.Stdout)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, db := dbtest.GetDatabase(t)
	defer db.Close()
	require.NoError(t, SetupTables(ctx, db, nil))
	_, err := db.ExecContext(ctx, `ALTER TABLE schedules ADD column test_id uuid;`)
	require.NoError(t, err)

	builder := squirrel.StatementBuilder.
		PlaceholderFormat(squirrel.Dollar).
		RunWith(db)

	cases := []struct {
		name     string
		task     queue.TaskScheduleRequest
		expError string
	}{
		{
			name: "Returns an error when the queue name is empty",
			task: queue.TaskScheduleRequest{
				TaskBase: queue.TaskBase{
					Type: "test",
					Spec: spec,
				},
				CronSchedule: "@weekly",
			},
			expError: queue.ErrTaskQueueNotSpecified.Error(),
		},
		{
			name: "Returns an error when the task type is empty",
			task: queue.TaskScheduleRequest{
				TaskBase: queue.TaskBase{
					Queue: queueID1,
					Spec:  spec,
				},
				CronSchedule: "@weekly",
			},
			expError: queue.ErrTaskTypeNotSpecified.Error(),
		},
		{
			name: "Returns an error when the spec is not valid JSON",
			task: queue.TaskScheduleRequest{
				TaskBase: queue.TaskBase{
					Queue: "queue1",
					Type:  "test",
					Spec:  []byte("invalid"),
				},
				CronSchedule: "@weekly",
			},
			expError: "pq: invalid input syntax for type json",
		},
		{
			name: "Returns no error when a schedule is valid",
			task: queue.TaskScheduleRequest{
				TaskBase: queue.TaskBase{
					Queue: "queue2",
					Type:  "test",
					Spec:  spec,
				},
				CronSchedule: "@weekly",
			},
		},
		{
			name: "Returns error when a cron schedule is not valid",
			task: queue.TaskScheduleRequest{
				TaskBase: queue.TaskBase{
					Queue: "queue2",
					Type:  "test",
					Spec:  spec,
				},
				CronSchedule: "invalid",
			},
			expError: "failed to parse crontab: Expected exactly 5 fields, found 1: invalid",
		},

		{
			name: "Returns no error when a task has no spec",
			task: queue.TaskScheduleRequest{
				TaskBase: queue.TaskBase{
					Queue: "queue2",
					Type:  "test",
				},
				CronSchedule: "@weekly",
			},
		},
		{
			name: "Returns no error when a task has references",
			task: queue.TaskScheduleRequest{
				TaskBase: queue.TaskBase{
					Queue: "queue2",
					Type:  "test",
					Spec:  spec,
				},
				CronSchedule: "@weekly",
				References: queue.References{
					"test_id": uuid.NewV4(),
				},
			},
		},
		{
			name: "Returns error when a task has invalid references",
			task: queue.TaskScheduleRequest{
				TaskBase: queue.TaskBase{
					Queue: "queue2",
					Type:  "test",
					Spec:  spec,
				},
				CronSchedule: "@weekly",
				References: queue.References{
					"invalid": uuid.NewV4(),
				},
			},
			expError: `pq: column "invalid" of relation "schedules" does not exist`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q := NewScheduler(db)
			err := q.Schedule(ctx, builder, tc.task)

			if tc.expError != "" {
				require.Error(t, err)
				require.Equal(t, tc.expError, err.Error())
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestEnsure(t *testing.T) {
	logrus.SetOutput(ioutil.Discard)
	defer logrus.SetOutput(os.Stdout)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, db := dbtest.GetDatabase(t)
	defer db.Close()
	require.NoError(t, SetupTables(ctx, db, nil))
	_, err := db.ExecContext(ctx, `ALTER TABLE schedules ADD column test_id uuid;`)
	require.NoError(t, err)

	task := queue.TaskScheduleRequest{
		TaskBase: queue.TaskBase{
			Queue: "queue2",
			Type:  "test",
			Spec:  spec,
		},
		CronSchedule: "@weekly",
		References: queue.References{
			"test_id": uuid.NewV4(),
		},
	}
	refColumns, refValues := task.References.GetNamesAndValues()
	builder := squirrel.StatementBuilder.
		PlaceholderFormat(squirrel.Dollar).
		RunWith(db)

	_, err = builder.
		Insert("schedules").
		Columns(
			append(
				refColumns,
				"schedule_id",
				"task_queue",
				"task_type",
				"task_spec",
				"cron_schedule",
				"next_execution_time",
			)...,
		).
		Values(
			append(
				refValues,
				uuid.NewV4().String(),
				task.Queue,
				task.Type,
				task.Spec,
				task.CronSchedule,
				time.Now(), // the schedule will enqueue the task immediately
			)...,
		).
		ExecContext(ctx)
	require.NoError(t, err)

	cases := []struct {
		name     string
		task     queue.TaskScheduleRequest
		expError string
	}{
		{
			name: "Returns an error when the queue name is empty",
			task: queue.TaskScheduleRequest{
				TaskBase: queue.TaskBase{
					Type: "test",
					Spec: spec,
				},
				CronSchedule: "@weekly",
			},
			expError: queue.ErrTaskQueueNotSpecified.Error(),
		},
		{
			name: "Returns an error when the task type is empty",
			task: queue.TaskScheduleRequest{
				TaskBase: queue.TaskBase{
					Queue: queueID1,
					Spec:  spec,
				},
				CronSchedule: "@weekly",
			},
			expError: queue.ErrTaskTypeNotSpecified.Error(),
		},
		{
			name: "Returns error when schedule can not be found",
			task: queue.TaskScheduleRequest{
				TaskBase: queue.TaskBase{
					Queue: "queue2",
					Type:  "test",
					Spec:  spec,
				},
				CronSchedule: "@weekly",
				References: queue.References{
					"test_id": uuid.NewV4(),
				},
			},
			expError: queue.ErrNotScheduled.Error(),
		},
		{
			name: "Returns no error when schedule exists in db",
			task: task,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q := NewScheduler(db)
			err := q.EnsureSchedule(ctx, builder, tc.task)

			if tc.expError != "" {
				require.Error(t, err)
				require.Equal(t, tc.expError, err.Error())
				return
			}

			require.NoError(t, err)
		})
	}
}
