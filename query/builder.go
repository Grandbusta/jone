// Package query provides query building for DML operations.
package query

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/Grandbusta/jone/dialect"
	"github.com/Grandbusta/jone/types"
)

// Builder is the main query builder interface.
type Builder interface {
	// ToSQL generates the SQL string and arguments.
	ToSQL() (string, []any)
}

// WhereGroup collects conditions for a parenthesized sub-group, built by
// passing a func to Where/OrWhere:
//
//	q.Where(func(g *query.WhereGroup) {
//	    g.Where("role", "admin").OrWhere("age", ">", 65)
//	})
//
// The full condition family is available inside, including nested groups.
type WhereGroup struct {
	conds []dialect.Cond
	err   error
}

// setErr records a deferred error without masking an earlier one.
func (g *WhereGroup) setErr(err error) {
	if g.err == nil {
		g.err = err
	}
}

// Where adds a parameterized condition to the group: Where(column, value)
// for equality, Where(column, operator, value), or Where(func) for a nested group.
func (g *WhereGroup) Where(args ...any) *WhereGroup {
	cond, err := parseWhere(false, args)
	if err != nil {
		g.setErr(err)
		return g
	}
	g.conds = append(g.conds, cond)
	return g
}

// OrWhere is like Where but joins with OR instead of AND.
func (g *WhereGroup) OrWhere(args ...any) *WhereGroup {
	cond, err := parseWhere(true, args)
	if err != nil {
		g.setErr(err)
		return g
	}
	g.conds = append(g.conds, cond)
	return g
}

// WhereNot adds a negated condition: WhereNot(column, value),
// WhereNot(column, operator, value), or WhereNot(func) for NOT (...).
func (g *WhereGroup) WhereNot(args ...any) *WhereGroup {
	cond, err := parseWhereNot(false, args)
	if err != nil {
		g.setErr(err)
		return g
	}
	g.conds = append(g.conds, cond)
	return g
}

// OrWhereNot is like WhereNot but joins with OR instead of AND.
func (g *WhereGroup) OrWhereNot(args ...any) *WhereGroup {
	cond, err := parseWhereNot(true, args)
	if err != nil {
		g.setErr(err)
		return g
	}
	g.conds = append(g.conds, cond)
	return g
}

// WhereIn adds an IN condition. Accepts variadic values or a single slice.
func (g *WhereGroup) WhereIn(column string, values ...any) *WhereGroup {
	g.conds = append(g.conds, dialect.Cond{Kind: dialect.CondIn, Column: column, Values: inValues(values)})
	return g
}

// OrWhereIn is like WhereIn but joins with OR instead of AND.
func (g *WhereGroup) OrWhereIn(column string, values ...any) *WhereGroup {
	g.conds = append(g.conds, dialect.Cond{Kind: dialect.CondIn, Or: true, Column: column, Values: inValues(values)})
	return g
}

// WhereNotIn adds a NOT IN condition.
func (g *WhereGroup) WhereNotIn(column string, values ...any) *WhereGroup {
	g.conds = append(g.conds, dialect.Cond{Kind: dialect.CondIn, Not: true, Column: column, Values: inValues(values)})
	return g
}

// OrWhereNotIn is like WhereNotIn but joins with OR instead of AND.
func (g *WhereGroup) OrWhereNotIn(column string, values ...any) *WhereGroup {
	g.conds = append(g.conds, dialect.Cond{Kind: dialect.CondIn, Or: true, Not: true, Column: column, Values: inValues(values)})
	return g
}

// WhereNull adds an IS NULL condition.
func (g *WhereGroup) WhereNull(column string) *WhereGroup {
	g.conds = append(g.conds, dialect.Cond{Kind: dialect.CondNull, Column: column})
	return g
}

// OrWhereNull is like WhereNull but joins with OR instead of AND.
func (g *WhereGroup) OrWhereNull(column string) *WhereGroup {
	g.conds = append(g.conds, dialect.Cond{Kind: dialect.CondNull, Or: true, Column: column})
	return g
}

// WhereNotNull adds an IS NOT NULL condition.
func (g *WhereGroup) WhereNotNull(column string) *WhereGroup {
	g.conds = append(g.conds, dialect.Cond{Kind: dialect.CondNull, Not: true, Column: column})
	return g
}

// OrWhereNotNull is like WhereNotNull but joins with OR instead of AND.
func (g *WhereGroup) OrWhereNotNull(column string) *WhereGroup {
	g.conds = append(g.conds, dialect.Cond{Kind: dialect.CondNull, Or: true, Not: true, Column: column})
	return g
}

// WhereRaw adds a raw SQL condition with ? placeholders bound to args.
func (g *WhereGroup) WhereRaw(raw string, args ...any) *WhereGroup {
	cond, err := parseWhereRaw(raw, args)
	if err != nil {
		g.setErr(err)
		return g
	}
	g.conds = append(g.conds, cond)
	return g
}

// OrWhereRaw is like WhereRaw but joins with OR instead of AND.
func (g *WhereGroup) OrWhereRaw(raw string, args ...any) *WhereGroup {
	cond, err := parseWhereRaw(raw, args)
	if err != nil {
		g.setErr(err)
		return g
	}
	cond.Or = true
	g.conds = append(g.conds, cond)
	return g
}

// WhereBetween adds a BETWEEN condition: "col" BETWEEN low AND high.
func (g *WhereGroup) WhereBetween(column string, low, high any) *WhereGroup {
	g.conds = append(g.conds, betweenCond(false, false, column, low, high))
	return g
}

// OrWhereBetween is like WhereBetween but joins with OR instead of AND.
func (g *WhereGroup) OrWhereBetween(column string, low, high any) *WhereGroup {
	g.conds = append(g.conds, betweenCond(true, false, column, low, high))
	return g
}

// WhereNotBetween adds a NOT BETWEEN condition.
func (g *WhereGroup) WhereNotBetween(column string, low, high any) *WhereGroup {
	g.conds = append(g.conds, betweenCond(false, true, column, low, high))
	return g
}

// OrWhereNotBetween is like WhereNotBetween but joins with OR instead of AND.
func (g *WhereGroup) OrWhereNotBetween(column string, low, high any) *WhereGroup {
	g.conds = append(g.conds, betweenCond(true, true, column, low, high))
	return g
}

// WhereExists adds an EXISTS (subquery) condition. The subquery is either a
// *SelectBuilder or a raw SQL string with ? placeholders bound to args.
func (g *WhereGroup) WhereExists(subquery any, args ...any) *WhereGroup {
	cond, err := parseExists(false, false, subquery, args)
	if err != nil {
		g.setErr(err)
		return g
	}
	g.conds = append(g.conds, cond)
	return g
}

