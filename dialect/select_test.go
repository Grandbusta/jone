package dialect

import (
	"reflect"
	"testing"
)

func TestSelectSQL_DistinctAndGroupBy(t *testing.T) {
	pg := &PostgresDialect{}
	my := &MySQLDialect{}
	ten := 10

	tests := []struct {
		name     string
		dialect  Dialect
		sub      SubSelect
		wantSQL  string
		wantArgs []any
	}{
		{
			name:    "postgres distinct star",
			dialect: pg,
			sub:     SubSelect{Table: "users", Distinct: true},
			wantSQL: `SELECT DISTINCT * FROM "users";`,
		},
		{
			name:    "mysql distinct with columns",
			dialect: my,
			sub:     SubSelect{Table: "users", Distinct: true, Columns: []string{"city", "state"}},
			wantSQL: "SELECT DISTINCT `city`, `state` FROM `users`;",
		},
		{
			name:    "postgres distinct on single column",
			dialect: pg,
			sub:     SubSelect{Table: "logins", DistinctOn: []string{"user_id"}},
			wantSQL: `SELECT DISTINCT ON ("user_id") * FROM "logins";`,
		},
		{
			name:    "postgres distinct on two columns wins over distinct flag",
			dialect: pg,
			sub:     SubSelect{Table: "logins", Distinct: true, DistinctOn: []string{"user_id", "device"}},
			wantSQL: `SELECT DISTINCT ON ("user_id", "device") * FROM "logins";`,
		},
		{
			name:    "postgres group by quoted columns",
			dialect: pg,
			sub:     SubSelect{Table: "orders", GroupBys: []GroupClause{{Column: "status"}, {Column: "region"}}},
			wantSQL: `SELECT * FROM "orders" GROUP BY "status", "region";`,
		},
		{
			name:    "mysql group by raw",
			dialect: my,
			sub:     SubSelect{Table: "orders", GroupBys: []GroupClause{{Raw: "DATE(created_at)"}}},
			wantSQL: "SELECT * FROM `orders` GROUP BY DATE(created_at);",
		},
		{
			name:    "postgres full clause order",
			dialect: pg,
			sub: SubSelect{
				Table:    "orders",
				Columns:  []string{"status"},
				Distinct: true,
				Wheres:   []Cond{{Kind: CondCmp, Column: "total", Op: ">", Value: 100}},
				GroupBys: []GroupClause{{Column: "status"}, {Raw: "DATE(created_at)"}},
				OrderBys: []OrderClause{{Column: "status", Dir: "ASC"}},
				Limit:    &ten,
			},
			wantSQL:  `SELECT DISTINCT "status" FROM "orders" WHERE "total" > $1 GROUP BY "status", DATE(created_at) ORDER BY "status" ASC LIMIT 10;`,
			wantArgs: []any{100},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args := tt.dialect.SelectSQL(tt.sub)
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

func TestSelectSQL_Having(t *testing.T) {
	pg := &PostgresDialect{}
	my := &MySQLDialect{}

	tests := []struct {
		name     string
		dialect  Dialect
		sub      SubSelect
		wantSQL  string
		wantArgs []any
	}{
		{
			name:    "postgres having after group by",
			dialect: pg,
			sub: SubSelect{
				Table:    "orders",
				Columns:  []string{"status"},
				GroupBys: []GroupClause{{Column: "status"}},
				Havings:  []Cond{{Kind: CondCmp, Column: "count", Op: ">", Value: 5}},
			},
			wantSQL:  `SELECT "status" FROM "orders" GROUP BY "status" HAVING "count" > $1;`,
			wantArgs: []any{5},
		},
		{
			name:    "mysql having after group by",
			dialect: my,
			sub: SubSelect{
				Table:    "orders",
				GroupBys: []GroupClause{{Column: "status"}},
				Havings:  []Cond{{Kind: CondCmp, Column: "total", Op: ">=", Value: 100}},
			},
			wantSQL:  "SELECT * FROM `orders` GROUP BY `status` HAVING `total` >= ?;",
			wantArgs: []any{100},
		},
		{
			name:    "postgres having numbering continues past where",
			dialect: pg,
			sub: SubSelect{
				Table:    "orders",
				Wheres:   []Cond{{Kind: CondCmp, Column: "region", Op: "=", Value: "eu"}},
				GroupBys: []GroupClause{{Column: "status"}},
				Havings: []Cond{
					{Kind: CondCmp, Column: "c", Op: ">", Value: 5},
					{Kind: CondCmp, Column: "sum", Op: "<", Value: 999},
				},
			},
			wantSQL:  `SELECT * FROM "orders" WHERE "region" = $1 GROUP BY "status" HAVING "c" > $2 AND "sum" < $3;`,
			wantArgs: []any{"eu", 5, 999},
		},
		{
			name:    "postgres having without group by",
			dialect: pg,
			sub: SubSelect{
				Table:   "orders",
				Havings: []Cond{{Kind: CondCmp, Column: "total", Op: ">", Value: 0}},
			},
			wantSQL:  `SELECT * FROM "orders" HAVING "total" > $1;`,
			wantArgs: []any{0},
		},
		{
			name:    "postgres having raw rebinds placeholders",
			dialect: pg,
			sub: SubSelect{
				Table:    "orders",
				GroupBys: []GroupClause{{Column: "status"}},
				Havings:  []Cond{{Kind: CondRaw, Raw: "COUNT(*) > ?", Values: []any{5}}},
			},
			wantSQL:  `SELECT * FROM "orders" GROUP BY "status" HAVING COUNT(*) > $1;`,
			wantArgs: []any{5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args := tt.dialect.SelectSQL(tt.sub)
			if sql != tt.wantSQL {
				t.Errorf("SQL = %q, want %q", sql, tt.wantSQL)
			}
			if !reflect.DeepEqual(args, tt.wantArgs) {
				t.Errorf("args = %v, want %v", args, tt.wantArgs)
			}
		})
	}
}
