package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/contiamo/go-base/v3/pkg/db"
	cstrings "github.com/contiamo/go-base/v3/pkg/strings"
	"github.com/sirupsen/logrus"
)

const (
	// TasksTable is the name of the Postgres table that is used for tasks
	TasksTable = "tasks"
	// SchedulesTable is the name of the Postgres table used for schedules
	SchedulesTable = "schedules"

	createTableTmpl = `
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE IF NOT EXISTS %s (
%s
);
`
	alterTableTmpl = `
CREATE EXTENSION IF NOT EXISTS citext;

ALTER TABLE %s
%s;
`
	notifySetup = `
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
`
)

var (
	none = nothing{}

	// list of system columns required for the queue to function

	scheduleColumns = tableColumnSet{
		"schedule_id":         "uuid PRIMARY KEY",
		"task_queue":          "citext NOT NULL",
		"task_type":           "citext NOT NULL",
		"task_spec":           "jsonb NOT NULL",
		"cron_schedule":       "citext NOT NULL DEFAULT ''",
		"next_execution_time": "timestamptz",
		"created_at":          "timestamptz NOT NULL DEFAULT NOW()",
		"updated_at":          "timestamptz NOT NULL DEFAULT NOW()",
	}

	taskColumns = tableColumnSet{
		"task_id":           "uuid PRIMARY KEY",
		"queue":             "citext NOT NULL",
		"type":              "citext NOT NULL",
		"spec":              "jsonb NOT NULL",
		"status":            "citext NOT NULL",
		"progress":          "jsonb NOT NULL",
		"created_at":        "timestamptz NOT NULL DEFAULT NOW()",
		"updated_at":        "timestamptz NOT NULL DEFAULT NOW()",
		"started_at":        "timestamptz",
		"finished_at":       "timestamptz",
		"last_heartbeat_at": "timestamptz",
		"schedule_id":       "uuid REFERENCES schedules ON DELETE CASCADE",
	}

	// list of indexes on the system columns defined above
	indexes = indexList{

		// schedules
		{
			Table:   SchedulesTable,
			Columns: []string{"next_execution_time DESC"},
		},
		{
			Table:   SchedulesTable,
			Columns: []string{"task_queue"},
			Type:    "hash",
		},
		{
			Table:   SchedulesTable,
			Columns: []string{"task_type"},
			Type:    "hash",
		},
		{
			Table:   SchedulesTable,
			Columns: []string{"created_at DESC", "updated_at DESC"},
		},
		{
			Table:     SchedulesTable,
			Name:      "unique_retention_idx",
			Columns:   []string{"task_queue", "task_type", "(task_spec->>'queueName')", "(task_spec->>'taskType')", "(task_spec->>'status')"},
			Unique:    true,
			Condition: fmt.Sprintf("task_type='%s'", RetentionTask),
		},

		// tasks
		{
			Table:   TasksTable,
			Columns: []string{"queue"},
			Type:    "hash",
		},
		{
			Table:   TasksTable,
			Columns: []string{"type"},
			Type:    "hash",
		},
		{
			Table:   TasksTable,
			Columns: []string{"status"},
			Type:    "hash",
		},
		{
			Table:   TasksTable,
			Columns: []string{"schedule_id"},
			Type:    "hash",
		},
		{
			Table:   TasksTable,
			Columns: []string{"created_at", "last_heartbeat_at"},
		},
		{
			Table:   TasksTable,
			Columns: []string{"created_at DESC", "last_heartbeat_at DESC"},
		},
		{
			Table:   TasksTable,
			Columns: []string{"created_at DESC"},
		},
		{
			Table:   TasksTable,
			Columns: []string{"last_heartbeat_at DESC"},
		},
		{
			Table:   TasksTable,
			Columns: []string{"created_at DESC", "updated_at DESC"},
		},
		{
			Table:   TasksTable,
			Columns: []string{"started_at DESC"},
		},
		{
			Table:   TasksTable,
			Columns: []string{"finished_at DESC"},
		},
	}
)

// ForeignReference describes a foreign key reference in the queue system
type ForeignReference struct {
	// ColumnName is a name of the colum in the `tasks` and `schedules` tables
	ColumnName string
	// ColumnType is a type of the colum in the `tasks` and `schedules` tables
	ColumnType string
	// ReferencedTable is a table name this column should be referencing
	ReferencedTable string
	// ReferencedColumn is a column name this column should be referencing
	ReferencedColumn string
}

// SetupTables sets up all the necessary tables, foreign keys and indexes.
//
// This supports both: initial bootstrapping and changing of the reference list.
// However, it does not apply changes to an existing reference, this will do nothing.
func SetupTables(ctx context.Context, db db.SQLDB, references []ForeignReference) (err error) {
	logrus.Debug("checking queue-related tables...")

	logrus.Debug("checking `schedules` table...")
	err = syncTable(ctx, db, SchedulesTable, scheduleColumns, references)
	if err != nil {
		return err
	}
	logrus.Debug("`schedules` table is up to date")

	logrus.Debug("checking `tasks` table...")
	err = syncTable(ctx, db, TasksTable, taskColumns, references)
	if err != nil {
		return err
	}
	logrus.Debug("`tasks` table is up to date")

	logrus.Debug("assert the notification trigger...")
	logrus.Debug(notifySetup)
	_, err = db.ExecContext(ctx, notifySetup)
	if err != nil {
		return err
	}
	logrus.Debug("the notification trigger is up to date")

	applyIndexes := make(indexList, 0, len(references)+len(indexes))
	for _, ref := range references {
		applyIndexes = append(applyIndexes, index{
			Table:   SchedulesTable,
			Columns: []string{ref.ColumnName},
			Type:    "hash",
		})
		applyIndexes = append(applyIndexes, index{
			Table:   TasksTable,
			Columns: []string{ref.ColumnName},
			Type:    "hash",
		})
	}
	applyIndexes = append(indexes, applyIndexes...)
	logrus.
		WithField("count", len(applyIndexes)).
		Debug("assert indexes...")
	query := strings.Join(applyIndexes.generateStatements(), "\n")
	logrus.Debug(query)
	_, err = db.ExecContext(ctx, query)
	if err != nil {
		return err
	}
	logrus.Debug("indexes are up to date")
	return err
}

