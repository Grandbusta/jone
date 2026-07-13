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
)

// Cond represents a single WHERE condition built by the query builders.
type Cond struct {
	Kind   CondKind
	Or     bool   // join with OR instead of AND (ignored on the first condition)
	Not    bool   // CondIn → NOT IN, CondNull → IS NOT NULL
	Column string // quoted with QuoteIdentifier at compile time
	Op     string // comparison operator, used verbatim for CondCmp
	Value  any    // CondCmp bound value
	Values []any  // CondIn values / CondRaw args
	Raw    string // CondRaw SQL containing ? placeholders
}

// compileWheres renders the WHERE clause body (without the "WHERE " prefix)
// and its bound args. quote quotes identifiers; placeholder(n) returns the
// 1-based placeholder for param n ("$3" for postgres, "?" for mysql);
// startIdx is the count of params already emitted by the caller (e.g. UPDATE
// SET args), so placeholder numbering continues from there.
//
// Conditions are joined left-to-right with AND/OR per each condition's Or
// flag, without parenthesization — standard SQL precedence applies.
func compileWheres(conds []Cond, quote func(string) string, placeholder func(int) string, startIdx int) (string, []any) {
	var sb strings.Builder
	var args []any
	paramIdx := startIdx

	nextPlaceholder := func(v any) string {
		paramIdx++
		args = append(args, v)
		return placeholder(paramIdx)
	}

	for i, c := range conds {
		if i > 0 {
			if c.Or {
				sb.WriteString(" OR ")
			} else {
				sb.WriteString(" AND ")
			}
		}

		switch c.Kind {
		case CondCmp:
			fmt.Fprintf(&sb, "%s %s %s", quote(c.Column), c.Op, nextPlaceholder(c.Value))
		case CondIn:
			if len(c.Values) == 0 {
				// Empty IN matches nothing; empty NOT IN matches everything.
				if c.Not {
					sb.WriteString("1 = 1")
				} else {
					sb.WriteString("1 = 0")
				}
				continue
			}
			placeholders := make([]string, len(c.Values))
			for j, v := range c.Values {
				placeholders[j] = nextPlaceholder(v)
			}
			op := "IN"
			if c.Not {
				op = "NOT IN"
			}
			fmt.Fprintf(&sb, "%s %s (%s)", quote(c.Column), op, strings.Join(placeholders, ", "))
		case CondNull:
			if c.Not {
				sb.WriteString(quote(c.Column) + " IS NOT NULL")
			} else {
				sb.WriteString(quote(c.Column) + " IS NULL")
			}
		case CondRaw:
			// Rewrite each ? to the dialect placeholder, binding Values in order.
			// Every ? is treated as a placeholder, including inside string literals.
			valIdx := 0
			for _, r := range c.Raw {
				if r == '?' && valIdx < len(c.Values) {
					sb.WriteString(nextPlaceholder(c.Values[valIdx]))
					valIdx++
				} else {
					sb.WriteRune(r)
				}
			}
		}
	}

	return sb.String(), args
}
