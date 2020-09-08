package postgres

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	dbtest "github.com/contiamo/go-base/pkg/db/test"
	"github.com/contiamo/go-base/pkg/queue"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestSetupTables(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logrus.SetOutput(ioutil.Discard)
	defer logrus.SetOutput(os.Stdout)

	t.Run("bootstraps without references", func(t *testing.T) {
		_, db := dbtest.GetDatabase(t)
		defer db.Close()
		require.NoError(t, SetupTables(ctx, db, nil))
	})

	t.Run("bootstraps with references", func(t *testing.T) {
		_, db := dbtest.GetDatabase(t)
		defer db.Close()

		_, err := db.ExecContext(ctx, "CREATE TABLE some_entity(id UUID PRIMARY KEY);")
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, "INSERT INTO some_entity(id) values('27195537-cc37-40db-acdb-424d448ec805');")
		require.NoError(t, err)

		err = SetupTables(ctx, db, []ForeignReference{
			{
				ColumnName:       "entity_id",
				ColumnType:       "UUID",
				ReferencedTable:  "some_entity",
				ReferencedColumn: "id",
			},
		})
		require.NoError(t, err)

		// create a task on the queue using this reference
		queuer := NewQueuer(db)
		err = queuer.Enqueue(ctx, queue.TaskEnqueueRequest{
			TaskBase: queue.TaskBase{
				Queue: "test-queue",
				Type:  queue.TaskType("test-task-type"),
				Spec:  queue.Spec("{}"),
			},
			References: queue.References{
				"entity_id": "27195537-cc37-40db-acdb-424d448ec805",
			},
		})
		require.NoError(t, err)
		dbtest.EqualCount(t, db, 1, "tasks", nil)

		// delete the entity, and see how the task disappears
		_, err = db.ExecContext(ctx, "DELETE FROM some_entity;")
		require.NoError(t, err)
		dbtest.EqualCount(t, db, 0, "tasks", nil)

	})

	t.Run("returns error when try to override a system column with a reference", func(t *testing.T) {
		_, db := dbtest.GetDatabase(t)
		defer db.Close()

		err := SetupTables(ctx, db, []ForeignReference{
			{
				ColumnName:       "schedule_id",
				ColumnType:       "UUID",
				ReferencedTable:  "schedules",
				ReferencedColumn: "schedule_id",
			},
		})
		require.Error(t, err)
		require.Equal(t, "failed to replace a system column \"schedule_id\" with a reference", err.Error())
	})

	t.Run("adds and drops columns when the references change", func(t *testing.T) {
		_, db := dbtest.GetDatabase(t)
		defer db.Close()

		_, err := db.ExecContext(ctx, "CREATE TABLE first(id UUID PRIMARY KEY);")
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, "INSERT INTO first(id) values('27195537-cc37-40db-acdb-424d448ec805');")
		require.NoError(t, err)

		_, err = db.ExecContext(ctx, "CREATE TABLE second(id UUID PRIMARY KEY);")
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, "INSERT INTO second(id) values('27195537-cc37-40db-acdb-424d448ec806');")
		require.NoError(t, err)

		err = SetupTables(ctx, db, []ForeignReference{
			{
				ColumnName:       "first_id",
				ColumnType:       "UUID",
				ReferencedTable:  "first",
				ReferencedColumn: "id",
			},
		})
		require.NoError(t, err)

		// now change the references
		err = SetupTables(ctx, db, []ForeignReference{
			{
				ColumnName:       "second_id",
				ColumnType:       "UUID",
				ReferencedTable:  "second",
				ReferencedColumn: "id",
			},
		})
		require.NoError(t, err)

		queuer := NewQueuer(db)

		// try create a task on the queue using the old reference
		err = queuer.Enqueue(ctx, queue.TaskEnqueueRequest{
			TaskBase: queue.TaskBase{
				Queue: "test-queue",
				Type:  queue.TaskType("test-task-type"),
				Spec:  queue.Spec("{}"),
			},
			References: queue.References{
				"first_id": "27195537-cc37-40db-acdb-424d448ec805",
			},
		})
		require.Error(t, err)
		require.Equal(t, "pq: column \"first_id\" of relation \"tasks\" does not exist", err.Error())
		dbtest.EqualCount(t, db, 0, "tasks", nil)

		err = queuer.Enqueue(ctx, queue.TaskEnqueueRequest{
			TaskBase: queue.TaskBase{
				Queue: "test-queue",
				Type:  queue.TaskType("test-task-type"),
				Spec:  queue.Spec("{}"),
			},
			References: queue.References{
				"second_id": "27195537-cc37-40db-acdb-424d448ec806",
			},
		})
		require.NoError(t, err)
		dbtest.EqualCount(t, db, 1, "tasks", nil)

		// delete the entity, and see how the task disappears
		_, err = db.ExecContext(ctx, "DELETE FROM second;")
		require.NoError(t, err)
		dbtest.EqualCount(t, db, 0, "tasks", nil)

	})
}
