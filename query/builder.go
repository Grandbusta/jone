// Package query provides query building for DML operations.
package query

import (
	"database/sql"
	"fmt"

	"github.com/Grandbusta/jone/dialect"
)

// Builder is the main query builder interface.
type Builder interface {
	// ToSQL generates the SQL string and arguments.
	ToSQL() (string, []any)
}

// SelectBuilder builds SELECT queries.
type SelectBuilder struct {
	table   string
	columns []string
	where   []string
	orderBy []string
	limit   *int
	offset  *int
	dialect dialect.Dialect
	execer  Execer
	err     error
}

// NewSelectBuilder creates a dialect-aware SelectBuilder (used by Schema).
func NewSelectBuilder(columns []string, d dialect.Dialect, execer Execer, err error) *SelectBuilder {
	return &SelectBuilder{columns: columns, dialect: d, execer: execer, err: err}
}

// Select starts building a SELECT query.
func Select(columns ...string) *SelectBuilder {
	return &SelectBuilder{columns: columns}
}

// From sets the table to select from.
func (s *SelectBuilder) From(table string) *SelectBuilder {
	s.table = table
	return s
}

// Where adds a WHERE condition.
func (s *SelectBuilder) Where(condition string) *SelectBuilder {
	s.where = append(s.where, condition)
	return s
}

// OrderBy adds an ORDER BY clause.
func (s *SelectBuilder) OrderBy(column string) *SelectBuilder {
	s.orderBy = append(s.orderBy, column)
	return s
}

// Limit sets the LIMIT clause.
func (s *SelectBuilder) Limit(n int) *SelectBuilder {
	s.limit = &n
	return s
}

// Offset sets the OFFSET clause.
func (s *SelectBuilder) Offset(n int) *SelectBuilder {
	s.offset = &n
	return s
}

// ToSQL generates the SELECT SQL.
func (s *SelectBuilder) ToSQL() (string, []any) {
	if s.dialect == nil {
		return "", nil
	}
	return s.dialect.SelectSQL(s.table, s.columns, s.where, s.orderBy, s.limit, s.offset), nil
}

// Exec executes the SELECT query and returns the result rows.
func (s *SelectBuilder) Exec() (*sql.Rows, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.execer == nil {
		return nil, fmt.Errorf("no database connection")
	}
	if s.table == "" {
		return nil, fmt.Errorf("no table specified: call From() before Exec()")
	}
	sqlStr, args := s.ToSQL()
	return s.execer.Query(sqlStr, args...)
}

// UpdateBuilder builds UPDATE queries.
type UpdateBuilder struct {
	table   string
	set     map[string]any
	where   []string
	dialect dialect.Dialect
	execer  Execer
	err     error
}

// NewUpdateBuilder creates a dialect-aware UpdateBuilder (used by Schema).
func NewUpdateBuilder(table string, d dialect.Dialect, execer Execer, err error) *UpdateBuilder {
	return &UpdateBuilder{table: table, set: make(map[string]any), dialect: d, execer: execer, err: err}
}

// Update starts building an UPDATE query.
func Update(table string) *UpdateBuilder {
	return &UpdateBuilder{table: table, set: make(map[string]any)}
}

// Set adds a column=value pair to update.
func (u *UpdateBuilder) Set(column string, value any) *UpdateBuilder {
	u.set[column] = value
	return u
}

// Where adds a WHERE condition (raw string).
func (u *UpdateBuilder) Where(condition string) *UpdateBuilder {
	u.where = append(u.where, condition)
	return u
}

// ToSQL generates the UPDATE SQL.
func (u *UpdateBuilder) ToSQL() (string, []any) {
	if u.dialect == nil {
		return "", nil
	}
	return u.dialect.UpdateSQL(u.table, u.set, u.where)
}

// Exec executes the UPDATE query and returns the result.
func (u *UpdateBuilder) Exec() (sql.Result, error) {
	if u.err != nil {
		return nil, u.err
	}
	if u.execer == nil {
		return nil, fmt.Errorf("no database connection")
	}
	if len(u.set) == 0 {
		return nil, fmt.Errorf("no values to update: call Set() before Exec()")
	}
	sqlStr, args := u.ToSQL()
	return u.execer.Exec(sqlStr, args...)
}

// DeleteBuilder builds DELETE queries.
type DeleteBuilder struct {
	table   string
	where   []string
	dialect dialect.Dialect
	execer  Execer
	err     error
}

// NewDeleteBuilder creates a dialect-aware DeleteBuilder (used by Schema).
func NewDeleteBuilder(table string, d dialect.Dialect, execer Execer, err error) *DeleteBuilder {
	return &DeleteBuilder{table: table, dialect: d, execer: execer, err: err}
}

// Delete starts building a DELETE query.
func Delete(table string) *DeleteBuilder {
	return &DeleteBuilder{table: table}
}

// Where adds a WHERE condition (raw string).
func (d *DeleteBuilder) Where(condition string) *DeleteBuilder {
	d.where = append(d.where, condition)
	return d
}

// ToSQL generates the DELETE SQL.
func (d *DeleteBuilder) ToSQL() (string, []any) {
	if d.dialect == nil {
		return "", nil
	}
	return d.dialect.DeleteSQL(d.table, d.where), nil
}

// Exec executes the DELETE query and returns the result.
func (d *DeleteBuilder) Exec() (sql.Result, error) {
	if d.err != nil {
		return nil, d.err
	}
	if d.execer == nil {
		return nil, fmt.Errorf("no database connection")
	}
	sqlStr, _ := d.ToSQL()
	return d.execer.Exec(sqlStr)
}
