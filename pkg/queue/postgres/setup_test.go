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
)

func TestSetupTablesWithoutReferences(t *testing.T) {
	logrus.SetOutput(ioutil.Discard)
	defer logrus.SetOutput(os.Stdout)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, db := dbtest.GetDatabase(t)
	defer db.Close()
	require.NoError(t, SetupTables(ctx, db, nil))
}

func TestStupWithReferences(t *testing.T) {
	logrus.SetOutput(ioutil.Discard)
	defer logrus.SetOutput(os.Stdout)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, db := dbtest.GetDatabase(t)
	defer db.Close()

	_, err := db.ExecContext(ctx, "CREATE TABLE some_entity(id UUID PRIMARY KEY);")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, "INSERT INTO some_entity(id) values('27195537-cc37-40db-acdb-424d448ec805');")
	require.NoError(t, err)

	// setup with foreign reference
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
}
