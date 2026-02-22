package query

import (
	"database/sql"
	"fmt"
	"reflect"

	"github.com/Grandbusta/jone/dialect"
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
	dialect          dialect.Dialect
	execer           Execer
	err              error // deferred error (e.g. from lazy connection)
	onConflictIgnore bool
}

// ConflictBuilder provides conflict resolution methods for INSERT queries.
type ConflictBuilder struct {
	insert *InsertBuilder
}

// OnConflict starts a conflict resolution clause.
func (i *InsertBuilder) OnConflict() *ConflictBuilder {
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

// Into sets the target table and executes the insert.
func (i *InsertBuilder) Into(table string) (sql.Result, error) {
	if i.err != nil {
		return nil, i.err
	}
	if i.execer == nil {
		return nil, fmt.Errorf("no database connection")
	}

	opts := dialect.InsertOptions{
		OnConflictIgnore: i.onConflictIgnore,
	}

	switch data := i.data.(type) {
	case map[string]any:
		if len(data) == 0 {
			return nil, fmt.Errorf("no data to insert")
		}
		query, args := i.dialect.InsertSQL(table, data, opts)
		return i.execer.Exec(query, args...)
	case []map[string]any:
		if len(data) == 0 {
			return nil, fmt.Errorf("no data to insert")
		}
		query, args := i.dialect.InsertManySQL(table, data, opts)
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
			query, args := i.dialect.InsertSQL(table, m, opts)
			return i.execer.Exec(query, args...)
		case reflect.Slice:
			maps, err := structsToMaps(i.data)
			if err != nil {
				return nil, err
			}
			if len(maps) == 0 {
				return nil, fmt.Errorf("no data to insert")
			}
			query, args := i.dialect.InsertManySQL(table, maps, opts)
			return i.execer.Exec(query, args...)
		default:
			return nil, fmt.Errorf("Insert expects map[string]any, []map[string]any, struct, or []struct, got %T", i.data)
		}
	}
}
