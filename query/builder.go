// Package query provides query building for DML operations.
package query

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/Grandbusta/jone/dialect"
)

// Builder is the main query builder interface.
type Builder interface {
	// ToSQL generates the SQL string and arguments.
	ToSQL() (string, []any)
}

// parseWhere builds a comparison condition from the variadic Where/OrWhere forms:
// (column, value) with an implied "=", or (column, operator, value).
func parseWhere(or bool, args []any) (dialect.Cond, error) {
	switch len(args) {
	case 2:
		col, ok := args[0].(string)
		if !ok {
			return dialect.Cond{}, fmt.Errorf("Where column must be a string, got %T", args[0])
		}
		return dialect.Cond{Kind: dialect.CondCmp, Or: or, Column: col, Op: "=", Value: args[1]}, nil
	case 3:
		col, ok := args[0].(string)
		if !ok {
			return dialect.Cond{}, fmt.Errorf("Where column must be a string, got %T", args[0])
		}
		op, ok := args[1].(string)
		if !ok {
			return dialect.Cond{}, fmt.Errorf("Where operator must be a string, got %T", args[1])
		}
		return dialect.Cond{Kind: dialect.CondCmp, Or: or, Column: col, Op: op, Value: args[2]}, nil
	default:
		return dialect.Cond{}, fmt.Errorf("Where expects (column, value) or (column, operator, value), got %d args", len(args))
	}
}

// inValues normalizes WhereIn's variadic input: a single slice argument
// (e.g. []string{"a", "b"}) is expanded element-by-element.
func inValues(values []any) []any {
	if len(values) == 1 {
		rv := reflect.ValueOf(values[0])
		if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
			expanded := make([]any, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				expanded[i] = rv.Index(i).Interface()
			}
			return expanded
		}
	}
	return values
}

// parseWhereRaw builds a raw condition, validating that the number of ?
// placeholders matches the number of bound args.
func parseWhereRaw(raw string, args []any) (dialect.Cond, error) {
	if n := strings.Count(raw, "?"); n != len(args) {
		return dialect.Cond{}, fmt.Errorf("WhereRaw has %d placeholders but %d args", n, len(args))
	}
	return dialect.Cond{Kind: dialect.CondRaw, Raw: raw, Values: args}, nil
}