// OrWhereExists is like WhereExists but joins with OR instead of AND.
func (g *WhereGroup) OrWhereExists(subquery any, args ...any) *WhereGroup {
	cond, err := parseExists(true, false, subquery, args)
	if err != nil {
		g.setErr(err)
		return g
	}
	g.conds = append(g.conds, cond)
	return g
}

// WhereNotExists adds a NOT EXISTS (subquery) condition.
func (g *WhereGroup) WhereNotExists(subquery any, args ...any) *WhereGroup {
	cond, err := parseExists(false, true, subquery, args)
	if err != nil {
		g.setErr(err)
		return g
	}
	g.conds = append(g.conds, cond)
	return g
}

// OrWhereNotExists is like WhereNotExists but joins with OR instead of AND.
func (g *WhereGroup) OrWhereNotExists(subquery any, args ...any) *WhereGroup {
	cond, err := parseExists(true, true, subquery, args)
	if err != nil {
		g.setErr(err)
		return g
	}
	g.conds = append(g.conds, cond)
	return g
}

// WhereLike adds a case-sensitive pattern match: LIKE on PostgreSQL,
// LIKE BINARY on MySQL.
func (g *WhereGroup) WhereLike(column string, pattern any) *WhereGroup {
	g.conds = append(g.conds, likeCond(false, false, column, pattern))
	return g
}

// OrWhereLike is like WhereLike but joins with OR instead of AND.
func (g *WhereGroup) OrWhereLike(column string, pattern any) *WhereGroup {
	g.conds = append(g.conds, likeCond(true, false, column, pattern))
	return g
}

// WhereILike adds a case-insensitive pattern match: ILIKE on PostgreSQL,
// LIKE on MySQL (whose default collation is case-insensitive).
func (g *WhereGroup) WhereILike(column string, pattern any) *WhereGroup {
	g.conds = append(g.conds, likeCond(false, true, column, pattern))
	return g
}

// OrWhereILike is like WhereILike but joins with OR instead of AND.
func (g *WhereGroup) OrWhereILike(column string, pattern any) *WhereGroup {
	g.conds = append(g.conds, likeCond(true, true, column, pattern))
	return g
}

// parseWhere builds a condition from the variadic Where/OrWhere forms:
// (column, value) with an implied "=", (column, operator, value), or a
// single func(*WhereGroup) for a parenthesized sub-group.
func parseWhere(or bool, args []any) (dialect.Cond, error) {
	if len(args) == 1 {
		if fn, ok := args[0].(func(*WhereGroup)); ok {
			g := &WhereGroup{}
			fn(g)
			if g.err != nil {
				return dialect.Cond{}, g.err
			}
			return dialect.Cond{Kind: dialect.CondGroup, Or: or, Group: g.conds}, nil
		}
	}
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
		return dialect.Cond{}, fmt.Errorf("Where expects (column, value), (column, operator, value), or (func(*WhereGroup)), got %d args", len(args))
	}
}

