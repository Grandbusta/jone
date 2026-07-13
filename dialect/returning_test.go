package dialect

import (
	"reflect"
	"testing"
)

func TestPostgresReturning(t *testing.T) {
	pg := &PostgresDialect{}

	t.Run("insert single column", func(t *testing.T) {
		sql, args := pg.InsertSQL("users", map[string]any{"name": "John"}, InsertOptions{Returning: []string{"id"}})
		want := `INSERT INTO "users" ("name") VALUES ($1) RETURNING "id";`
		if sql != want {
			t.Errorf("SQL = %q, want %q", sql, want)
		}
		if !reflect.DeepEqual(args, []any{"John"}) {
			t.Errorf("args = %v", args)
		}
	})

	t.Run("insert returning after on conflict", func(t *testing.T) {
		sql, _ := pg.InsertSQL("users", map[string]any{"name": "John"},
			InsertOptions{OnConflictIgnore: true, Returning: []string{"id", "created_at"}})
		want := `INSERT INTO "users" ("name") VALUES ($1) ON CONFLICT DO NOTHING RETURNING "id", "created_at";`
		if sql != want {
			t.Errorf("SQL = %q, want %q", sql, want)
		}
	})

	t.Run("insert many", func(t *testing.T) {
		sql, _ := pg.InsertManySQL("users",
			[]map[string]any{{"name": "a"}, {"name": "b"}},
			InsertOptions{Returning: []string{"id"}})
		want := `INSERT INTO "users" ("name") VALUES ($1), ($2) RETURNING "id";`
		if sql != want {
			t.Errorf("SQL = %q, want %q", sql, want)
		}
	})

	t.Run("update with where", func(t *testing.T) {
		sql, args := pg.UpdateSQL("users", map[string]any{"name": "Alice"},
			[]Cond{{Kind: CondCmp, Column: "id", Op: "=", Value: 7}},
			[]string{"id", "updated_at"})
		want := `UPDATE "users" SET "name" = $1 WHERE "id" = $2 RETURNING "id", "updated_at";`
		if sql != want {
			t.Errorf("SQL = %q, want %q", sql, want)
		}
		if !reflect.DeepEqual(args, []any{"Alice", 7}) {
			t.Errorf("args = %v", args)
		}
	})

	t.Run("delete", func(t *testing.T) {
		sql, args := pg.DeleteSQL("users",
			[]Cond{{Kind: CondCmp, Column: "id", Op: "=", Value: 7}},
			[]string{"id"})
		want := `DELETE FROM "users" WHERE "id" = $1 RETURNING "id";`
		if sql != want {
			t.Errorf("SQL = %q, want %q", sql, want)
		}
		if !reflect.DeepEqual(args, []any{7}) {
			t.Errorf("args = %v", args)
		}
	})
}

func TestMySQLReturningIgnored(t *testing.T) {
	my := &MySQLDialect{}

	if my.SupportsReturning() {
		t.Error("MySQL SupportsReturning() = true, want false")
	}

	sql, _ := my.UpdateSQL("users", map[string]any{"name": "Alice"}, nil, []string{"id"})
	want := "UPDATE `users` SET `name` = ?;"
	if sql != want {
		t.Errorf("SQL = %q, want %q (returning must be ignored)", sql, want)
	}
}

func TestPostgresSupportsReturning(t *testing.T) {
	if !(&PostgresDialect{}).SupportsReturning() {
		t.Error("Postgres SupportsReturning() = false, want true")
	}
}
