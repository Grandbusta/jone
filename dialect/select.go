package dialect

import (
	"fmt"
	"strings"
)

// SubSelect is a neutral description of a SELECT statement, used both as the
// body of the public SelectSQL methods and for EXISTS subqueries, where it
// compiles inline with the outer query's placeholder numbering.
type SubSelect struct {
	Table      string
	FromSub    *SubSelect // derived-table FROM: (subquery) AS FromAlias; wins over Table
	FromAlias  string     // required alias for FromSub (both dialects demand one)
	Columns    []string
	Distinct   bool     // SELECT DISTINCT
	DistinctOn []string // SELECT DISTINCT ON (cols) — PostgreSQL only; wins over Distinct
	Joins      []JoinClause
	Wheres     []Cond
	GroupBys   []GroupClause
	Havings    []Cond
	OrderBys   []OrderClause
	Limit      *int
	Offset     *int
}

// GroupClause represents a single GROUP BY term built by the query builders.
type GroupClause struct {
	Column string // quoted with QuoteIdentifier at compile time
	Raw    string // raw SQL used verbatim; takes precedence over Column
}

// compileGroupBys renders the GROUP BY clause body (without the "GROUP BY " prefix).
func compileGroupBys(groups []GroupClause, quote func(string) string) string {
	parts := make([]string, len(groups))
	for i, g := range groups {
		if g.Raw != "" {
			parts[i] = g.Raw
			continue
		}
		parts[i] = quote(g.Column)
	}
	return strings.Join(parts, ", ")
}

// selectSQL renders a SELECT statement (without the trailing semicolon, so it
// can nest inside EXISTS parentheses) shared by all dialects. startIdx is the
// count of params already emitted by the caller, so placeholder numbering
// continues from there.
func selectSQL(sub SubSelect, quote func(string) string, placeholder func(int) string, likeOp func(insensitive bool) string, startIdx int) (string, []any) {
	cols := "*"
	if len(sub.Columns) > 0 {
		quoted := make([]string, len(sub.Columns))
		for i, c := range sub.Columns {
			if c == "*" {
				quoted[i] = c
			} else {
				quoted[i] = quote(c)
			}
		}
		cols = strings.Join(quoted, ", ")
	}

	distinct := ""
	if len(sub.DistinctOn) > 0 {
		quoted := make([]string, len(sub.DistinctOn))
		for i, c := range sub.DistinctOn {
			quoted[i] = quote(c)
		}
		distinct = "DISTINCT ON (" + strings.Join(quoted, ", ") + ") "
	} else if sub.Distinct {
		distinct = "DISTINCT "
	}

	from := quoteTable(sub.Table, quote)
	var args []any
	if sub.FromSub != nil {
		body, fromArgs := selectSQL(*sub.FromSub, quote, placeholder, likeOp, startIdx)
		from = "(" + body + ") AS " + quote(sub.FromAlias)
		args = fromArgs
		startIdx += len(fromArgs)
	}

	sql := fmt.Sprintf("SELECT %s%s FROM %s", distinct, cols, from)

	if len(sub.Joins) > 0 {
		joinSQL, joinArgs := compileJoins(sub.Joins, quote, placeholder, startIdx)
		sql += " " + joinSQL
		args = append(args, joinArgs...)
		startIdx += len(joinArgs)
	}

	whereSQL, whereArgs := compileWheres(sub.Wheres, quote, placeholder, likeOp, startIdx)
	if whereSQL != "" {
		sql += " WHERE " + whereSQL
	}
	args = append(args, whereArgs...)
	startIdx += len(whereArgs)
	if len(sub.GroupBys) > 0 {
		sql += " GROUP BY " + compileGroupBys(sub.GroupBys, quote)
	}
	if len(sub.Havings) > 0 {
		havingSQL, havingArgs := compileWheres(sub.Havings, quote, placeholder, likeOp, startIdx)
		if havingSQL != "" {
			sql += " HAVING " + havingSQL
			args = append(args, havingArgs...)
		}
	}
	if len(sub.OrderBys) > 0 {
		sql += " ORDER BY " + compileOrderBys(sub.OrderBys, quote)
	}
	if sub.Limit != nil {
		sql += fmt.Sprintf(" LIMIT %d", *sub.Limit)
	}
	if sub.Offset != nil {
		sql += fmt.Sprintf(" OFFSET %d", *sub.Offset)
	}
	return sql, args
}
