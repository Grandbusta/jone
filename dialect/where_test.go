package dialect

import (
	"reflect"
	"testing"

	"github.com/Grandbusta/jone/types"
)

func TestSelectSQL_Wheres(t *testing.T) {
	pg := &PostgresDialect{}
	my := &MySQLDialect{}

	tests := []struct {
		name     string
		dialect  Dialect
		wheres   []Cond
		wantSQL  string
		wantArgs []any
	}{
		{
			name:     "postgres single cmp",
			dialect:  pg,
			wheres:   []Cond{{Kind: CondCmp, Column: "age", Op: ">", Value: 18}},
			wantSQL:  `SELECT * FROM "users" WHERE "age" > $1;`,
			wantArgs: []any{18},
		},
		{
			name:     "mysql single cmp",
			dialect:  my,
			wheres:   []Cond{{Kind: CondCmp, Column: "age", Op: ">", Value: 18}},
			wantSQL:  "SELECT * FROM `users` WHERE `age` > ?;",
			wantArgs: []any{18},
		},
		{
			name:    "postgres and-or joining is flat",
			dialect: pg,
			wheres: []Cond{
				{Kind: CondCmp, Column: "a", Op: "=", Value: 1},
				{Kind: CondCmp, Column: "b", Op: "=", Value: 2},
				{Kind: CondCmp, Or: true, Column: "c", Op: "=", Value: 3},
			},
			wantSQL:  `SELECT * FROM "users" WHERE "a" = $1 AND "b" = $2 OR "c" = $3;`,
			wantArgs: []any{1, 2, 3},
		},
		{
			name:     "postgres where in",
			dialect:  pg,
			wheres:   []Cond{{Kind: CondIn, Column: "status", Values: []any{"active", "pending"}}},
			wantSQL:  `SELECT * FROM "users" WHERE "status" IN ($1, $2);`,
			wantArgs: []any{"active", "pending"},
		},
		{
			name:     "mysql where not in",
			dialect:  my,
			wheres:   []Cond{{Kind: CondIn, Not: true, Column: "status", Values: []any{"a", "b"}}},
			wantSQL:  "SELECT * FROM `users` WHERE `status` NOT IN (?, ?);",
			wantArgs: []any{"a", "b"},
		},
		{
			name:     "empty in matches nothing",
			dialect:  pg,
			wheres:   []Cond{{Kind: CondIn, Column: "status", Values: nil}},
			wantSQL:  `SELECT * FROM "users" WHERE 1 = 0;`,
			wantArgs: nil,
		},
		{
			name:     "empty not in matches everything",
			dialect:  pg,
			wheres:   []Cond{{Kind: CondIn, Not: true, Column: "status", Values: nil}},
			wantSQL:  `SELECT * FROM "users" WHERE 1 = 1;`,
			wantArgs: nil,
		},
		{
			name:     "null and not null",
			dialect:  pg,
			wheres:   []Cond{{Kind: CondNull, Column: "deleted_at"}, {Kind: CondNull, Not: true, Column: "email"}},
			wantSQL:  `SELECT * FROM "users" WHERE "deleted_at" IS NULL AND "email" IS NOT NULL;`,
			wantArgs: nil,
		},
		{
			name:     "postgres raw rebinding",
			dialect:  pg,
			wheres:   []Cond{{Kind: CondRaw, Raw: "lower(email) = ? AND age > ?", Values: []any{"john@x.com", 18}}},
			wantSQL:  `SELECT * FROM "users" WHERE lower(email) = $1 AND age > $2;`,
			wantArgs: []any{"john@x.com", 18},
		},
		{
			name:     "mysql raw is identity",
			dialect:  my,
			wheres:   []Cond{{Kind: CondRaw, Raw: "lower(email) = ?", Values: []any{"john@x.com"}}},
			wantSQL:  "SELECT * FROM `users` WHERE lower(email) = ?;",
			wantArgs: []any{"john@x.com"},
		},
		{
			name:     "no wheres",
			dialect:  pg,
			wheres:   nil,
			wantSQL:  `SELECT * FROM "users";`,
			wantArgs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args := tt.dialect.SelectSQL("users", nil, tt.wheres, nil, nil, nil)
			if sql != tt.wantSQL {
				t.Errorf("SQL = %q, want %q", sql, tt.wantSQL)
			}
			if !reflect.DeepEqual(args, tt.wantArgs) {
				t.Errorf("args = %v, want %v", args, tt.wantArgs)
			}
		})
	}
}

