package query

import (
	"database/sql"
	"fmt"
	"reflect"

	"github.com/Grandbusta/jone/dialect"
	"github.com/Grandbusta/jone/types"
)

// Execer is an interface for executing SQL (both *sql.DB and *sql.Tx).
type Execer interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

// InsertBuilder builds and executes INSERT queries.
type InsertBuilder struct {
	data             any
	table            string
	dialect          dialect.Dialect
	execer           Execer
	err              error // deferred error (e.g. from lazy connection)
	onConflictIgnore bool
	conflictColumns  []string
	conflictRaw      string
}

// ConflictBuilder provides conflict resolution methods for INSERT queries.
type ConflictBuilder struct {
	insert *InsertBuilder
}

// OnConflict starts a conflict resolution clause with an optional conflict target.
// Pass column names to target a specific index: OnConflict("email") or OnConflict("email", "user_id").
// Pass a jone.Raw() expression for partial indexes: OnConflict(jone.Raw("(email) WHERE active")).
func (i *InsertBuilder) OnConflict(targets ...any) *ConflictBuilder {
	for _, t := range targets {
		switch v := t.(type) {
		case types.RawExpr:
			i.conflictRaw = v.Expr
		case string:
			i.conflictColumns = append(i.conflictColumns, v)
		}
	}
	return &ConflictBuilder{insert: i}
}

// Ignore skips rows that conflict with existing data.
func (c *ConflictBuilder) Ignore() *InsertBuilder {
	c.insert.onConflictIgnore = true
	return c.insert
}

// NewInsertBuilder creates a new InsertBuilder.
func NewInsertBuilder(data any, d dialect.Dialect, execer Execer, err error) *InsertBuilder {
	return &InsertBuilder{data: data, dialect: d, execer: execer, err: err}
}

// Into sets the target table for the insert.
func (i *InsertBuilder) Into(table string) *InsertBuilder {
	i.table = table
	return i
}

// Exec executes the insert and returns the result.
func (i *InsertBuilder) Exec() (sql.Result, error) {
	if i.err != nil {
		return nil, i.err
	}
	if i.execer == nil {
		return nil, fmt.Errorf("no database connection")
	}
	if i.table == "" {
		return nil, fmt.Errorf("no table specified: call Into() before Exec()")
	}

	opts := dialect.InsertOptions{
		OnConflictIgnore: i.onConflictIgnore,
		ConflictColumns:  i.conflictColumns,
		ConflictRaw:      i.conflictRaw,
	}

	switch data := i.data.(type) {
	case map[string]any:
		if len(data) == 0 {
			return nil, fmt.Errorf("no data to insert")
		}
		query, args := i.dialect.InsertSQL(i.table, data, opts)
		return i.execer.Exec(query, args...)
	case []map[string]any:
		if len(data) == 0 {
			return nil, fmt.Errorf("no data to insert")
		}
		query, args := i.dialect.InsertManySQL(i.table, data, opts)
		return i.execer.Exec(query, args...)
	default:
		rv := reflect.ValueOf(i.data)
		if rv.Kind() == reflect.Ptr {
			rv = rv.Elem()
		}
		switch rv.Kind() {
		case reflect.Struct:
			m, err := structToMap(i.data)
			if err != nil {
				return nil, err
			}
			if len(m) == 0 {
				return nil, fmt.Errorf("no data to insert")
			}
			query, args := i.dialect.InsertSQL(i.table, m, opts)
			return i.execer.Exec(query, args...)
		case reflect.Slice:
			maps, err := structsToMaps(i.data)
			if err != nil {
				return nil, err
			}
			if len(maps) == 0 {
				return nil, fmt.Errorf("no data to insert")
			}
			query, args := i.dialect.InsertManySQL(i.table, maps, opts)
			return i.execer.Exec(query, args...)
		default:
			return nil, fmt.Errorf("Insert expects map[string]any, []map[string]any, struct, or []struct, got %T", i.data)
		}
	}
}
