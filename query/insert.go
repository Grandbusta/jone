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

// build normalizes the insert data and generates the SQL and args.
// returning columns are appended as a RETURNING clause where supported.
func (i *InsertBuilder) build(returning []string) (string, []any, error) {
	opts := dialect.InsertOptions{
		OnConflictIgnore: i.onConflictIgnore,
		ConflictColumns:  i.conflictColumns,
		ConflictRaw:      i.conflictRaw,
		Returning:        returning,
	}

	switch data := i.data.(type) {
	case map[string]any:
		if len(data) == 0 {
			return "", nil, fmt.Errorf("no data to insert")
		}
		query, args := i.dialect.InsertSQL(i.table, data, opts)
		return query, args, nil
	case []map[string]any:
		if len(data) == 0 {
			return "", nil, fmt.Errorf("no data to insert")
		}
		query, args := i.dialect.InsertManySQL(i.table, data, opts)
		return query, args, nil
	default:
		rv := reflect.ValueOf(i.data)
		if rv.Kind() == reflect.Ptr {
			rv = rv.Elem()
		}
		switch rv.Kind() {
		case reflect.Struct:
			m, err := structToMap(i.data)
			if err != nil {
				return "", nil, err
			}
			if len(m) == 0 {
				return "", nil, fmt.Errorf("no data to insert")
			}
			query, args := i.dialect.InsertSQL(i.table, m, opts)
			return query, args, nil
		case reflect.Slice:
			maps, err := structsToMaps(i.data)
			if err != nil {
				return "", nil, err
			}
			if len(maps) == 0 {
				return "", nil, fmt.Errorf("no data to insert")
			}
			query, args := i.dialect.InsertManySQL(i.table, maps, opts)
			return query, args, nil
		default:
			return "", nil, fmt.Errorf("Insert expects map[string]any, []map[string]any, struct, or []struct, got %T", i.data)
		}
	}
}

// ToSQL generates the INSERT SQL.
func (i *InsertBuilder) ToSQL() (string, []any) {
	if i.dialect == nil {
		return "", nil
	}
	query, args, _ := i.build(nil)
	return query, args
}

// check validates the builder state before execution.
func (i *InsertBuilder) check() error {
	if i.err != nil {
		return i.err
	}
	if i.execer == nil {
		return fmt.Errorf("no database connection")
	}
	if i.table == "" {
		return fmt.Errorf("no table specified: call Into() before Exec()")
	}
	return nil
}

// Exec executes the insert and returns the result.
func (i *InsertBuilder) Exec() (sql.Result, error) {
	if err := i.check(); err != nil {
		return nil, err
	}
	query, args, err := i.build(nil)
	if err != nil {
		return nil, err
	}
	return i.execer.Exec(query, args...)
}

// ExecReturning executes the insert with a RETURNING clause and returns the
// inserted rows. PostgreSQL only — on MySQL use Exec() and LastInsertId().
func (i *InsertBuilder) ExecReturning(columns ...string) ([]map[string]any, error) {
	if err := i.check(); err != nil {
		return nil, err
	}
	if err := checkReturning(i.dialect, columns); err != nil {
		return nil, err
	}
	query, args, err := i.build(columns)
	if err != nil {
		return nil, err
	}
	return execReturning(i.execer, query, args)
}