// parseWhereNot is parseWhere with the condition negated: NOT "col" = $1 for
// the comparison forms, NOT (...) for the group form.
func parseWhereNot(or bool, args []any) (dialect.Cond, error) {
	cond, err := parseWhere(or, args)
	if err != nil {
		return cond, err
	}
	cond.Not = true
	return cond, nil
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

// betweenCond builds a [NOT] BETWEEN condition.
func betweenCond(or, not bool, column string, low, high any) dialect.Cond {
	return dialect.Cond{Kind: dialect.CondBetween, Or: or, Not: not, Column: column, Values: []any{low, high}}
}

// likeCond builds a LIKE pattern-match condition. insensitive selects the
// dialect's case-insensitive operator at compile time.
func likeCond(or, insensitive bool, column string, pattern any) dialect.Cond {
	return dialect.Cond{Kind: dialect.CondLike, Or: or, Insensitive: insensitive, Column: column, Value: pattern}
}

// parseExists builds an EXISTS condition from either a *SelectBuilder
// subquery or a raw SQL string with ? placeholders bound to args.
func parseExists(or, not bool, subquery any, args []any) (dialect.Cond, error) {
	switch sq := subquery.(type) {
	case *SelectBuilder:
		if len(args) > 0 {
			return dialect.Cond{}, fmt.Errorf("WhereExists args are only used with a raw SQL subquery, got %d args with a *SelectBuilder", len(args))
		}
		if sq.err != nil {
			return dialect.Cond{}, sq.err
		}
		if sq.table == "" && sq.fromSub == nil {
			return dialect.Cond{}, fmt.Errorf("WhereExists subquery has no table: call From() on it")
		}
		sub := sq.subSelect()
		return dialect.Cond{Kind: dialect.CondExists, Or: or, Not: not, Sub: &sub}, nil
	case string:
		if n := strings.Count(sq, "?"); n != len(args) {
			return dialect.Cond{}, fmt.Errorf("WhereExists subquery has %d placeholders but %d args", n, len(args))
		}
		return dialect.Cond{Kind: dialect.CondExists, Or: or, Not: not, Raw: sq, Values: args}, nil
	default:
		return dialect.Cond{}, fmt.Errorf("WhereExists expects a *SelectBuilder or a raw SQL string, got %T", subquery)
	}
}

// scanRowMaps drains rows into maps keyed by column name.
// []byte column values are converted to string.
func scanRowMaps(rows *sql.Rows) ([]map[string]any, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var result []map[string]any
	for rows.Next() {
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
		result = append(result, row)
	}
	return result, rows.Err()
}

// execReturning runs a RETURNING statement via the execer and scans the
// returned rows into maps.
func execReturning(execer Execer, sqlStr string, args []any) ([]map[string]any, error) {
	rows, err := execer.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRowMaps(rows)
}

// checkReturning validates an ExecReturning call against the dialect.
func checkReturning(d dialect.Dialect, cols []string) error {
	if len(cols) == 0 {
		return fmt.Errorf("ExecReturning requires at least one column")
	}
	if !d.SupportsReturning() {
		return fmt.Errorf("RETURNING is not supported by %s; use Exec() and LastInsertId()", d.Name())
	}
	return nil
}

// SelectBuilder builds SELECT queries.
type SelectBuilder struct {
	table      string
	fromSub    *SelectBuilder // derived-table FROM; wins over table
	alias      string         // name for this query when nested via From
	columns    []string
	distinct   bool
	distinctOn []string
	joins      []dialect.JoinClause
	where      []dialect.Cond
	groupBy    []dialect.GroupClause
	having     []dialect.Cond
	orderBy    []dialect.OrderClause
	limit      *int
	offset     *int
	dialect    dialect.Dialect
	execer     Execer
	err        error
}

// subSelect gathers the builder's state into the neutral dialect form used
// both for compilation and as an EXISTS subquery.
func (s *SelectBuilder) subSelect() dialect.SubSelect {
	sub := dialect.SubSelect{
		Table:      s.table,
		Columns:    s.columns,
		Distinct:   s.distinct,
		DistinctOn: s.distinctOn,
		Joins:      s.joins,
		Wheres:     s.where,
		GroupBys:   s.groupBy,
		Havings:    s.having,
		OrderBys:   s.orderBy,
		Limit:      s.limit,
		Offset:     s.offset,
	}
	if s.fromSub != nil {
		inner := s.fromSub.subSelect()
		sub.FromSub = &inner
		sub.FromAlias = s.fromSub.alias
	}
	return sub
}

// NewSelectBuilder creates a dialect-aware SelectBuilder (used by Schema).
func NewSelectBuilder(columns []string, d dialect.Dialect, execer Execer, err error) *SelectBuilder {
	return &SelectBuilder{columns: columns, dialect: d, execer: execer, err: err}
}

// Select starts building a SELECT query.
func Select(columns ...string) *SelectBuilder {
	return &SelectBuilder{columns: columns}
}

// From sets the source to select from: a table name string, or a
// *SelectBuilder used as a derived table — the subquery must be named with
// As() first, e.g. From(jone.Select("user_id").From("orders").As("t")).
func (s *SelectBuilder) From(table any) *SelectBuilder {
	switch t := table.(type) {
	case string:
		s.table = t
	case *SelectBuilder:
		if t.err != nil {
			s.setErr(t.err)
			return s
		}
		if t.table == "" && t.fromSub == nil {
			s.setErr(fmt.Errorf("From subquery has no table: call From() on it"))
			return s
		}
		if t.alias == "" {
			s.setErr(fmt.Errorf("From subquery requires an alias: call As() on it"))
			return s
		}
		s.fromSub = t
	default:
		s.setErr(fmt.Errorf("From expects a table name string or a *SelectBuilder subquery, got %T", table))
	}
	return s
}

// join appends a structured join clause, accepting the two ON forms.
func (s *SelectBuilder) join(kind, table string, on []string) *SelectBuilder {
	j := dialect.JoinClause{Kind: kind, Table: table, Op: "="}
	switch len(on) {
	case 2:
		j.Left, j.Right = on[0], on[1]
	case 3:
		j.Left, j.Op, j.Right = on[0], on[1], on[2]
	default:
		s.setErr(fmt.Errorf("Join expects (table, left, right) or (table, left, op, right), got %d ON args", len(on)))
		return s
	}
	s.joins = append(s.joins, j)
	return s
}

// Join adds an INNER JOIN: Join(table, left, right) for an equality ON, or
// Join(table, left, op, right) for other operators. Columns are qualified
// identifiers ("users.id"); the table may use "orders as o" aliasing.
func (s *SelectBuilder) Join(table string, on ...string) *SelectBuilder {
	return s.join("INNER JOIN", table, on)
}

// LeftJoin is Join with a LEFT JOIN.
func (s *SelectBuilder) LeftJoin(table string, on ...string) *SelectBuilder {
	return s.join("LEFT JOIN", table, on)
}

// RightJoin is Join with a RIGHT JOIN.
func (s *SelectBuilder) RightJoin(table string, on ...string) *SelectBuilder {
	return s.join("RIGHT JOIN", table, on)
}

// LeftOuterJoin is Join with a LEFT OUTER JOIN (equivalent to LeftJoin).
func (s *SelectBuilder) LeftOuterJoin(table string, on ...string) *SelectBuilder {
	return s.join("LEFT OUTER JOIN", table, on)
}

// RightOuterJoin is Join with a RIGHT OUTER JOIN (equivalent to RightJoin).
func (s *SelectBuilder) RightOuterJoin(table string, on ...string) *SelectBuilder {
	return s.join("RIGHT OUTER JOIN", table, on)
}

// FullOuterJoin is Join with a FULL OUTER JOIN. PostgreSQL only.
func (s *SelectBuilder) FullOuterJoin(table string, on ...string) *SelectBuilder {
	if s.dialect != nil && !s.dialect.SupportsFullOuterJoin() {
		s.setErr(fmt.Errorf("FULL OUTER JOIN is not supported by %s", s.dialect.Name()))
		return s
	}
	return s.join("FULL OUTER JOIN", table, on)
}

// CrossJoin adds a CROSS JOIN (cartesian product) — no ON condition.
func (s *SelectBuilder) CrossJoin(table string) *SelectBuilder {
	s.joins = append(s.joins, dialect.JoinClause{Kind: "CROSS JOIN", Table: table})
	return s
}

// JoinRaw adds a raw join fragment used verbatim, with ? placeholders bound
// to args, e.g. JoinRaw("LEFT JOIN orders o ON o.user_id = users.id AND o.total > ?", 100).
func (s *SelectBuilder) JoinRaw(raw string, args ...any) *SelectBuilder {
	if n := strings.Count(raw, "?"); n != len(args) {
		s.setErr(fmt.Errorf("JoinRaw has %d placeholders but %d args", n, len(args)))
		return s
	}
	s.joins = append(s.joins, dialect.JoinClause{Raw: raw, Values: args})
	return s
}

// As names this query for use as a derived table in an outer From().
// The alias is required by both PostgreSQL and MySQL; it has no effect when
// the query is executed directly.
func (s *SelectBuilder) As(alias string) *SelectBuilder {
	s.alias = alias
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

// WhereNot adds a negated condition: WhereNot(column, value),
// WhereNot(column, operator, value), or WhereNot(func) for NOT (...).
func (s *SelectBuilder) WhereNot(args ...any) *SelectBuilder {
	cond, err := parseWhereNot(false, args)
	if err != nil {
		s.setErr(err)
		return s
	}
	s.where = append(s.where, cond)
	return s
}

// OrWhereNot is like WhereNot but joins with OR instead of AND.
func (s *SelectBuilder) OrWhereNot(args ...any) *SelectBuilder {
	cond, err := parseWhereNot(true, args)
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

// OrWhereIn is like WhereIn but joins with OR instead of AND.
func (s *SelectBuilder) OrWhereIn(column string, values ...any) *SelectBuilder {
	s.where = append(s.where, dialect.Cond{Kind: dialect.CondIn, Or: true, Column: column, Values: inValues(values)})
	return s
}

// WhereNotIn adds a NOT IN condition.
func (s *SelectBuilder) WhereNotIn(column string, values ...any) *SelectBuilder {
	s.where = append(s.where, dialect.Cond{Kind: dialect.CondIn, Not: true, Column: column, Values: inValues(values)})
	return s
}

// OrWhereNotIn is like WhereNotIn but joins with OR instead of AND.
func (s *SelectBuilder) OrWhereNotIn(column string, values ...any) *SelectBuilder {
	s.where = append(s.where, dialect.Cond{Kind: dialect.CondIn, Or: true, Not: true, Column: column, Values: inValues(values)})
	return s
}

// WhereNull adds an IS NULL condition.
func (s *SelectBuilder) WhereNull(column string) *SelectBuilder {
	s.where = append(s.where, dialect.Cond{Kind: dialect.CondNull, Column: column})
	return s
}

// OrWhereNull is like WhereNull but joins with OR instead of AND.
func (s *SelectBuilder) OrWhereNull(column string) *SelectBuilder {
	s.where = append(s.where, dialect.Cond{Kind: dialect.CondNull, Or: true, Column: column})
	return s
}

// WhereNotNull adds an IS NOT NULL condition.
func (s *SelectBuilder) WhereNotNull(column string) *SelectBuilder {
	s.where = append(s.where, dialect.Cond{Kind: dialect.CondNull, Not: true, Column: column})
	return s
}

// OrWhereNotNull is like WhereNotNull but joins with OR instead of AND.
func (s *SelectBuilder) OrWhereNotNull(column string) *SelectBuilder {
	s.where = append(s.where, dialect.Cond{Kind: dialect.CondNull, Or: true, Not: true, Column: column})
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

// OrWhereRaw is like WhereRaw but joins with OR instead of AND.
func (s *SelectBuilder) OrWhereRaw(raw string, args ...any) *SelectBuilder {
	cond, err := parseWhereRaw(raw, args)
	if err != nil {
		s.setErr(err)
		return s
	}
	cond.Or = true
	s.where = append(s.where, cond)
	return s
}

// WhereBetween adds a BETWEEN condition: "col" BETWEEN low AND high.
func (s *SelectBuilder) WhereBetween(column string, low, high any) *SelectBuilder {
	s.where = append(s.where, betweenCond(false, false, column, low, high))
	return s
}

// OrWhereBetween is like WhereBetween but joins with OR instead of AND.
func (s *SelectBuilder) OrWhereBetween(column string, low, high any) *SelectBuilder {
	s.where = append(s.where, betweenCond(true, false, column, low, high))
	return s
}

// WhereNotBetween adds a NOT BETWEEN condition.
func (s *SelectBuilder) WhereNotBetween(column string, low, high any) *SelectBuilder {
	s.where = append(s.where, betweenCond(false, true, column, low, high))
	return s
}

// OrWhereNotBetween is like WhereNotBetween but joins with OR instead of AND.
func (s *SelectBuilder) OrWhereNotBetween(column string, low, high any) *SelectBuilder {
	s.where = append(s.where, betweenCond(true, true, column, low, high))
	return s
}

// WhereExists adds an EXISTS (subquery) condition. The subquery is either a
// *SelectBuilder or a raw SQL string with ? placeholders bound to args.
func (s *SelectBuilder) WhereExists(subquery any, args ...any) *SelectBuilder {
	cond, err := parseExists(false, false, subquery, args)
	if err != nil {
		s.setErr(err)
		return s
	}
	s.where = append(s.where, cond)
	return s
}

// OrWhereExists is like WhereExists but joins with OR instead of AND.
func (s *SelectBuilder) OrWhereExists(subquery any, args ...any) *SelectBuilder {
	cond, err := parseExists(true, false, subquery, args)
	if err != nil {
		s.setErr(err)
		return s
	}
	s.where = append(s.where, cond)
	return s
}

// WhereNotExists adds a NOT EXISTS (subquery) condition.
func (s *SelectBuilder) WhereNotExists(subquery any, args ...any) *SelectBuilder {
	cond, err := parseExists(false, true, subquery, args)
	if err != nil {
		s.setErr(err)
		return s
	}
	s.where = append(s.where, cond)
	return s
}

// OrWhereNotExists is like WhereNotExists but joins with OR instead of AND.
func (s *SelectBuilder) OrWhereNotExists(subquery any, args ...any) *SelectBuilder {
	cond, err := parseExists(true, true, subquery, args)
	if err != nil {
		s.setErr(err)
		return s
	}
	s.where = append(s.where, cond)
	return s
}

// WhereLike adds a case-sensitive pattern match: LIKE on PostgreSQL,
// LIKE BINARY on MySQL.
func (s *SelectBuilder) WhereLike(column string, pattern any) *SelectBuilder {
	s.where = append(s.where, likeCond(false, false, column, pattern))
	return s
}

// OrWhereLike is like WhereLike but joins with OR instead of AND.
func (s *SelectBuilder) OrWhereLike(column string, pattern any) *SelectBuilder {
	s.where = append(s.where, likeCond(true, false, column, pattern))
	return s
}

// WhereILike adds a case-insensitive pattern match: ILIKE on PostgreSQL,
// LIKE on MySQL (whose default collation is case-insensitive).
func (s *SelectBuilder) WhereILike(column string, pattern any) *SelectBuilder {
	s.where = append(s.where, likeCond(false, true, column, pattern))
	return s
}

// OrWhereILike is like WhereILike but joins with OR instead of AND.
func (s *SelectBuilder) OrWhereILike(column string, pattern any) *SelectBuilder {
	s.where = append(s.where, likeCond(true, true, column, pattern))
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

// Distinct makes the query SELECT DISTINCT. Columns are optional; any given
// are appended to the select list: Distinct("city", "state") selects
// distinct city/state pairs.
func (s *SelectBuilder) Distinct(columns ...string) *SelectBuilder {
	s.distinct = true
	s.columns = append(s.columns, columns...)
	return s
}

// DistinctOn makes the query SELECT DISTINCT ON (columns...), keeping the
// first row of each set of rows sharing the given columns' values (order the
// query so the row you want comes first). PostgreSQL only.
func (s *SelectBuilder) DistinctOn(columns ...string) *SelectBuilder {
	if len(columns) == 0 {
		s.setErr(fmt.Errorf("DistinctOn requires at least one column"))
		return s
	}
	if s.dialect != nil && !s.dialect.SupportsDistinctOn() {
		s.setErr(fmt.Errorf("DISTINCT ON is not supported by %s", s.dialect.Name()))
		return s
	}
	s.distinctOn = append(s.distinctOn, columns...)
	return s
}

// GroupBy adds columns to the GROUP BY clause. Columns are quoted; for
// expressions use GroupByRaw.
func (s *SelectBuilder) GroupBy(columns ...string) *SelectBuilder {
	for _, c := range columns {
		s.groupBy = append(s.groupBy, dialect.GroupClause{Column: c})
	}
	return s
}

// GroupByRaw adds a raw GROUP BY expression used verbatim,
// e.g. GroupByRaw("date_trunc('day', created_at)").
func (s *SelectBuilder) GroupByRaw(raw string) *SelectBuilder {
	s.groupBy = append(s.groupBy, dialect.GroupClause{Raw: raw})
	return s
}

// Having adds a condition filtering grouped rows, using the same argument
// forms as Where: Having(column, value), Having(column, operator, value), or
// Having(func(g *jone.WhereGroup) {...}) for a parenthesized group. Usually
// paired with GroupBy and an aggregate, e.g. Having("count", ">", 5).
func (s *SelectBuilder) Having(args ...any) *SelectBuilder {
	cond, err := parseWhere(false, args)
	if err != nil {
		s.setErr(err)
		return s
	}
	s.having = append(s.having, cond)
	return s
}

// HavingRaw adds a raw HAVING condition with ? placeholders bound to args,
// e.g. HavingRaw("COUNT(*) > ?", 5).
func (s *SelectBuilder) HavingRaw(raw string, args ...any) *SelectBuilder {
	cond, err := parseWhereRaw(raw, args)
	if err != nil {
		s.setErr(err)
		return s
	}
	s.having = append(s.having, cond)
	return s
}

// HavingIn adds an IN condition to the HAVING clause. Accepts variadic
// values or a single slice.
func (s *SelectBuilder) HavingIn(column string, values ...any) *SelectBuilder {
	s.having = append(s.having, dialect.Cond{Kind: dialect.CondIn, Column: column, Values: inValues(values)})
	return s
}

// HavingNotIn adds a NOT IN condition to the HAVING clause.
func (s *SelectBuilder) HavingNotIn(column string, values ...any) *SelectBuilder {
	s.having = append(s.having, dialect.Cond{Kind: dialect.CondIn, Not: true, Column: column, Values: inValues(values)})
	return s
}

// HavingNull adds an IS NULL condition to the HAVING clause.
func (s *SelectBuilder) HavingNull(column string) *SelectBuilder {
	s.having = append(s.having, dialect.Cond{Kind: dialect.CondNull, Column: column})
	return s
}

// HavingNotNull adds an IS NOT NULL condition to the HAVING clause.
func (s *SelectBuilder) HavingNotNull(column string) *SelectBuilder {
	s.having = append(s.having, dialect.Cond{Kind: dialect.CondNull, Not: true, Column: column})
	return s
}

// HavingBetween adds a BETWEEN condition to the HAVING clause:
// "col" BETWEEN low AND high.
func (s *SelectBuilder) HavingBetween(column string, low, high any) *SelectBuilder {
	s.having = append(s.having, betweenCond(false, false, column, low, high))
	return s
}

// HavingNotBetween adds a NOT BETWEEN condition to the HAVING clause.
func (s *SelectBuilder) HavingNotBetween(column string, low, high any) *SelectBuilder {
	s.having = append(s.having, betweenCond(false, true, column, low, high))
	return s
}

// HavingExists adds an EXISTS (subquery) condition to the HAVING clause. The
// subquery is either a *SelectBuilder or a raw SQL string with ? placeholders
// bound to args.
func (s *SelectBuilder) HavingExists(subquery any, args ...any) *SelectBuilder {
	cond, err := parseExists(false, false, subquery, args)
	if err != nil {
		s.setErr(err)
		return s
	}
	s.having = append(s.having, cond)
	return s
}

// HavingNotExists adds a NOT EXISTS (subquery) condition to the HAVING clause.
func (s *SelectBuilder) HavingNotExists(subquery any, args ...any) *SelectBuilder {
	cond, err := parseExists(false, true, subquery, args)
	if err != nil {
		s.setErr(err)
		return s
	}
	s.having = append(s.having, cond)
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
	return s.dialect.SelectSQL(s.subSelect())
}

// Exec executes the SELECT query and returns the result rows.
func (s *SelectBuilder) Exec() (*sql.Rows, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.execer == nil {
		return nil, fmt.Errorf("no database connection")
	}
	if s.table == "" && s.fromSub == nil {
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

	maps, err := scanRowMaps(rows)
	if err != nil {
		return nil, err
	}
	if len(maps) == 0 {
		return nil, sql.ErrNoRows
	}
	return maps[0], nil
}

// All executes the query and returns every matching row as a map keyed by
// column name. []byte column values are converted to string; no rows yields
// an empty result, not an error.
func (s *SelectBuilder) All() ([]map[string]any, error) {
	rows, err := s.Exec()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRowMaps(rows)
}

// aggregate compiles and runs a single-value aggregate query, scanning the
// result into dest. WHERE conditions apply; ORDER BY, LIMIT and OFFSET do not.
func (s *SelectBuilder) aggregate(fn, column string, dest any) error {
	if s.err != nil {
		return s.err
	}
	if s.execer == nil {
		return fmt.Errorf("no database connection")
	}
	if s.fromSub != nil {
		return fmt.Errorf("%s over a From() subquery is not supported", fn)
	}
	if s.table == "" {
		return fmt.Errorf("no table specified: call From() before %s()", fn)
	}
	if column == "" && fn != "COUNT" {
		return fmt.Errorf("%s requires a column", fn)
	}
	sqlStr, args := s.dialect.AggregateSQL(s.table, fn, column, s.where)
	return s.execer.QueryRow(sqlStr, args...).Scan(dest)
}

// aggregateFloat runs a numeric aggregate, returning 0 when no rows match
// (the database returns NULL).
func (s *SelectBuilder) aggregateFloat(fn, column string) (float64, error) {
	var v sql.NullFloat64
	if err := s.aggregate(fn, column, &v); err != nil {
		return 0, err
	}
	return v.Float64, nil
}

// aggregateAny runs an aggregate whose type depends on the column, returning
// nil when no rows match. []byte values are converted to string.
func (s *SelectBuilder) aggregateAny(fn, column string) (any, error) {
	var v any
	if err := s.aggregate(fn, column, &v); err != nil {
		return nil, err
	}
	if b, ok := v.([]byte); ok {
		return string(b), nil
	}
	return v, nil
}

// Count executes SELECT COUNT(*) with the builder's WHERE conditions.
func (s *SelectBuilder) Count() (int64, error) {
	var n int64
	if err := s.aggregate("COUNT", "*", &n); err != nil {
		return 0, err
	}
	return n, nil
}

// Sum executes SELECT SUM(column). Returns 0 when no rows match.
func (s *SelectBuilder) Sum(column string) (float64, error) {
	return s.aggregateFloat("SUM", column)
}

// Avg executes SELECT AVG(column). Returns 0 when no rows match.
func (s *SelectBuilder) Avg(column string) (float64, error) {
	return s.aggregateFloat("AVG", column)
}

// Min executes SELECT MIN(column). Returns nil when no rows match.
func (s *SelectBuilder) Min(column string) (any, error) {
	return s.aggregateAny("MIN", column)
}

// Max executes SELECT MAX(column). Returns nil when no rows match.
func (s *SelectBuilder) Max(column string) (any, error) {
	return s.aggregateAny("MAX", column)
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
// args carry the update data in the forms accepted by Update.
func NewUpdateBuilder(table string, args []any, d dialect.Dialect, execer Execer, err error) *UpdateBuilder {
	u := &UpdateBuilder{table: table, dialect: d, execer: execer, err: err}
	set, dataErr := parseUpdateData(args)
	if dataErr != nil {
		u.setErr(dataErr)
	}
	u.set = set
	return u
}

// Update starts building an UPDATE query. The data to set is passed directly:
// Update(table, map) for a map of columns, Update(table, struct) using db
// tags with snake_case fallback, or Update(table, column, value) for a single
// pair. The bare Update(table) form is valid when only Increment/Decrement
// follow.
func Update(table string, args ...any) *UpdateBuilder {
	return NewUpdateBuilder(table, args, nil, nil, nil)
}

// parseUpdateData normalizes Update's variadic data argument into a set map.
// The map is always freshly allocated so later Increment/Decrement calls
// never mutate a caller-supplied map.
func parseUpdateData(args []any) (map[string]any, error) {
	set := make(map[string]any)
	switch len(args) {
	case 0:
		return set, nil
	case 1:
		switch data := args[0].(type) {
		case map[string]any:
			for k, v := range data {
				set[k] = v
			}
			return set, nil
		default:
			rv := reflect.ValueOf(args[0])
			if rv.Kind() == reflect.Ptr {
				rv = rv.Elem()
			}
			if rv.Kind() != reflect.Struct {
				return set, fmt.Errorf("Update expects map[string]any, a struct, or (column, value), got %T", args[0])
			}
			m, err := structToMap(args[0])
			if err != nil {
				return set, err
			}
			return m, nil
		}
	case 2:
		col, ok := args[0].(string)
		if !ok {
			return set, fmt.Errorf("Update column must be a string, got %T", args[0])
		}
		set[col] = args[1]
		return set, nil
	default:
		return set, fmt.Errorf("Update expects (table, data), (table, column, value), or (table), got %d data args", len(args))
	}
}

// Increment sets column = column + amount. The amount defaults to 1.
func (u *UpdateBuilder) Increment(column string, amount ...int) *UpdateBuilder {
	return u.step(column, "+", amount)
}

// Decrement sets column = column - amount. The amount defaults to 1.
func (u *UpdateBuilder) Decrement(column string, amount ...int) *UpdateBuilder {
	return u.step(column, "-", amount)
}

// step records an increment/decrement as a raw set expression.
func (u *UpdateBuilder) step(column, op string, amount []int) *UpdateBuilder {
	if u.dialect == nil {
		u.setErr(fmt.Errorf("no dialect available for Increment/Decrement"))
		return u
	}
	n := 1
	if len(amount) > 0 {
		n = amount[0]
	}
	u.set[column] = types.RawExpr{Expr: fmt.Sprintf("%s %s %d", u.dialect.QuoteIdentifier(column), op, n)}
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

// WhereNot adds a negated condition: WhereNot(column, value),
// WhereNot(column, operator, value), or WhereNot(func) for NOT (...).
func (u *UpdateBuilder) WhereNot(args ...any) *UpdateBuilder {
	cond, err := parseWhereNot(false, args)
	if err != nil {
		u.setErr(err)
		return u
	}
	u.where = append(u.where, cond)
	return u
}

// OrWhereNot is like WhereNot but joins with OR instead of AND.
func (u *UpdateBuilder) OrWhereNot(args ...any) *UpdateBuilder {
	cond, err := parseWhereNot(true, args)
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

// OrWhereIn is like WhereIn but joins with OR instead of AND.
func (u *UpdateBuilder) OrWhereIn(column string, values ...any) *UpdateBuilder {
	u.where = append(u.where, dialect.Cond{Kind: dialect.CondIn, Or: true, Column: column, Values: inValues(values)})
	return u
}

// WhereNotIn adds a NOT IN condition.
func (u *UpdateBuilder) WhereNotIn(column string, values ...any) *UpdateBuilder {
	u.where = append(u.where, dialect.Cond{Kind: dialect.CondIn, Not: true, Column: column, Values: inValues(values)})
	return u
}

// OrWhereNotIn is like WhereNotIn but joins with OR instead of AND.
func (u *UpdateBuilder) OrWhereNotIn(column string, values ...any) *UpdateBuilder {
	u.where = append(u.where, dialect.Cond{Kind: dialect.CondIn, Or: true, Not: true, Column: column, Values: inValues(values)})
	return u
}

// WhereNull adds an IS NULL condition.
func (u *UpdateBuilder) WhereNull(column string) *UpdateBuilder {
	u.where = append(u.where, dialect.Cond{Kind: dialect.CondNull, Column: column})
	return u
}

// OrWhereNull is like WhereNull but joins with OR instead of AND.
func (u *UpdateBuilder) OrWhereNull(column string) *UpdateBuilder {
	u.where = append(u.where, dialect.Cond{Kind: dialect.CondNull, Or: true, Column: column})
	return u
}

// WhereNotNull adds an IS NOT NULL condition.
func (u *UpdateBuilder) WhereNotNull(column string) *UpdateBuilder {
	u.where = append(u.where, dialect.Cond{Kind: dialect.CondNull, Not: true, Column: column})
	return u
}

// OrWhereNotNull is like WhereNotNull but joins with OR instead of AND.
func (u *UpdateBuilder) OrWhereNotNull(column string) *UpdateBuilder {
	u.where = append(u.where, dialect.Cond{Kind: dialect.CondNull, Or: true, Not: true, Column: column})
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

// OrWhereRaw is like WhereRaw but joins with OR instead of AND.
func (u *UpdateBuilder) OrWhereRaw(raw string, args ...any) *UpdateBuilder {
	cond, err := parseWhereRaw(raw, args)
	if err != nil {
		u.setErr(err)
		return u
	}
	cond.Or = true
	u.where = append(u.where, cond)
	return u
}

// WhereBetween adds a BETWEEN condition: "col" BETWEEN low AND high.
func (u *UpdateBuilder) WhereBetween(column string, low, high any) *UpdateBuilder {
	u.where = append(u.where, betweenCond(false, false, column, low, high))
	return u
}

// OrWhereBetween is like WhereBetween but joins with OR instead of AND.
func (u *UpdateBuilder) OrWhereBetween(column string, low, high any) *UpdateBuilder {
	u.where = append(u.where, betweenCond(true, false, column, low, high))
	return u
}

// WhereNotBetween adds a NOT BETWEEN condition.
func (u *UpdateBuilder) WhereNotBetween(column string, low, high any) *UpdateBuilder {
	u.where = append(u.where, betweenCond(false, true, column, low, high))
	return u
}

// OrWhereNotBetween is like WhereNotBetween but joins with OR instead of AND.
func (u *UpdateBuilder) OrWhereNotBetween(column string, low, high any) *UpdateBuilder {
	u.where = append(u.where, betweenCond(true, true, column, low, high))
	return u
}

// WhereExists adds an EXISTS (subquery) condition. The subquery is either a
// *SelectBuilder or a raw SQL string with ? placeholders bound to args.
func (u *UpdateBuilder) WhereExists(subquery any, args ...any) *UpdateBuilder {
	cond, err := parseExists(false, false, subquery, args)
	if err != nil {
		u.setErr(err)
		return u
	}
	u.where = append(u.where, cond)
	return u
}

// OrWhereExists is like WhereExists but joins with OR instead of AND.
func (u *UpdateBuilder) OrWhereExists(subquery any, args ...any) *UpdateBuilder {
	cond, err := parseExists(true, false, subquery, args)
	if err != nil {
		u.setErr(err)
		return u
	}
	u.where = append(u.where, cond)
	return u
}

// WhereNotExists adds a NOT EXISTS (subquery) condition.
func (u *UpdateBuilder) WhereNotExists(subquery any, args ...any) *UpdateBuilder {
	cond, err := parseExists(false, true, subquery, args)
	if err != nil {
		u.setErr(err)
		return u
	}
	u.where = append(u.where, cond)
	return u
}

// OrWhereNotExists is like WhereNotExists but joins with OR instead of AND.
func (u *UpdateBuilder) OrWhereNotExists(subquery any, args ...any) *UpdateBuilder {
	cond, err := parseExists(true, true, subquery, args)
	if err != nil {
		u.setErr(err)
		return u
	}
	u.where = append(u.where, cond)
	return u
}

// WhereLike adds a case-sensitive pattern match: LIKE on PostgreSQL,
// LIKE BINARY on MySQL.
func (u *UpdateBuilder) WhereLike(column string, pattern any) *UpdateBuilder {
	u.where = append(u.where, likeCond(false, false, column, pattern))
	return u
}

// OrWhereLike is like WhereLike but joins with OR instead of AND.
func (u *UpdateBuilder) OrWhereLike(column string, pattern any) *UpdateBuilder {
	u.where = append(u.where, likeCond(true, false, column, pattern))
	return u
}

// WhereILike adds a case-insensitive pattern match: ILIKE on PostgreSQL,
// LIKE on MySQL (whose default collation is case-insensitive).
func (u *UpdateBuilder) WhereILike(column string, pattern any) *UpdateBuilder {
	u.where = append(u.where, likeCond(false, true, column, pattern))
	return u
}

// OrWhereILike is like WhereILike but joins with OR instead of AND.
func (u *UpdateBuilder) OrWhereILike(column string, pattern any) *UpdateBuilder {
	u.where = append(u.where, likeCond(true, true, column, pattern))
	return u
}

// ToSQL generates the UPDATE SQL.
func (u *UpdateBuilder) ToSQL() (string, []any) {
	if u.dialect == nil {
		return "", nil
	}
	return u.dialect.UpdateSQL(u.table, u.set, u.where, nil)
}

// check validates the builder state before execution.
func (u *UpdateBuilder) check() error {
	if u.err != nil {
		return u.err
	}
	if u.execer == nil {
		return fmt.Errorf("no database connection")
	}
	if len(u.set) == 0 {
		return fmt.Errorf("no values to update: pass data to Update() before Exec()")
	}
	return nil
}

// Exec executes the UPDATE query and returns the result.
func (u *UpdateBuilder) Exec() (sql.Result, error) {
	if err := u.check(); err != nil {
		return nil, err
	}
	sqlStr, args := u.ToSQL()
	return u.execer.Exec(sqlStr, args...)
}

// ExecReturning executes the UPDATE with a RETURNING clause and returns the
// affected rows. PostgreSQL only — on MySQL use Exec().
func (u *UpdateBuilder) ExecReturning(columns ...string) ([]map[string]any, error) {
	if err := u.check(); err != nil {
		return nil, err
	}
	if err := checkReturning(u.dialect, columns); err != nil {
		return nil, err
	}
	sqlStr, args := u.dialect.UpdateSQL(u.table, u.set, u.where, columns)
	return execReturning(u.execer, sqlStr, args)
}

// TruncateBuilder builds TRUNCATE TABLE statements.
type TruncateBuilder struct {
	table   string
	dialect dialect.Dialect
	execer  Execer
	err     error
}

// NewTruncateBuilder creates a dialect-aware TruncateBuilder (used by Schema).
func NewTruncateBuilder(table string, d dialect.Dialect, execer Execer, err error) *TruncateBuilder {
	return &TruncateBuilder{table: table, dialect: d, execer: execer, err: err}
}

// ToSQL generates the TRUNCATE SQL.
func (t *TruncateBuilder) ToSQL() (string, []any) {
	if t.dialect == nil {
		return "", nil
	}
	return t.dialect.TruncateSQL(t.table), nil
}

// Exec executes the TRUNCATE and returns the result.
func (t *TruncateBuilder) Exec() (sql.Result, error) {
	if t.err != nil {
		return nil, t.err
	}
	if t.execer == nil {
		return nil, fmt.Errorf("no database connection")
	}
	sqlStr, _ := t.ToSQL()
	return t.execer.Exec(sqlStr)
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

// WhereNot adds a negated condition: WhereNot(column, value),
// WhereNot(column, operator, value), or WhereNot(func) for NOT (...).
func (d *DeleteBuilder) WhereNot(args ...any) *DeleteBuilder {
	cond, err := parseWhereNot(false, args)
	if err != nil {
		d.setErr(err)
		return d
	}
	d.where = append(d.where, cond)
	return d
}

// OrWhereNot is like WhereNot but joins with OR instead of AND.
func (d *DeleteBuilder) OrWhereNot(args ...any) *DeleteBuilder {
	cond, err := parseWhereNot(true, args)
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

// OrWhereIn is like WhereIn but joins with OR instead of AND.
func (d *DeleteBuilder) OrWhereIn(column string, values ...any) *DeleteBuilder {
	d.where = append(d.where, dialect.Cond{Kind: dialect.CondIn, Or: true, Column: column, Values: inValues(values)})
	return d
}

// WhereNotIn adds a NOT IN condition.
func (d *DeleteBuilder) WhereNotIn(column string, values ...any) *DeleteBuilder {
	d.where = append(d.where, dialect.Cond{Kind: dialect.CondIn, Not: true, Column: column, Values: inValues(values)})
	return d
}

// OrWhereNotIn is like WhereNotIn but joins with OR instead of AND.
func (d *DeleteBuilder) OrWhereNotIn(column string, values ...any) *DeleteBuilder {
	d.where = append(d.where, dialect.Cond{Kind: dialect.CondIn, Or: true, Not: true, Column: column, Values: inValues(values)})
	return d
}

// WhereNull adds an IS NULL condition.
func (d *DeleteBuilder) WhereNull(column string) *DeleteBuilder {
	d.where = append(d.where, dialect.Cond{Kind: dialect.CondNull, Column: column})
	return d
}

// OrWhereNull is like WhereNull but joins with OR instead of AND.
func (d *DeleteBuilder) OrWhereNull(column string) *DeleteBuilder {
	d.where = append(d.where, dialect.Cond{Kind: dialect.CondNull, Or: true, Column: column})
	return d
}

// WhereNotNull adds an IS NOT NULL condition.
func (d *DeleteBuilder) WhereNotNull(column string) *DeleteBuilder {
	d.where = append(d.where, dialect.Cond{Kind: dialect.CondNull, Not: true, Column: column})
	return d
}

// OrWhereNotNull is like WhereNotNull but joins with OR instead of AND.
func (d *DeleteBuilder) OrWhereNotNull(column string) *DeleteBuilder {
	d.where = append(d.where, dialect.Cond{Kind: dialect.CondNull, Or: true, Not: true, Column: column})
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

// OrWhereRaw is like WhereRaw but joins with OR instead of AND.
func (d *DeleteBuilder) OrWhereRaw(raw string, args ...any) *DeleteBuilder {
	cond, err := parseWhereRaw(raw, args)
	if err != nil {
		d.setErr(err)
		return d
	}
	cond.Or = true
	d.where = append(d.where, cond)
	return d
}

// WhereBetween adds a BETWEEN condition: "col" BETWEEN low AND high.
func (d *DeleteBuilder) WhereBetween(column string, low, high any) *DeleteBuilder {
	d.where = append(d.where, betweenCond(false, false, column, low, high))
	return d
}

// OrWhereBetween is like WhereBetween but joins with OR instead of AND.
func (d *DeleteBuilder) OrWhereBetween(column string, low, high any) *DeleteBuilder {
	d.where = append(d.where, betweenCond(true, false, column, low, high))
	return d
}

// WhereNotBetween adds a NOT BETWEEN condition.
func (d *DeleteBuilder) WhereNotBetween(column string, low, high any) *DeleteBuilder {
	d.where = append(d.where, betweenCond(false, true, column, low, high))
	return d
}

// OrWhereNotBetween is like WhereNotBetween but joins with OR instead of AND.
func (d *DeleteBuilder) OrWhereNotBetween(column string, low, high any) *DeleteBuilder {
	d.where = append(d.where, betweenCond(true, true, column, low, high))
	return d
}

// WhereExists adds an EXISTS (subquery) condition. The subquery is either a
// *SelectBuilder or a raw SQL string with ? placeholders bound to args.
func (d *DeleteBuilder) WhereExists(subquery any, args ...any) *DeleteBuilder {
	cond, err := parseExists(false, false, subquery, args)
	if err != nil {
		d.setErr(err)
		return d
	}
	d.where = append(d.where, cond)
	return d
}

// OrWhereExists is like WhereExists but joins with OR instead of AND.
func (d *DeleteBuilder) OrWhereExists(subquery any, args ...any) *DeleteBuilder {
	cond, err := parseExists(true, false, subquery, args)
	if err != nil {
		d.setErr(err)
		return d
	}
	d.where = append(d.where, cond)
	return d
}

// WhereNotExists adds a NOT EXISTS (subquery) condition.
func (d *DeleteBuilder) WhereNotExists(subquery any, args ...any) *DeleteBuilder {
	cond, err := parseExists(false, true, subquery, args)
	if err != nil {
		d.setErr(err)
		return d
	}
	d.where = append(d.where, cond)
	return d
}

// OrWhereNotExists is like WhereNotExists but joins with OR instead of AND.
func (d *DeleteBuilder) OrWhereNotExists(subquery any, args ...any) *DeleteBuilder {
	cond, err := parseExists(true, true, subquery, args)
	if err != nil {
		d.setErr(err)
		return d
	}
	d.where = append(d.where, cond)
	return d
}

// WhereLike adds a case-sensitive pattern match: LIKE on PostgreSQL,
// LIKE BINARY on MySQL.
func (d *DeleteBuilder) WhereLike(column string, pattern any) *DeleteBuilder {
	d.where = append(d.where, likeCond(false, false, column, pattern))
	return d
}

// OrWhereLike is like WhereLike but joins with OR instead of AND.
func (d *DeleteBuilder) OrWhereLike(column string, pattern any) *DeleteBuilder {
	d.where = append(d.where, likeCond(true, false, column, pattern))
	return d
}

// WhereILike adds a case-insensitive pattern match: ILIKE on PostgreSQL,
// LIKE on MySQL (whose default collation is case-insensitive).
func (d *DeleteBuilder) WhereILike(column string, pattern any) *DeleteBuilder {
	d.where = append(d.where, likeCond(false, true, column, pattern))
	return d
}

// OrWhereILike is like WhereILike but joins with OR instead of AND.
func (d *DeleteBuilder) OrWhereILike(column string, pattern any) *DeleteBuilder {
	d.where = append(d.where, likeCond(true, true, column, pattern))
	return d
}

// ToSQL generates the DELETE SQL.
func (d *DeleteBuilder) ToSQL() (string, []any) {
	if d.dialect == nil {
		return "", nil
	}
	return d.dialect.DeleteSQL(d.table, d.where, nil)
}

// check validates the builder state before execution.
func (d *DeleteBuilder) check() error {
	if d.err != nil {
		return d.err
	}
	if d.execer == nil {
		return fmt.Errorf("no database connection")
	}
	return nil
}

// Exec executes the DELETE query and returns the result.
func (d *DeleteBuilder) Exec() (sql.Result, error) {
	if err := d.check(); err != nil {
		return nil, err
	}
	sqlStr, args := d.ToSQL()
	return d.execer.Exec(sqlStr, args...)
}

// ExecReturning executes the DELETE with a RETURNING clause and returns the
// deleted rows. PostgreSQL only — on MySQL use Exec().
func (d *DeleteBuilder) ExecReturning(columns ...string) ([]map[string]any, error) {
	if err := d.check(); err != nil {
		return nil, err
	}
	if err := checkReturning(d.dialect, columns); err != nil {
		return nil, err
	}
	sqlStr, args := d.dialect.DeleteSQL(d.table, d.where, columns)
	return execReturning(d.execer, sqlStr, args)
}
