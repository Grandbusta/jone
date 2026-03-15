package dialect

import "sort"

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
