package query

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/Grandbusta/jone/dialect"
)

// RawBuilder runs a raw SQL statement with ? placeholders bound to args,
// rebound to the dialect's placeholder style ($n on PostgreSQL).
type RawBuilder struct {
	raw     string
	args    []any
	dialect dialect.Dialect
	execer  Execer
	err     error
}

// NewRawBuilder creates a dialect-aware RawBuilder (used by Schema).
func NewRawBuilder(raw string, args []any, d dialect.Dialect, execer Execer, err error) *RawBuilder {
	b := &RawBuilder{raw: raw, args: args, dialect: d, execer: execer, err: err}
	if n := strings.Count(raw, "?"); n != len(args) && b.err == nil {
		b.err = fmt.Errorf("RawQuery has %d placeholders but %d args", n, len(args))
	}
	return b
}

// ToSQL returns the statement with placeholders rebound and its args.
func (r *RawBuilder) ToSQL() (string, []any) {
	if r.dialect == nil {
		return "", nil
	}
	return r.dialect.RawSQL(r.raw, r.args)
}

// check validates the builder state before execution.
func (r *RawBuilder) check() error {
	if r.err != nil {
		return r.err
	}
	if r.execer == nil {
		return fmt.Errorf("no database connection")
	}
	return nil
}

// Exec executes the statement and returns the result. Use for statements
// that don't return rows (UPDATE, DELETE, DDL).
func (r *RawBuilder) Exec() (sql.Result, error) {
	if err := r.check(); err != nil {
		return nil, err
	}
	sqlStr, args := r.ToSQL()
	return r.execer.Exec(sqlStr, args...)
}

// All executes the query and returns every row as a map keyed by column
// name. []byte column values are converted to string; no rows yields an
// empty result, not an error.
func (r *RawBuilder) All() ([]map[string]any, error) {
	if err := r.check(); err != nil {
		return nil, err
	}
	sqlStr, args := r.ToSQL()
	rows, err := r.execer.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRowMaps(rows)
}