func syncTable(ctx context.Context, db db.SQLDB, tableName string, initColumns tableColumnSet, references []ForeignReference) (err error) {
	expectedColumns := make(tableColumnSet, len(initColumns)+len(references))
	for columnName := range initColumns {
		expectedColumns[columnName] = initColumns[columnName]
	}
	for _, ref := range references {
		_, ok := expectedColumns[ref.ColumnName]
		if ok {
			return fmt.Errorf(
				"failed to replace a system column %q with a reference",
				ref.ColumnName,
			)
		}

		expectedColumns[ref.ColumnName] = fmt.Sprintf(
			"%s REFERENCES %s (%s) ON DELETE CASCADE",
			ref.ColumnType,
			ref.ReferencedTable,
			ref.ReferencedColumn,
		)

	}

	logrus.Debug("getting the current list of columns...")
	currentColumns, err := listColumns(ctx, db, tableName)
	if err != nil {
		return err
	}
	logrus.Debug("current list of columns received")

	// the table does not exist yet
	if len(currentColumns) == 0 {
		logrus.Debugf("table %q does not exist yet, bootstrapping...", tableName)
		columnStmts := expectedColumns.generateStatements()
		query := fmt.Sprintf(createTableTmpl, tableName, strings.Join(columnStmts, ",\n"))
		logrus.Debug(query)
		_, err = db.ExecContext(ctx, query)
		return err
	}

	dropColumns := make([]string, 0, len(currentColumns))

	// drop redundant columns
	for curColName := range currentColumns {
		_, ok := expectedColumns[curColName]
		if ok {
			continue
		}

		// if the current column is not on the expected list it should be dropped
		dropColumns = append(dropColumns, fmt.Sprintf("DROP COLUMN %s CASCADE", curColName))
	}

	addColumns := make([]string, 0, len(expectedColumns))

	// add missing columns
	for expectedColName := range expectedColumns {
		_, ok := currentColumns[expectedColName]
		if ok {
			continue
		}

		// if the expected column is not on the current list it should be added
		addColumns = append(addColumns, fmt.Sprintf(
			"ADD COLUMN %s %s",
			expectedColName,
			expectedColumns[expectedColName],
		),
		)
	}

	columnStmts := append(addColumns, dropColumns...)
	if len(columnStmts) == 0 {
		return nil
	}

	logrus.
		WithField("adding", len(addColumns)).
		WithField("dropping", len(dropColumns)).
		Debug("applying changes...")
	query := fmt.Sprintf(alterTableTmpl, tableName, strings.Join(columnStmts, ",\n"))
	logrus.Debug(query)
	_, err = db.ExecContext(ctx, query)
	return err
}

func listColumns(ctx context.Context, db db.SQLDB, tableName string) (columns map[string]nothing, err error) {
	rows, err := db.QueryContext(ctx,
		`SELECT column_name FROM information_schema.columns WHERE table_name = $1;`,
		tableName,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns = make(map[string]nothing)

	for rows.Next() {
		var columnName string
		err = rows.Scan(&columnName)
		if err != nil {
			return nil, err
		}
		columns[columnName] = none
	}

	return columns, nil
}

type nothing struct{}

type tableColumnSet map[string]string

// generateStatements generates a list of statements that can be put
// inside a CREATE TABLE <tableName> (result[0], result[1]...) statement.
func (s tableColumnSet) generateStatements() []string {
	columnDefinitions := make([]string, 0, len(s))
	for name := range s {
		columnDefinitions = append(columnDefinitions, name+" "+s[name])
	}

	return columnDefinitions
}

// index describes all important properties of an index
type index struct {
	Name      string
	Table     string
	Columns   []string
	Type      string
	Unique    bool
	Condition string
}
type indexList []index

// generateStatements generates a list of statements that can be executed
// and they will make sure that contained indexes exist
func (l indexList) generateStatements() []string {
	indexDefinitions := make([]string, 0, len(l))
	for _, index := range l {
		stmt := strings.Builder{}
		stmt.WriteString("CREATE ")
		if index.Unique {
			stmt.WriteString("UNIQUE ")
		}
		stmt.WriteString("INDEX IF NOT EXISTS ")

		columnList := strings.Join(index.Columns, ",")

		// is a noop when string is empty
		stmt.WriteString(index.Name)
		if index.Name == "" {
			stmt.WriteString(fmt.Sprintf(
				"%s_%s_idx",
				index.Table,
				cstrings.ToUnderscoreCase(columnList),
			))
		}

		stmt.WriteString(" ON ")
		stmt.WriteString(index.Table)

		if index.Type != "" {
			stmt.WriteString(fmt.Sprintf(" USING %s ", index.Type))
		}

		stmt.WriteString("(" + columnList + ")")
		if index.Condition != "" {
			stmt.WriteString(" WHERE ")
			stmt.WriteString(index.Condition)
		}
		stmt.WriteString(";")
		indexDefinitions = append(indexDefinitions, stmt.String())
	}

	return indexDefinitions
}