// SelectBuilder builds SELECT queries.
type SelectBuilder struct {
	table   string
	columns []string
	where   []dialect.Cond
	orderBy []dialect.OrderClause
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

// setErr records a deferred error without masking an earlier one.
func (s *SelectBuilder) setErr(err error) {
	if s.err == nil {
		s.err = err
	}
}

// Where adds a parameterized condition: Where(column, value) for equality,
// or Where(column, operator, value).
func (s *SelectBuilder) Where(args ...any) *SelectBuilder {
	cond, err := parseWhere(false, args)
	if err != nil {
		s.setErr(err)
		return s
	}
	s.where = append(s.where, cond)
	return s
}

// OrWhere is like Where but joins with OR instead of AND.
func (s *SelectBuilder) OrWhere(args ...any) *SelectBuilder {
	cond, err := parseWhere(true, args)
	if err != nil {
		s.setErr(err)
		return s
	}
	s.where = append(s.where, cond)
	return s
}

// WhereIn adds an IN condition. Accepts variadic values or a single slice:
// WhereIn("status", "active", "pending") or WhereIn("status", []string{...}).
func (s *SelectBuilder) WhereIn(column string, values ...any) *SelectBuilder {
	s.where = append(s.where, dialect.Cond{Kind: dialect.CondIn, Column: column, Values: inValues(values)})
	return s
}

// WhereNotIn adds a NOT IN condition.
func (s *SelectBuilder) WhereNotIn(column string, values ...any) *SelectBuilder {
	s.where = append(s.where, dialect.Cond{Kind: dialect.CondIn, Not: true, Column: column, Values: inValues(values)})
	return s
}

// WhereNull adds an IS NULL condition.
func (s *SelectBuilder) WhereNull(column string) *SelectBuilder {
	s.where = append(s.where, dialect.Cond{Kind: dialect.CondNull, Column: column})
	return s
}

// WhereNotNull adds an IS NOT NULL condition.
func (s *SelectBuilder) WhereNotNull(column string) *SelectBuilder {
	s.where = append(s.where, dialect.Cond{Kind: dialect.CondNull, Not: true, Column: column})
	return s
}

// WhereRaw adds a raw SQL condition with ? placeholders bound to args.
// Every ? is treated as a placeholder, including inside string literals.
func (s *SelectBuilder) WhereRaw(raw string, args ...any) *SelectBuilder {
	cond, err := parseWhereRaw(raw, args)
	if err != nil {
		s.setErr(err)
		return s
	}
	s.where = append(s.where, cond)
	return s
}

// OrderBy adds an ORDER BY clause for a column with an optional direction
// ("asc" or "desc", case-insensitive). Omitting the direction uses the
// database default (ascending). The column is quoted; for expressions use
// OrderByRaw.
func (s *SelectBuilder) OrderBy(column string, direction ...string) *SelectBuilder {
	clause := dialect.OrderClause{Column: column}
	switch len(direction) {
	case 0:
	case 1:
		dir := strings.ToUpper(direction[0])
		if dir != "ASC" && dir != "DESC" {
			s.setErr(fmt.Errorf(`OrderBy direction must be "asc" or "desc", got %q`, direction[0]))
			return s
		}
		clause.Dir = dir
	default:
		s.setErr(fmt.Errorf("OrderBy expects (column) or (column, direction), got %d args", len(direction)+1))
		return s
	}
	s.orderBy = append(s.orderBy, clause)
	return s
}

// OrderByRaw adds a raw ORDER BY expression used verbatim,
// e.g. OrderByRaw("lower(name) DESC").
func (s *SelectBuilder) OrderByRaw(raw string) *SelectBuilder {
	s.orderBy = append(s.orderBy, dialect.OrderClause{Raw: raw})
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
	return s.dialect.SelectSQL(s.table, s.columns, s.where, s.orderBy, s.limit, s.offset)
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

// First executes the query with LIMIT 1 and returns the first row as a map.
// Returns sql.ErrNoRows if no row matches. []byte column values are
// converted to string.
func (s *SelectBuilder) First() (map[string]any, error) {
	one := 1
	s.limit = &one

	rows, err := s.Exec()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, sql.ErrNoRows
	}

	vals := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	if err := rows.Scan(ptrs...); err != nil {
		return nil, err
	}

	row := make(map[string]any, len(cols))
	for i, col := range cols {
		if b, ok := vals[i].([]byte); ok {
			row[col] = string(b)
		} else {
			row[col] = vals[i]
		}
	}
	return row, rows.Err()
}

// UpdateBuilder builds UPDATE queries.
type UpdateBuilder struct {
	table   string
	set     map[string]any
	where   []dialect.Cond
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

// setErr records a deferred error without masking an earlier one.
func (u *UpdateBuilder) setErr(err error) {
	if u.err == nil {
		u.err = err
	}
}

// Where adds a parameterized condition: Where(column, value) for equality,
// or Where(column, operator, value).
func (u *UpdateBuilder) Where(args ...any) *UpdateBuilder {
	cond, err := parseWhere(false, args)
	if err != nil {
		u.setErr(err)
		return u
	}
	u.where = append(u.where, cond)
	return u
}

// OrWhere is like Where but joins with OR instead of AND.
func (u *UpdateBuilder) OrWhere(args ...any) *UpdateBuilder {
	cond, err := parseWhere(true, args)
	if err != nil {
		u.setErr(err)
		return u
	}
	u.where = append(u.where, cond)
	return u
}

// WhereIn adds an IN condition. Accepts variadic values or a single slice.
func (u *UpdateBuilder) WhereIn(column string, values ...any) *UpdateBuilder {
	u.where = append(u.where, dialect.Cond{Kind: dialect.CondIn, Column: column, Values: inValues(values)})
	return u
}

// WhereNotIn adds a NOT IN condition.
func (u *UpdateBuilder) WhereNotIn(column string, values ...any) *UpdateBuilder {
	u.where = append(u.where, dialect.Cond{Kind: dialect.CondIn, Not: true, Column: column, Values: inValues(values)})
	return u
}

// WhereNull adds an IS NULL condition.
func (u *UpdateBuilder) WhereNull(column string) *UpdateBuilder {
	u.where = append(u.where, dialect.Cond{Kind: dialect.CondNull, Column: column})
	return u
}

// WhereNotNull adds an IS NOT NULL condition.
func (u *UpdateBuilder) WhereNotNull(column string) *UpdateBuilder {
	u.where = append(u.where, dialect.Cond{Kind: dialect.CondNull, Not: true, Column: column})
	return u
}

// WhereRaw adds a raw SQL condition with ? placeholders bound to args.
func (u *UpdateBuilder) WhereRaw(raw string, args ...any) *UpdateBuilder {
	cond, err := parseWhereRaw(raw, args)
	if err != nil {
		u.setErr(err)
		return u
	}
	u.where = append(u.where, cond)
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
	where   []dialect.Cond
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

// setErr records a deferred error without masking an earlier one.
func (d *DeleteBuilder) setErr(err error) {
	if d.err == nil {
		d.err = err
	}
}

// Where adds a parameterized condition: Where(column, value) for equality,
// or Where(column, operator, value).
func (d *DeleteBuilder) Where(args ...any) *DeleteBuilder {
	cond, err := parseWhere(false, args)
	if err != nil {
		d.setErr(err)
		return d
	}
	d.where = append(d.where, cond)
	return d
}

// OrWhere is like Where but joins with OR instead of AND.
func (d *DeleteBuilder) OrWhere(args ...any) *DeleteBuilder {
	cond, err := parseWhere(true, args)
	if err != nil {
		d.setErr(err)
		return d
	}
	d.where = append(d.where, cond)
	return d
}

// WhereIn adds an IN condition. Accepts variadic values or a single slice.
func (d *DeleteBuilder) WhereIn(column string, values ...any) *DeleteBuilder {
	d.where = append(d.where, dialect.Cond{Kind: dialect.CondIn, Column: column, Values: inValues(values)})
	return d
}

// WhereNotIn adds a NOT IN condition.
func (d *DeleteBuilder) WhereNotIn(column string, values ...any) *DeleteBuilder {
	d.where = append(d.where, dialect.Cond{Kind: dialect.CondIn, Not: true, Column: column, Values: inValues(values)})
	return d
}

// WhereNull adds an IS NULL condition.
func (d *DeleteBuilder) WhereNull(column string) *DeleteBuilder {
	d.where = append(d.where, dialect.Cond{Kind: dialect.CondNull, Column: column})
	return d
}

// WhereNotNull adds an IS NOT NULL condition.
func (d *DeleteBuilder) WhereNotNull(column string) *DeleteBuilder {
	d.where = append(d.where, dialect.Cond{Kind: dialect.CondNull, Not: true, Column: column})
	return d
}

// WhereRaw adds a raw SQL condition with ? placeholders bound to args.
func (d *DeleteBuilder) WhereRaw(raw string, args ...any) *DeleteBuilder {
	cond, err := parseWhereRaw(raw, args)
	if err != nil {
		d.setErr(err)
		return d
	}
	d.where = append(d.where, cond)
	return d
}

// ToSQL generates the DELETE SQL.
func (d *DeleteBuilder) ToSQL() (string, []any) {
	if d.dialect == nil {
		return "", nil
	}
	return d.dialect.DeleteSQL(d.table, d.where)
}

// Exec executes the DELETE query and returns the result.
func (d *DeleteBuilder) Exec() (sql.Result, error) {
	if d.err != nil {
		return nil, d.err
	}
	if d.execer == nil {
		return nil, fmt.Errorf("no database connection")
	}
	sqlStr, args := d.ToSQL()
	return d.execer.Exec(sqlStr, args...)
}