func TestSelectSQL_Groups(t *testing.T) {
	pg := &PostgresDialect{}
	my := &MySQLDialect{}

	tests := []struct {
		name     string
		dialect  Dialect
		wheres   []Cond
		wantSQL  string
		wantArgs []any
	}{
		{
			name:    "cond and group, numbering crosses boundary",
			dialect: pg,
			wheres: []Cond{
				{Kind: CondCmp, Column: "a", Op: "=", Value: 1},
				{Kind: CondGroup, Group: []Cond{
					{Kind: CondCmp, Column: "b", Op: "=", Value: 2},
					{Kind: CondCmp, Or: true, Column: "c", Op: "=", Value: 3},
				}},
			},
			wantSQL:  `SELECT * FROM "users" WHERE "a" = $1 AND ("b" = $2 OR "c" = $3);`,
			wantArgs: []any{1, 2, 3},
		},
		{
			name:    "mysql group",
			dialect: my,
			wheres: []Cond{
				{Kind: CondCmp, Column: "a", Op: "=", Value: 1},
				{Kind: CondGroup, Group: []Cond{
					{Kind: CondCmp, Column: "b", Op: "=", Value: 2},
					{Kind: CondCmp, Or: true, Column: "c", Op: "=", Value: 3},
				}},
			},
			wantSQL:  "SELECT * FROM `users` WHERE `a` = ? AND (`b` = ? OR `c` = ?);",
			wantArgs: []any{1, 2, 3},
		},
		{
			name:    "group as first condition",
			dialect: pg,
			wheres: []Cond{
				{Kind: CondGroup, Group: []Cond{
					{Kind: CondCmp, Column: "a", Op: "=", Value: 1},
					{Kind: CondCmp, Or: true, Column: "b", Op: "=", Value: 2},
				}},
				{Kind: CondCmp, Column: "c", Op: "=", Value: 3},
			},
			wantSQL:  `SELECT * FROM "users" WHERE ("a" = $1 OR "b" = $2) AND "c" = $3;`,
			wantArgs: []any{1, 2, 3},
		},
		{
			name:    "or-joined group",
			dialect: pg,
			wheres: []Cond{
				{Kind: CondCmp, Column: "a", Op: "=", Value: 1},
				{Kind: CondGroup, Or: true, Group: []Cond{
					{Kind: CondCmp, Column: "b", Op: "=", Value: 2},
					{Kind: CondCmp, Column: "c", Op: "=", Value: 3},
				}},
			},
			wantSQL:  `SELECT * FROM "users" WHERE "a" = $1 OR ("b" = $2 AND "c" = $3);`,
			wantArgs: []any{1, 2, 3},
		},
		{
			name:    "nested group",
			dialect: pg,
			wheres: []Cond{
				{Kind: CondGroup, Group: []Cond{
					{Kind: CondCmp, Column: "a", Op: "=", Value: 1},
					{Kind: CondGroup, Or: true, Group: []Cond{
						{Kind: CondNull, Column: "deleted_at"},
						{Kind: CondCmp, Column: "b", Op: "=", Value: 2},
					}},
				}},
			},
			wantSQL:  `SELECT * FROM "users" WHERE ("a" = $1 OR ("deleted_at" IS NULL AND "b" = $2));`,
			wantArgs: []any{1, 2},
		},
		{
			name:     "empty group alone omits WHERE entirely",
			dialect:  pg,
			wheres:   []Cond{{Kind: CondGroup}},
			wantSQL:  `SELECT * FROM "users";`,
			wantArgs: nil,
		},
		{
			name:    "empty group between conds leaves no doubled connector",
			dialect: pg,
			wheres: []Cond{
				{Kind: CondCmp, Column: "a", Op: "=", Value: 1},
				{Kind: CondGroup},
				{Kind: CondCmp, Column: "b", Op: "=", Value: 2},
			},
			wantSQL:  `SELECT * FROM "users" WHERE "a" = $1 AND "b" = $2;`,
			wantArgs: []any{1, 2},
		},
		{
			name:    "group containing only empty group vanishes",
			dialect: pg,
			wheres: []Cond{
				{Kind: CondGroup, Group: []Cond{{Kind: CondGroup}}},
			},
			wantSQL:  `SELECT * FROM "users";`,
			wantArgs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args := tt.dialect.SelectSQL("users", nil, tt.wheres, nil, nil, nil)
			if sql != tt.wantSQL {
				t.Errorf("SQL = %q, want %q", sql, tt.wantSQL)
			}
			if !reflect.DeepEqual(args, tt.wantArgs) {
				t.Errorf("args = %v, want %v", args, tt.wantArgs)
			}
		})
	}
}

