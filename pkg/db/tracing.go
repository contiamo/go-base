package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/contiamo/go-base/pkg/tracing"
	"github.com/sirupsen/logrus"
)

// SQLDB is the standard SQL database interface
// which the standard library should have but it does not
type SQLDB interface {
	// Query executes the given query as implemented by database/sql.Query.
	Query(string, ...interface{}) (*sql.Rows, error)
	// QueryContext executes a query that returns rows, typically a SELECT.
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	// QueryRow executes a query that is expected to return at most one row. QueryRow always returns a
	// non-nil value.
	QueryRow(string, ...interface{}) *sql.Row
	// QueryRowContext executes a query that is expected to return at most one row. QueryRow always
	// returns a non-nil value.
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
	// Exec executes a query without returning any rows.
	Exec(string, ...interface{}) (sql.Result, error)
	// ExecContext executes a query without returning any rows.
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
}

// TraceableDB is the SQL database wrapped with open tracing
type TraceableDB interface {
	SQLDB
	tracing.Tracer
}

// WrapWithTracing Wraps a SQL database with open tracing and logging
func WrapWithTracing(db SQLDB) TraceableDB {
	return &traceableDB{
		SQLDB:  db,
		Tracer: tracing.NewTracer("db", "traceableSQL"),
	}
}

type traceableDB struct {
	SQLDB
	tracing.Tracer
}

func (d traceableDB) ExecContext(ctx context.Context, query string, args ...interface{}) (result sql.Result, err error) {
	span, ctx := d.StartSpan(ctx, "ExecContext")
	defer func() {
		d.FinishSpan(span, err)
	}()
	span.SetTag("sql", query)
	logrus.WithTime(time.Now()).WithField("sql.method", "ExecContext").Debug(query)

	result, err = d.SQLDB.ExecContext(ctx, query, args...)
	return result, err
}
func (d traceableDB) Exec(query string, args ...interface{}) (result sql.Result, err error) {
	return d.ExecContext(context.Background(), query, args...)
}

func (d traceableDB) QueryContext(ctx context.Context, query string, args ...interface{}) (rows *sql.Rows, err error) {
	span, ctx := d.StartSpan(ctx, "QueryContext")
	defer func() {
		d.FinishSpan(span, err)
	}()
	span.SetTag("sql", query)

	logrus.WithTime(time.Now()).WithField("sql.method", "QueryContext").Debug(query)
	rows, err = d.SQLDB.QueryContext(ctx, query, args...)
	return rows, err
}
func (d traceableDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return d.QueryContext(context.Background(), query, args...)
}

func (d traceableDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	span, ctx := d.StartSpan(ctx, "QueryRowContext")
	defer d.FinishSpan(span, nil)

	span.SetTag("sql", query)

	logrus.WithTime(time.Now()).WithField("sql.method", "QueryRowContext").Debug(query)
	return d.SQLDB.QueryRowContext(ctx, query, args...)
}

func (d traceableDB) QueryRow(query string, args ...interface{}) *sql.Row {
	return d.QueryRowContext(context.Background(), query, args...)
}
