package dialect

import (
	"fmt"
	"strings"
)

// CondKind identifies the type of a WHERE condition.
type CondKind int

const (
	// CondCmp is a comparison: Column Op Value (e.g. "age" > $1).
	CondCmp CondKind = iota
	// CondIn is an IN clause: Column IN (Values...). Not flips it to NOT IN.
	CondIn
	// CondNull is an IS NULL check. Not flips it to IS NOT NULL.
	CondNull
	// CondRaw is a raw SQL fragment with ? placeholders bound to Values.
	CondRaw
	// CondGroup is a parenthesized sub-group of conditions.
	CondGroup
	// CondBetween is a range check: Column BETWEEN Values[0] AND Values[1].
	// Not flips it to NOT BETWEEN.
	CondBetween
	// CondExists is an EXISTS (subquery) check: Sub holds a builder-form
	// subquery, or Raw/Values hold a raw subquery with ? placeholders.
	// Not flips it to NOT EXISTS.
	CondExists
	// CondLike is a pattern match: Column LIKE Value. Insensitive selects the
	// dialect's case-insensitive form (ILIKE on PostgreSQL, plain LIKE on
	// MySQL); the sensitive form is LIKE on PostgreSQL, LIKE BINARY on MySQL.
	// Not flips it to NOT ... .
	CondLike
)

// Cond represents a single WHERE condition built by the query builders.
type Cond struct {
	Kind   CondKind
	Or     bool   // join with OR instead of AND (ignored on the first condition)
	Not    bool   // CondIn → NOT IN, CondNull → IS NOT NULL, CondCmp/CondGroup → NOT ...
	Column string // quoted with QuoteIdentifier at compile time
	Op     string // comparison operator, used verbatim for CondCmp
	Value  any    // CondCmp bound value
	Values []any      // CondIn values / CondRaw args / CondBetween [low, high]
	Raw    string     // CondRaw or raw-form CondExists SQL containing ? placeholders
	Group  []Cond     // CondGroup nested conditions, compiled inside parentheses
	Sub    *SubSelect // builder-form CondExists subquery

	// Insensitive selects the dialect's case-insensitive operator for CondLike.
	Insensitive bool
}

// compileWheres renders the WHERE clause body (without the "WHERE " prefix)
// and its bound args. quote quotes identifiers; placeholder(n) returns the
// 1-based placeholder for param n ("$3" for postgres, "?" for mysql);
// likeOp returns the dialect's LIKE operator for the given case sensitivity;
// startIdx is the count of params already emitted by the caller (e.g. UPDATE
// SET args), so placeholder numbering continues from there.
//
// Conditions are joined left-to-right with AND/OR per each condition's Or
// flag. Groups compile inside parentheses; empty groups vanish (an empty
// result string means no condition survived, and the caller should omit the
// WHERE clause entirely).
func compileWheres(conds []Cond, quote func(string) string, placeholder func(int) string, likeOp func(insensitive bool) string, startIdx int) (string, []any) {
	var args []any
	paramIdx := startIdx

	nextPlaceholder := func(v any) string {
		paramIdx++
		args = append(args, v)
		return placeholder(paramIdx)
	}

	// renderRaw rewrites each ? in a raw fragment to the dialect placeholder,
	// binding values in order. Every ? is treated as a placeholder, including
	// inside string literals.
	renderRaw := func(raw string, values []any) string {
		var sb strings.Builder
		valIdx := 0
		for _, r := range raw {
			if r == '?' && valIdx < len(values) {
				sb.WriteString(nextPlaceholder(values[valIdx]))
				valIdx++
			} else {
				sb.WriteRune(r)
			}
		}
		return sb.String()
	}

	// renderCond returns the SQL fragment for one condition, or "" if it
	// compiles to nothing (an empty group).
	var render func(conds []Cond) string
	var renderCond func(c Cond) string

	renderCond = func(c Cond) string {
		switch c.Kind {
		case CondCmp:
			frag := fmt.Sprintf("%s %s %s", quote(c.Column), c.Op, nextPlaceholder(c.Value))
			if c.Not {
				return "NOT " + frag
			}
			return frag
		case CondIn:
			if len(c.Values) == 0 {
				// Empty IN matches nothing; empty NOT IN matches everything.
				if c.Not {
					return "1 = 1"
				}
				return "1 = 0"
			}
			placeholders := make([]string, len(c.Values))
			for j, v := range c.Values {
				placeholders[j] = nextPlaceholder(v)
			}
			op := "IN"
			if c.Not {
				op = "NOT IN"
			}
			return fmt.Sprintf("%s %s (%s)", quote(c.Column), op, strings.Join(placeholders, ", "))
		case CondNull:
			if c.Not {
				return quote(c.Column) + " IS NOT NULL"
			}
			return quote(c.Column) + " IS NULL"
		case CondRaw:
			return renderRaw(c.Raw, c.Values)
		case CondLike:
			frag := fmt.Sprintf("%s %s %s", quote(c.Column), likeOp(c.Insensitive), nextPlaceholder(c.Value))
			if c.Not {
				return "NOT " + frag
			}
			return frag
		case CondBetween:
			op := "BETWEEN"
			if c.Not {
				op = "NOT BETWEEN"
			}
			return fmt.Sprintf("%s %s %s AND %s", quote(c.Column), op, nextPlaceholder(c.Values[0]), nextPlaceholder(c.Values[1]))
		case CondExists:
			var body string
			if c.Sub != nil {
				var subArgs []any
				body, subArgs = selectSQL(*c.Sub, quote, placeholder, likeOp, paramIdx)
				paramIdx += len(subArgs)
				args = append(args, subArgs...)
			} else {
				body = renderRaw(c.Raw, c.Values)
			}
			if c.Not {
				return "NOT EXISTS (" + body + ")"
			}
			return "EXISTS (" + body + ")"
		case CondGroup:
			body := render(c.Group)
			if body == "" {
				return ""
			}
			if c.Not {
				return "NOT (" + body + ")"
			}
			return "(" + body + ")"
		}
		return ""
	}

	render = func(conds []Cond) string {
		var sb strings.Builder
		wrote := false
		for _, c := range conds {
			frag := renderCond(c)
			if frag == "" {
				continue
			}
			if wrote {
				if c.Or {
					sb.WriteString(" OR ")
				} else {
					sb.WriteString(" AND ")
				}
			}
			sb.WriteString(frag)
			wrote = true
		}
		return sb.String()
	}

	return render(conds), args
}
