package dialect

import (
	"reflect"
	"testing"
)

func TestAggregateSQL(t *testing.T) {
	pg := &PostgresDialect{}
	my := &MySQLDialect{}

	tests := []struct {
		name     string
		dialect  Dialect
		fn       string
		column   string
		wheres   []Cond
		wantSQL  string
		wantArgs []any
	}{
		{
			name:    "postgres count star",
			dialect: pg,
			fn:      "COUNT",
			column:  "*",
			wantSQL: `SELECT COUNT(*) FROM "users";`,
		},
		{
			name:    "mysql count star",
			dialect: my,
			fn:      "COUNT",
			column:  "*",
			wantSQL: "SELECT COUNT(*) FROM `users`;",
		},
		{
			name:    "postgres count empty column",
			dialect: pg,
			fn:      "COUNT",
			column:  "",
			wantSQL: `SELECT COUNT(*) FROM "users";`,
		},
		{
			name:    "postgres sum quotes column",
			dialect: pg,
			fn:      "SUM",
			column:  "amount",
			wantSQL: `SELECT SUM("amount") FROM "users";`,
		},
		{
			name:    "mysql avg quotes column",
			dialect: my,
			fn:      "AVG",
			column:  "score",
			wantSQL: "SELECT AVG(`score`) FROM `users`;",
		},
		{
			name:     "postgres count with wheres rebinds placeholders",
			dialect:  pg,
			fn:       "COUNT",
			column:   "*",
			wheres:   []Cond{{Kind: CondCmp, Column: "age", Op: ">", Value: 18}, {Kind: CondCmp, Column: "active", Op: "=", Value: true}},
			wantSQL:  `SELECT COUNT(*) FROM "users" WHERE "age" > $1 AND "active" = $2;`,
			wantArgs: []any{18, true},
		},
		{
			name:     "mysql min with wheres",
			dialect:  my,
			fn:       "MIN",
			column:   "created_at",
			wheres:   []Cond{{Kind: CondCmp, Column: "status", Op: "=", Value: "active"}},
			wantSQL:  "SELECT MIN(`created_at`) FROM `users` WHERE `status` = ?;",
			wantArgs: []any{"active"},
		},
		{
			name:    "postgres max no wheres",
			dialect: pg,
			fn:      "MAX",
			column:  "id",
			wantSQL: `SELECT MAX("id") FROM "users";`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args := tt.dialect.AggregateSQL("users", tt.fn, tt.column, tt.wheres)
			if sql != tt.wantSQL {
				t.Errorf("SQL = %q, want %q", sql, tt.wantSQL)
			}
			if len(tt.wantArgs) == 0 && len(args) != 0 {
				t.Errorf("args = %v, want none", args)
			}
			if len(tt.wantArgs) > 0 && !reflect.DeepEqual(args, tt.wantArgs) {
				t.Errorf("args = %v, want %v", args, tt.wantArgs)
			}
		})
	}
}
