package dialect

import "fmt"

// aggregateSQL renders a single-value aggregate SELECT shared by all dialects,
// e.g. SELECT COUNT(*) FROM "users" WHERE ...; A column of "" or "*" compiles
// to FN(*), anything else is quoted.
func aggregateSQL(table, fn, column string, wheres []Cond, quote func(string) string, placeholder func(int) string, likeOp func(insensitive bool) string) (string, []any) {
	expr := fn + "(*)"
	if column != "" && column != "*" {
		expr = fn + "(" + quote(column) + ")"
	}

	sql := fmt.Sprintf("SELECT %s FROM %s", expr, quote(table))

	whereSQL, args := compileWheres(wheres, quote, placeholder, likeOp, 0)
	if whereSQL != "" {
		sql += " WHERE " + whereSQL
	}
	return sql + ";", args
}
