package managers

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"

	"github.com/contiamo/go-base/pkg/db"
	"github.com/contiamo/go-base/pkg/http/parameters"
	"github.com/contiamo/go-base/pkg/tracing"
)

// PageInfo - Contains the pagination metadata for a response
type PageInfo struct {
	// Total number of items
	ItemCount uint32 `json:"itemCount"`
	// Maximum items that can be on the page.
	// They may be different from the requested number of times
	ItemsPerPage uint32 `json:"itemsPerPage"`
	// Item count if filters were not applied
	UnfilteredItemCount uint32 `json:"unfilteredItemCount"`
	// The current page number using 1-based array indexing
	Current uint32 `json:"current"`
}

// BaseManager describes a typical data manager
type BaseManager interface {
	tracing.Tracer
	// GetQueryBuilder creates a new squirrel builder for a SQL query
	GetQueryBuilder() squirrel.StatementBuilderType
	// GetTxQueryBuilder is the same as GetQueryBuilder but also opens a transaction
	GetTxQueryBuilder(ctx context.Context, opts *sql.TxOptions) (squirrel.StatementBuilderType, *sql.Tx, error)
	// GetPageInfo returns the page info object for a given page
	//
	// `scope` is the SQL where statement that defines the scope
	// of the query (e.g. organization_id)
	// `filter` is the SQL where statement that defines how the data is filtered by the user.
	// The user would not see any counts beyond the scope
	// but will see the total count beyond the filter
	GetPageInfo(ctx context.Context, table string, page parameters.Page, scope, filter squirrel.Sqlizer) (pageInfo PageInfo, err error)
}

// NewBaseManager creates a new base manager
func NewBaseManager(db *sql.DB, componentName string) BaseManager {
	return &baseManager{
		db:     db,
		Tracer: tracing.NewTracer("managers", componentName),
	}
}

type baseManager struct {
	db *sql.DB
	tracing.Tracer
}

func (m *baseManager) GetQueryBuilder() squirrel.StatementBuilderType {
	return squirrel.StatementBuilder.
		PlaceholderFormat(squirrel.Dollar).
		RunWith(db.WrapWithTracing(m.db))
}

func (m *baseManager) GetTxQueryBuilder(ctx context.Context, opts *sql.TxOptions) (squirrel.StatementBuilderType, *sql.Tx, error) {
	tx, err := m.db.BeginTx(ctx, opts)
	return squirrel.StatementBuilder.
		PlaceholderFormat(squirrel.Dollar).
		RunWith(db.WrapWithTracing(tx)), tx, err
}

func (m *baseManager) GetPageInfo(ctx context.Context, table string, page parameters.Page, scope, filter squirrel.Sqlizer) (pageInfo PageInfo, err error) {
	span, ctx := m.StartSpan(ctx, "GetPageInfo")
	defer func() {
		m.FinishSpan(span, err)
	}()

	pageInfo.ItemsPerPage = page.Size
	pageInfo.Current = page.Number

	span.SetTag("pageInfo.curent", pageInfo.Current)
	span.SetTag("pageInfo.itemsPerPage", pageInfo.ItemsPerPage)

	filterSQL := "1 = 1"
	filterArgs := []interface{}{}
	if filter != nil {
		filterSQL, filterArgs, err = filter.ToSql()
		if err != nil {
			return pageInfo, err
		}
	}

	scopeSQL := "1 = 1"
	scopeArgs := []interface{}{}
	if scope != nil {
		scopeSQL, scopeArgs, err = scope.ToSql()
		if err != nil {
			return pageInfo, err
		}
	}

	query := replaceQuestionMarks(fmt.Sprintf(`
SELECT COUNT(*) AS unfilteredItemCount, SUM(hit) AS itemCount FROM (
  SELECT
    *,
    (CASE WHEN (%s) THEN 1 ELSE 0 END) AS hit
  FROM %s
  WHERE %s
) AS subquery`, filterSQL, table, scopeSQL))

	args := append(filterArgs, scopeArgs...)
	row := m.db.QueryRowContext(ctx, query, args...)
	err = row.Scan(&pageInfo.UnfilteredItemCount, &pageInfo.ItemCount)
	if err != nil {
		return pageInfo, err
	}

	span.SetTag("pageInfo.itemCount", pageInfo.ItemCount)
	span.SetTag("pageInfo.unfilteredItemCount", pageInfo.UnfilteredItemCount)

	return pageInfo, err
}

func replaceQuestionMarks(query string) string {
	res := query
	i := 1
	for {
		next := strings.Replace(res, "?", fmt.Sprintf("$%v", i), 1)
		if next == res {
			break
		}
		i++
		res = next
	}
	return res
}