func TestUpdateSQL_GroupWithRawAfterSetParams(t *testing.T) {
	pg := &PostgresDialect{}

	set := map[string]any{"name": "John"}
	wheres := []Cond{
		{Kind: CondGroup, Group: []Cond{
			{Kind: CondRaw, Raw: "lower(email) = ?", Values: []any{"j@x.com"}},
			{Kind: CondCmp, Or: true, Column: "id", Op: "=", Value: 7},
		}},
	}

	sql, args := pg.UpdateSQL("users", set, wheres, nil)
	wantSQL := `UPDATE "users" SET "name" = $1 WHERE (lower(email) = $2 OR "id" = $3);`
	if sql != wantSQL {
		t.Errorf("SQL = %q, want %q", sql, wantSQL)
	}
	wantArgs := []any{"John", "j@x.com", 7}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Errorf("args = %v, want %v", args, wantArgs)
	}
}

func TestUpdateAndDeleteSQL_EmptyGroupOmitsWhere(t *testing.T) {
	pg := &PostgresDialect{}

	sql, args := pg.UpdateSQL("users", map[string]any{"name": "John"}, []Cond{{Kind: CondGroup}}, nil)
	if want := `UPDATE "users" SET "name" = $1;`; sql != want {
		t.Errorf("update SQL = %q, want %q", sql, want)
	}
	if !reflect.DeepEqual(args, []any{"John"}) {
		t.Errorf("update args = %v, want [John]", args)
	}

	sql, args = pg.DeleteSQL("users", []Cond{{Kind: CondGroup}}, nil)
	if want := `DELETE FROM "users";`; sql != want {
		t.Errorf("delete SQL = %q, want %q", sql, want)
	}
	if args != nil {
		t.Errorf("delete args = %v, want nil", args)
	}
}

func TestSelectSQL_OrderBy(t *testing.T) {
	pg := &PostgresDialect{}
	my := &MySQLDialect{}

	tests := []struct {
		name    string
		dialect Dialect
		orders  []OrderClause
		wantSQL string
	}{
		{
			name:    "postgres column with direction",
			dialect: pg,
			orders:  []OrderClause{{Column: "created_at", Dir: "DESC"}},
			wantSQL: `SELECT * FROM "users" ORDER BY "created_at" DESC;`,
		},
		{
			name:    "mysql column quoted",
			dialect: my,
			orders:  []OrderClause{{Column: "created_at", Dir: "DESC"}},
			wantSQL: "SELECT * FROM `users` ORDER BY `created_at` DESC;",
		},
		{
			name:    "no direction uses database default",
			dialect: pg,
			orders:  []OrderClause{{Column: "name"}},
			wantSQL: `SELECT * FROM "users" ORDER BY "name";`,
		},
		{
			name:    "multiple clauses and raw",
			dialect: pg,
			orders: []OrderClause{
				{Column: "age", Dir: "ASC"},
				{Raw: "lower(name) DESC"},
			},
			wantSQL: `SELECT * FROM "users" ORDER BY "age" ASC, lower(name) DESC;`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args := tt.dialect.SelectSQL("users", nil, nil, tt.orders, nil, nil)
			if sql != tt.wantSQL {
				t.Errorf("SQL = %q, want %q", sql, tt.wantSQL)
			}
			if args != nil {
				t.Errorf("args = %v, want nil", args)
			}
		})
	}
}

