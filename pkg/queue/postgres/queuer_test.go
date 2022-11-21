package postgres

import (
	"context"
	"io"
	"os"
	"testing"
	"time"

	dbtest "github.com/contiamo/go-base/v4/pkg/db/test"
	"github.com/contiamo/go-base/v4/pkg/queue"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestEnqueue(t *testing.T) {
	verifyLeak(t)

	logrus.SetOutput(io.Discard)
	defer logrus.SetOutput(os.Stdout)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, db := dbtest.GetDatabase(t)
	defer db.Close()
	require.NoError(t, SetupTables(ctx, db, nil))
	_, err := db.ExecContext(ctx, `ALTER TABLE tasks ADD column test_id uuid;`)
	require.NoError(t, err)

	cases := []struct {
		name     string
		task     queue.TaskEnqueueRequest
		expError string
	}{
		{
			name:     "Returns an error when the queue name is empty",
			task:     queue.TaskEnqueueRequest{TaskBase: queue.TaskBase{Type: "test"}},
			expError: queue.ErrTaskQueueNotSpecified.Error(),
		},
		{
			name:     "Returns an error when the task type is empty",
			task:     queue.TaskEnqueueRequest{TaskBase: queue.TaskBase{Queue: queueID1}},
			expError: queue.ErrTaskTypeNotSpecified.Error(),
		},

		{
			name: "Returns an error when the spec is not valid JSON",
			task: queue.TaskEnqueueRequest{
				TaskBase: queue.TaskBase{
					Queue: "queue1",
					Type:  "test",
					Spec:  []byte("invalid"),
				},
			},
			expError: "pq: invalid input syntax for type json",
		},
		{
			name: "Returns no error when a task is valid",
			task: queue.TaskEnqueueRequest{
				TaskBase: queue.TaskBase{
					Queue: "queue2",
					Type:  "test",
					Spec:  spec,
				},
			},
		},
		{
			name: "Returns no error when a task has no spec",
			task: queue.TaskEnqueueRequest{
				TaskBase: queue.TaskBase{
					Queue: "queue2",
					Type:  "test",
				},
			},
		},
		{
			name: "Returns no error when a task has references",
			task: queue.TaskEnqueueRequest{
				TaskBase: queue.TaskBase{
					Queue: "queue2",
					Type:  "test",
					Spec:  spec,
				},
				References: queue.References{
					"test_id": uuid.NewV4(),
				},
			},
		},
		{
			name: "Returns error when a task has invalid references",
			task: queue.TaskEnqueueRequest{
				TaskBase: queue.TaskBase{
					Queue: "queue2",
					Type:  "test",
					Spec:  spec,
				},
				References: queue.References{
					"invalid": uuid.NewV4(),
				},
			},
			expError: `pq: column "invalid" of relation "tasks" does not exist`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q := NewQueuer(db)
			err := q.Enqueue(ctx, tc.task)

			if tc.expError != "" {
				require.Error(t, err)
				require.Equal(t, tc.expError, err.Error())
				return
			}

			require.NoError(t, err)
		})
	}
}
