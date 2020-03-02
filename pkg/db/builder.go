package db

import "github.com/Masterminds/squirrel"

// SQLBuilder describes a basic SQL builder using the squirrel library
type SQLBuilder interface {

	// Select returns a SelectBuilder
	Select(columns ...string) squirrel.SelectBuilder

	// Insert returns a InsertBuilder
	Insert(into string) squirrel.InsertBuilder

	// Update returns a UpdateBuilder
	Update(table string) squirrel.UpdateBuilder

	// Delete returns a DeleteBuilder
	Delete(from string) squirrel.DeleteBuilder

	// RunWith sets the RunWith field for any child builders.
	RunWith(runner squirrel.BaseRunner) squirrel.StatementBuilderType
}
