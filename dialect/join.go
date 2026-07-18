package dialect

import (
	"regexp"
	"strings"
)

// quoteQualified quotes a possibly dot-qualified identifier segment by
// segment using quotePart, leaving * segments unquoted.
func quoteQualified(name string, quotePart func(string) string) string {
	parts := strings.Split(name, ".")
	for i, p := range parts {
		if p == "*" {
			continue
		}
		parts[i] = quotePart(p)
	}
	return strings.Join(parts, ".")
}

var asAliasRe = regexp.MustCompile(`(?i)^(.+?)\s+as\s+(.+)$`)

// quoteTable quotes a table reference, supporting "table as alias":
// "orders as o" → "orders" AS "o".
func quoteTable(name string, quote func(string) string) string {
	if m := asAliasRe.FindStringSubmatch(name); m != nil {
		return quote(m[1]) + " AS " + quote(m[2])
	}
	return quote(name)
}

// rebindRaw rewrites each ? in a raw fragment to the placeholder produced by
// next, binding values in order. Every ? is treated as a placeholder,
// including inside string literals.
func rebindRaw(raw string, values []any, next func(v any) string) string {
	var sb strings.Builder
	valIdx := 0
	for _, r := range raw {
		if r == '?' && valIdx < len(values) {
			sb.WriteString(next(values[valIdx]))
			valIdx++
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// JoinClause represents a single JOIN built by the query builders.
type JoinClause struct {
	Kind   string // "INNER JOIN", "LEFT [OUTER] JOIN", "RIGHT [OUTER] JOIN", "FULL OUTER JOIN", "CROSS JOIN"
	Table  string // join target; may use "table as alias"
	Left   string // ON Left Op Right — identifiers, quoted at compile time; empty Left means no ON (CROSS JOIN)
	Op     string
	Right  string
	Raw    string // verbatim join fragment with ? placeholders; wins over the structured form
	Values []any  // args bound to Raw's ? placeholders
}

// compileJoins renders JOIN clauses (space-separated, no leading space) and
// their bound args. startIdx is the count of params already emitted by the
// caller, so placeholder numbering continues from there.
func compileJoins(joins []JoinClause, quote func(string) string, placeholder func(int) string, startIdx int) (string, []any) {
	var args []any
	paramIdx := startIdx

	next := func(v any) string {
		paramIdx++
		args = append(args, v)
		return placeholder(paramIdx)
	}

	parts := make([]string, len(joins))
	for i, j := range joins {
		if j.Raw != "" {
			parts[i] = rebindRaw(j.Raw, j.Values, next)
			continue
		}
		parts[i] = j.Kind + " " + quoteTable(j.Table, quote)
		if j.Left != "" {
			parts[i] += " ON " + quote(j.Left) + " " + j.Op + " " + quote(j.Right)
		}
	}
	return strings.Join(parts, " "), args
}
