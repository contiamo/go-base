package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/contiamo/go-base/v4/pkg/tracing"
	"github.com/opentracing/opentracing-go"
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
	// WithTrimmedQuery sets the max length of the SQL query logged in open tracing.
	// Depending on the chosen transport open tracing can have limitations on the length of the span
	// If the consumer of the interface anticipates building very long SQL queries they might want
	// to use this function.
	// `maxLength == 0` means no query will be logged in open tracing.
	// `maxLength > 0` does `query[:maxLength]+"..."`, so please account for 3 additional characters.
	WithTrimmedQuery(maxLength uint) TraceableDB
}

// WrapWithTracing Wraps a SQL database with open tracing and logging
func WrapWithTracing(db SQLDB) TraceableDB {
	return &traceableDB{
		SQLDB:          db,
		Tracer:         tracing.NewTracer("db", "traceableSQL"),
		maxQueryLength: -1,
	}
}

type traceableDB struct {
	SQLDB
	tracing.Tracer
	maxQueryLength int
}

func (d traceableDB) logQuery(span opentracing.Span, query string) {
	// `0` means no logging
	if d.maxQueryLength == 0 {
		return
	}

	// `-1` means, no trimming
	// any other number trims the query
	if d.maxQueryLength > 0 && len(query) > d.maxQueryLength {
		query = query[:d.maxQueryLength] + "..."
	}
	span.LogKV("sql", query)
}

func (d traceableDB) WithTrimmedQuery(maxLength uint) TraceableDB {
	// we modify the copy of the struct here
	d.maxQueryLength = int(maxLength)
	return &d
}

func (d traceableDB) ExecContext(ctx context.Context, query string, args ...interface{}) (result sql.Result, err error) {
	span, ctx := d.StartSpan(ctx, "ExecContext")
	defer func() {
		d.FinishSpan(span, err)
	}()

	d.logQuery(span, query)
	logrus.WithTime(time.Now()).WithField("sql_method", "ExecContext").Debug(query)

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

	d.logQuery(span, query)
	logrus.WithTime(time.Now()).WithField("sql_method", "QueryContext").Debug(query)

	//nolint: sqlclosecheck // rows are supposed to be not closed because it's a wrapper
	rows, err = d.SQLDB.QueryContext(ctx, query, args...)
	return rows, err
}
func (d traceableDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return d.QueryContext(context.Background(), query, args...)
}

func (d traceableDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	span, ctx := d.StartSpan(ctx, "QueryRowContext")
	defer d.FinishSpan(span, nil)

	d.logQuery(span, query)
	logrus.WithTime(time.Now()).WithField("sql_method", "QueryRowContext").Debug(query)

	return d.SQLDB.QueryRowContext(ctx, query, args...)
}

func (d traceableDB) QueryRow(query string, args ...interface{}) *sql.Row {
	return d.QueryRowContext(context.Background(), query, args...)
}
