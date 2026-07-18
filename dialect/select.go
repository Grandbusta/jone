package dialect

import (
	"fmt"
	"strings"
)

// SubSelect is a neutral description of a SELECT statement, used both as the
// body of the public SelectSQL methods and for EXISTS subqueries, where it
// compiles inline with the outer query's placeholder numbering.
type SubSelect struct {
	CTEs       []CTE // WITH clauses, rendered (and numbered) before everything
	Table      string
	FromSub    *SubSelect // derived-table FROM: (subquery) AS FromAlias; wins over Table
	FromAlias  string     // required alias for FromSub (both dialects demand one)
	Columns    []string   // may use "col as alias"
	Distinct   bool       // SELECT DISTINCT
	DistinctOn []string   // SELECT DISTINCT ON (cols) — PostgreSQL only; wins over Distinct
	Joins      []JoinClause
	Wheres     []Cond
	GroupBys   []GroupClause
	Havings    []Cond
	Unions     []UnionClause // rendered after HAVING; ORDER BY/LIMIT apply to the combined result
	OrderBys   []OrderClause
	Limit      *int
	Offset     *int
	Lock       string // row-locking suffix, e.g. "FOR UPDATE SKIP LOCKED"; rendered last
}

// CTE is a single WITH clause: Name AS (subquery).
type CTE struct {
	Name      string
	Recursive bool       // any recursive CTE puts RECURSIVE after WITH (it applies to the whole list)
	Sub       *SubSelect // builder-form body
	Raw       string     // raw body with ? placeholders; wins over Sub
	Values    []any
}

// UnionClause combines another SELECT with UNION or UNION ALL.
type UnionClause struct {
	All    bool
	Sub    *SubSelect // builder-form query
	Raw    string     // raw query with ? placeholders; wins over Sub
	Values []any
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
// continues from there. Args accumulate in textual order: CTEs, FROM
// subquery, joins, WHERE, HAVING, unions.
func selectSQL(sub SubSelect, quote func(string) string, placeholder func(int) string, likeOp func(insensitive bool) string, startIdx int) (string, []any) {
	paramIdx := startIdx
	var args []any

	next := func(v any) string {
		paramIdx++
		args = append(args, v)
		return placeholder(paramIdx)
	}

	// subBody compiles a nested SubSelect continuing the outer numbering.
	subBody := func(s *SubSelect) string {
		body, subArgs := selectSQL(*s, quote, placeholder, likeOp, paramIdx)
		paramIdx += len(subArgs)
		args = append(args, subArgs...)
		return body
	}

	var sb strings.Builder

	if len(sub.CTEs) > 0 {
		sb.WriteString("WITH ")
		for _, c := range sub.CTEs {
			if c.Recursive {
				sb.WriteString("RECURSIVE ")
				break
			}
		}
		for i, c := range sub.CTEs {
			if i > 0 {
				sb.WriteString(", ")
			}
			body := ""
			if c.Raw != "" {
				body = rebindRaw(c.Raw, c.Values, next)
			} else {
				body = subBody(c.Sub)
			}
			sb.WriteString(quote(c.Name) + " AS (" + body + ")")
		}
		sb.WriteString(" ")
	}

	cols := "*"
	if len(sub.Columns) > 0 {
		quoted := make([]string, len(sub.Columns))
		for i, c := range sub.Columns {
			if c == "*" {
				quoted[i] = c
			} else {
				quoted[i] = quoteTable(c, quote) // handles "col as alias" too
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
	if sub.FromSub != nil {
		from = "(" + subBody(sub.FromSub) + ") AS " + quote(sub.FromAlias)
	}

	fmt.Fprintf(&sb, "SELECT %s%s FROM %s", distinct, cols, from)

	if len(sub.Joins) > 0 {
		joinSQL, joinArgs := compileJoins(sub.Joins, quote, placeholder, paramIdx)
		sb.WriteString(" " + joinSQL)
		paramIdx += len(joinArgs)
		args = append(args, joinArgs...)
	}

	whereSQL, whereArgs := compileWheres(sub.Wheres, quote, placeholder, likeOp, paramIdx)
	if whereSQL != "" {
		sb.WriteString(" WHERE " + whereSQL)
	}
	paramIdx += len(whereArgs)
	args = append(args, whereArgs...)

	if len(sub.GroupBys) > 0 {
		sb.WriteString(" GROUP BY " + compileGroupBys(sub.GroupBys, quote))
	}
	if len(sub.Havings) > 0 {
		havingSQL, havingArgs := compileWheres(sub.Havings, quote, placeholder, likeOp, paramIdx)
		if havingSQL != "" {
			sb.WriteString(" HAVING " + havingSQL)
			paramIdx += len(havingArgs)
			args = append(args, havingArgs...)
		}
	}
	for _, u := range sub.Unions {
		op := " UNION "
		if u.All {
			op = " UNION ALL "
		}
		if u.Raw != "" {
			sb.WriteString(op + rebindRaw(u.Raw, u.Values, next))
		} else {
			sb.WriteString(op + subBody(u.Sub))
		}
	}
	if len(sub.OrderBys) > 0 {
		sb.WriteString(" ORDER BY " + compileOrderBys(sub.OrderBys, quote))
	}
	if sub.Limit != nil {
		fmt.Fprintf(&sb, " LIMIT %d", *sub.Limit)
	}
	if sub.Offset != nil {
		fmt.Fprintf(&sb, " OFFSET %d", *sub.Offset)
	}
	if sub.Lock != "" {
		sb.WriteString(" " + sub.Lock)
	}
	return sb.String(), args
}
