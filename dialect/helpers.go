package dialect

import (
	"sort"
	"strings"
)

// sortedKeys returns the keys of a map sorted alphabetically.
// This ensures deterministic column ordering in generated SQL.
func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// returningClause renders a " RETURNING ..." clause with quoted columns,
// or "" when no columns are given.
func returningClause(cols []string, quote func(string) string) string {
	if len(cols) == 0 {
		return ""
	}
	quoted := make([]string, len(cols))
	for i, c := range cols {
		quoted[i] = quote(c)
	}
	return " RETURNING " + strings.Join(quoted, ", ")
}
