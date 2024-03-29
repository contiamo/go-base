package managers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	squirrel "github.com/Masterminds/squirrel"
	"github.com/contiamo/go-base/v4/pkg/db"
	dserrors "github.com/contiamo/go-base/v4/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// IDResolver makes possible to access database records by their IDs or unique values
type IDResolver interface {
	// Resolve returns an ID of the given record identified by the value which can be either
	// an UUID or a unique string value of the given secondary column.
	// where is a map of where statements to their list of arguments
	Resolve(ctx context.Context, sql db.SQLBuilder, value string, filter squirrel.Sqlizer) (string, error)
	// Sqlizer returns a Sqlizer interface that contains where statements for a given
	// filter and the ID column, so you can immediately use it with
	// the where of the select builder
	Sqlizer(ctx context.Context, sql db.SQLBuilder, value string, filter squirrel.Sqlizer) (squirrel.Sqlizer, error)
}

// NewIDResolver creates a new name->id resolver for a table, for example
// var (
//
//	CollectionIDResolver = NewIDResolver("collections", "collection_id", "name")
//	TableIDResolver = NewIDResolver("tables", "table_id", "name")
//
// )
func NewIDResolver(table, idColumn, secondaryColumn string) IDResolver {
	return &idResolver{
		table:           table,
		idColumn:        idColumn,
		secondaryColumn: secondaryColumn,
	}
}

type idResolver struct {
	table,
	idColumn,
	secondaryColumn string
}

func (r *idResolver) Sqlizer(ctx context.Context, sql db.SQLBuilder, value string, filter squirrel.Sqlizer) (squirrel.Sqlizer, error) {
	id, err := r.Resolve(ctx, sql, value, filter)
	if err != nil {
		return nil, err
	}

	idPred := squirrel.Eq{
		r.idColumn: id,
	}
	if filter == nil {
		return idPred, nil
	}
	return squirrel.And{
		filter,
		idPred,
	}, nil
}

func (r *idResolver) Resolve(ctx context.Context, sql db.SQLBuilder, value string, filter squirrel.Sqlizer) (string, error) {
	if value == "" {
		return value, dserrors.ValidationErrors{
			"id": errors.New("the id parameter can't be empty"),
		}.Filter()
	}

	// by default lookup the given value on the secondary column to retrieve the id
	lookupColumn := r.secondaryColumn

	_, err := uuid.FromString(strings.TrimSpace(value))
	if err == nil {
		// given value is already an id!
		// -> lookup value on the id column together with the additional filter
		//    to validate that the id exists under the given filter
		lookupColumn = r.idColumn
	}

	rows, err := sql.
		Select(r.idColumn).
		From(r.table).
		Where(filter).
		Where(squirrel.Eq{lookupColumn: value}).
		Limit(2).
		QueryContext(ctx)

	if err != nil {
		return "", err
	}

	defer rows.Close()

	// no results at all
	if !rows.Next() {
		return "", dserrors.ErrNotFound
	}

	if rows.Err() != nil {
		return "", rows.Err()
	}

	var id string
	err = rows.Scan(&id)
	if err != nil {
		return id, err
	}
	// non-unique result
	if rows.Next() {
		return "", fmt.Errorf(
			"id for `%s = %s` can't be resolved in `%s` due to non-unique results",
			r.secondaryColumn,
			value,
			r.table,
		)
	}

	if rows.Err() != nil {
		return "", rows.Err()
	}

	return id, err
}
