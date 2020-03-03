package managers

import (
	"context"
	"database/sql"

	"github.com/Masterminds/squirrel"

	"github.com/contiamo/go-base/v2/pkg/db"
	"github.com/contiamo/go-base/v2/pkg/http/parameters"
	"github.com/contiamo/go-base/v2/pkg/tracing"
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
	GetQueryBuilder() db.SQLBuilder
	// GetTxQueryBuilder is the same as GetQueryBuilder but also opens a transaction
	GetTxQueryBuilder(ctx context.Context, opts *sql.TxOptions) (db.SQLBuilder, *sql.Tx, error)
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

func (m *baseManager) GetQueryBuilder() db.SQLBuilder {
	return squirrel.StatementBuilder.
		PlaceholderFormat(squirrel.Dollar).
		RunWith(db.WrapWithTracing(m.db))
}

func (m *baseManager) GetTxQueryBuilder(ctx context.Context, opts *sql.TxOptions) (db.SQLBuilder, *sql.Tx, error) {
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

	builder := m.GetQueryBuilder()

	err = builder.
		Select("COUNT(*)").
		From(table).
		Where(scope).
		QueryRowContext(ctx).
		Scan(&pageInfo.UnfilteredItemCount)
	if err != nil {
		return pageInfo, err
	}

	if filter != nil {
		err = builder.
			Select("COUNT(*)").
			From(table).
			Where(scope).
			Where(filter).
			QueryRowContext(ctx).
			Scan(&pageInfo.ItemCount)
		if err != nil {
			return pageInfo, err
		}
	} else {
		pageInfo.ItemCount = pageInfo.UnfilteredItemCount
	}

	span.SetTag("pageInfo.itemCount", pageInfo.ItemCount)
	span.SetTag("pageInfo.unfilteredItemCount", pageInfo.UnfilteredItemCount)

	return pageInfo, err
}
