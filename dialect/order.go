package dialect

import "strings"

// OrderClause represents a single ORDER BY term built by the query builders.
type OrderClause struct {
	Column string // quoted with QuoteIdentifier at compile time
	Dir    string // "ASC", "DESC", or "" for the database default
	Raw    string // raw SQL used verbatim; takes precedence over Column
}

// compileOrderBys renders the ORDER BY clause body (without the "ORDER BY " prefix).
func compileOrderBys(orders []OrderClause, quote func(string) string) string {
	parts := make([]string, len(orders))
	for i, o := range orders {
		if o.Raw != "" {
			parts[i] = o.Raw
			continue
		}
		parts[i] = quote(o.Column)
		if o.Dir != "" {
			parts[i] += " " + o.Dir
		}
	}
	return strings.Join(parts, ", ")
}