func TestUpdateSQL_WhereParamContinuation(t *testing.T) {
	pg := &PostgresDialect{}

	// SET params are numbered first ($1, $2); WHERE params continue ($3).
	set := map[string]any{"email": "j@x.com", "name": "John"} // sorted: email, name
	wheres := []Cond{{Kind: CondCmp, Column: "id", Op: "=", Value: 7}}

	sql, args := pg.UpdateSQL("users", set, wheres, nil)
	wantSQL := `UPDATE "users" SET "email" = $1, "name" = $2 WHERE "id" = $3;`
	if sql != wantSQL {
		t.Errorf("SQL = %q, want %q", sql, wantSQL)
	}
	wantArgs := []any{"j@x.com", "John", 7}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Errorf("args = %v, want %v", args, wantArgs)
	}
}

func TestUpdateSQL_RawExprSetSkipsParam(t *testing.T) {
	pg := &PostgresDialect{}

	// RawExpr SET values consume no param, so WHERE numbering continues
	// after only the bound SET params.
	set := map[string]any{
		"name":       "John",
		"updated_at": types.RawExpr{Expr: "CURRENT_TIMESTAMP"},
	}
	wheres := []Cond{{Kind: CondRaw, Raw: "id = ?", Values: []any{7}}}

	sql, args := pg.UpdateSQL("users", set, wheres, nil)
	wantSQL := `UPDATE "users" SET "name" = $1, "updated_at" = CURRENT_TIMESTAMP WHERE id = $2;`
	if sql != wantSQL {
		t.Errorf("SQL = %q, want %q", sql, wantSQL)
	}
	wantArgs := []any{"John", 7}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Errorf("args = %v, want %v", args, wantArgs)
	}
}

func TestUpdateSQL_MySQLWhereArgsOrdered(t *testing.T) {
	my := &MySQLDialect{}

	set := map[string]any{"name": "John"}
	wheres := []Cond{{Kind: CondCmp, Column: "id", Op: "=", Value: 7}}

	sql, args := my.UpdateSQL("users", set, wheres, nil)
	wantSQL := "UPDATE `users` SET `name` = ? WHERE `id` = ?;"
	if sql != wantSQL {
		t.Errorf("SQL = %q, want %q", sql, wantSQL)
	}
	wantArgs := []any{"John", 7}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Errorf("args = %v, want %v", args, wantArgs)
	}
}

func TestDeleteSQL_ReturnsArgs(t *testing.T) {
	pg := &PostgresDialect{}
	my := &MySQLDialect{}

	wheres := []Cond{{Kind: CondCmp, Column: "id", Op: "=", Value: 7}}

	sql, args := pg.DeleteSQL("users", wheres, nil)
	if want := `DELETE FROM "users" WHERE "id" = $1;`; sql != want {
		t.Errorf("postgres SQL = %q, want %q", sql, want)
	}
	if !reflect.DeepEqual(args, []any{7}) {
		t.Errorf("postgres args = %v, want [7]", args)
	}

	sql, args = my.DeleteSQL("users", wheres, nil)
	if want := "DELETE FROM `users` WHERE `id` = ?;"; sql != want {
		t.Errorf("mysql SQL = %q, want %q", sql, want)
	}
	if !reflect.DeepEqual(args, []any{7}) {
		t.Errorf("mysql args = %v, want [7]", args)
	}
}
