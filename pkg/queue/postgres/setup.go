package postgres

import (
	"bytes"
	"context"
	"text/template"

	"github.com/contiamo/go-base/pkg/db"
)

type ForeignReference struct {
	ColumnName       string
	ColumnType       string
	ReferencedTable  string
	ReferencedColumn string
}

func SetupTables(ctx context.Context, db db.SQLDB, references []ForeignReference) error {
	// setup database
	buf := &bytes.Buffer{}
	err := dbSetupTemplate.Execute(buf, references)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, buf.String())
	if err != nil {
		return err
	}

	return nil
}

var dbSetupTemplate = template.Must(template.New("queue-db-setup").Parse(`
-- eventually add missing CITEXT extension
CREATE EXTENSION IF NOT EXISTS citext;

-- create 'schedules' and 'tasks' table
CREATE TABLE IF NOT EXISTS schedules (
    schedule_id uuid PRIMARY KEY,
    task_queue citext NOT NULL,
    task_type citext NOT NULL,
    task_spec jsonb NOT NULL,
    cron_schedule citext NOT NULL DEFAULT '',
    next_execution_time timestamptz,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW()
    -- Additional columns which reference to something, used for clean up
    {{ range . }}
    ,{{ .ColumnName }} {{ .ColumnType }} REFERENCES {{ .ReferencedTable }} ({{ .ReferencedColumn}}) ON DELETE CASCADE
    {{ end }}
);
CREATE TABLE IF NOT EXISTS tasks (
    task_id uuid PRIMARY KEY,
    queue citext NOT NULL,
    type citext NOT NULL,
    spec jsonb NOT NULL,
    status citext NOT NULL,
    progress jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    started_at timestamptz,
    finished_at timestamptz,
    last_heartbeat_at timestamptz,
    -- Additional columns for tasks used for clean up
    schedule_id uuid REFERENCES schedules ON DELETE CASCADE
    -- Additional columns which reference to something, used for clean up
    {{ range . }}
    ,{{ .ColumnName }} {{ .ColumnType }} REFERENCES {{ .ReferencedTable }} ({{ .ReferencedColumn}}) ON DELETE CASCADE
    {{ end }}
);

-- create indexes
CREATE INDEX IF NOT EXISTS schedule_next_execution_time_idx ON schedules (next_execution_time);
CREATE INDEX IF NOT EXISTS schedule_task_type_idx ON schedules (task_type);
CREATE INDEX IF NOT EXISTS tasks_queue_idx ON tasks (queue);
CREATE INDEX IF NOT EXISTS tasks_type_idx ON tasks USING hash (type);
CREATE INDEX IF NOT EXISTS tasks_created_heartbeat_at_idx ON tasks (created_at, last_heartbeat_at);
CREATE INDEX IF NOT EXISTS tasks_created_desc_heartbeat_at_desc_idx ON tasks (created_at DESC, last_heartbeat_at DESC);
CREATE INDEX IF NOT EXISTS tasks_created_desc_idx ON tasks (created_at DESC);
CREATE INDEX IF NOT EXISTS tasks_last_heartbeat_at_desc_idx ON tasks (last_heartbeat_at DESC);
CREATE INDEX IF NOT EXISTS tasks_created_updated_desc_idx ON tasks (created_at DESC, updated_at DESC);
CREATE INDEX IF NOT EXISTS tasks_started_at_idx ON tasks (started_at)
WHERE
  started_at IS NULL;
CREATE INDEX IF NOT EXISTS tasks_completed_at_idx ON tasks (finished_at)
  WHERE
  finished_at IS NULL;

-- notify on channel 'task_update' on changes on the tasks table
CREATE OR REPLACE FUNCTION notify_task_update ()
    RETURNS TRIGGER
    AS $$
BEGIN
    PERFORM
        pg_notify('task_update', '');
    RETURN NULL;
END;
$$
LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS notify_task_update_trigger ON tasks;
CREATE TRIGGER notify_task_update_trigger
    AFTER INSERT ON tasks
    FOR EACH ROW
    EXECUTE PROCEDURE notify_task_update ();
`))
