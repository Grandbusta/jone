package query

import (
	"database/sql"
	"database/sql/driver"
	"reflect"
	"strings"
	"testing"

	"github.com/Grandbusta/jone/dialect"
)

func TestInsertToSQL(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	t.Run("map", func(t *testing.T) {
		sql, args := NewInsertBuilder(map[string]any{"email": "j@x.com", "name": "John"}, pg, nil, nil).
			Into("users").ToSQL()
		want := `INSERT INTO "users" ("email", "name") VALUES ($1, $2);`
		if sql != want {
			t.Errorf("SQL = %q, want %q", sql, want)
		}
		if !reflect.DeepEqual(args, []any{"j@x.com", "John"}) {
			t.Errorf("args = %v", args)
		}
	})

	t.Run("multi-row", func(t *testing.T) {
		sql, _ := NewInsertBuilder([]map[string]any{{"name": "a"}, {"name": "b"}}, pg, nil, nil).
			Into("users").ToSQL()
		want := `INSERT INTO "users" ("name") VALUES ($1), ($2);`
		if sql != want {
			t.Errorf("SQL = %q, want %q", sql, want)
		}
	})

	t.Run("struct", func(t *testing.T) {
		type user struct {
			Name  string `db:"name"`
			Email string `db:"email"`
		}
		sql, args := NewInsertBuilder(user{Name: "John", Email: "j@x.com"}, pg, nil, nil).
			Into("users").ToSQL()
		want := `INSERT INTO "users" ("email", "name") VALUES ($1, $2);`
		if sql != want {
			t.Errorf("SQL = %q, want %q", sql, want)
		}
		if !reflect.DeepEqual(args, []any{"j@x.com", "John"}) {
			t.Errorf("args = %v", args)
		}
	})

	t.Run("nil dialect", func(t *testing.T) {
		sql, args := NewInsertBuilder(map[string]any{"name": "x"}, nil, nil, nil).Into("users").ToSQL()
		if sql != "" || args != nil {
			t.Errorf("got (%q, %v), want empty", sql, args)
		}
	})
}

func TestExecReturning_Insert(t *testing.T) {
	db, err := sql.Open("jone_stub", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	stubColumns = []string{"id"}
	stubRows = [][]driver.Value{{int64(42)}}

	pg := &dialect.PostgresDialect{}
	rows, err := NewInsertBuilder(map[string]any{"name": "John"}, pg, db, nil).
		Into("users").ExecReturning("id")
	if err != nil {
		t.Fatalf("ExecReturning() error: %v", err)
	}

	if len(rows) != 1 || rows[0]["id"] != int64(42) {
		t.Errorf("rows = %v, want [{id: 42}]", rows)
	}
	if !strings.Contains(stubLastQuery, `RETURNING "id"`) {
		t.Errorf("query %q missing RETURNING clause", stubLastQuery)
	}
}

func TestExecReturning_UpdateAndDelete(t *testing.T) {
	db, err := sql.Open("jone_stub", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	stubColumns = []string{"id", "name"}
	stubRows = [][]driver.Value{{int64(1), []byte("John")}, {int64(2), []byte("Jane")}}

	pg := &dialect.PostgresDialect{}

	rows, err := NewUpdateBuilder("users", []any{"active", false}, pg, db, nil).
		Where("age", ">", 90).ExecReturning("id", "name")
	if err != nil {
		t.Fatalf("update ExecReturning() error: %v", err)
	}
	if len(rows) != 2 || rows[1]["name"] != "Jane" {
		t.Errorf("rows = %v, want 2 rows with []byte converted to string", rows)
	}
	if !strings.Contains(stubLastQuery, `RETURNING "id", "name"`) {
		t.Errorf("query %q missing RETURNING clause", stubLastQuery)
	}

	rows, err = NewDeleteBuilder("users", pg, db, nil).
		Where("id", 1).ExecReturning("id")
	if err != nil {
		t.Fatalf("delete ExecReturning() error: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("rows = %v, want 2", rows)
	}
	if !strings.Contains(stubLastQuery, "DELETE FROM") || !strings.Contains(stubLastQuery, `RETURNING "id"`) {
		t.Errorf("query %q missing DELETE/RETURNING", stubLastQuery)
	}
}

func TestExecReturning_Errors(t *testing.T) {
	db, err := sql.Open("jone_stub", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	pg := &dialect.PostgresDialect{}
	my := &dialect.MySQLDialect{}

	// MySQL has no RETURNING support.
	_, err = NewInsertBuilder(map[string]any{"name": "x"}, my, db, nil).
		Into("users").ExecReturning("id")
	if err == nil || !strings.Contains(err.Error(), "not supported by mysql") {
		t.Errorf("expected mysql unsupported error, got %v", err)
	}

	// At least one column is required.
	_, err = NewInsertBuilder(map[string]any{"name": "x"}, pg, db, nil).
		Into("users").ExecReturning()
	if err == nil || !strings.Contains(err.Error(), "at least one column") {
		t.Errorf("expected column-count error, got %v", err)
	}

	// Update's empty-SET guard still applies.
	_, err = NewUpdateBuilder("users", nil, pg, db, nil).Where("id", 1).ExecReturning("id")
	if err == nil || !strings.Contains(err.Error(), "no values to update") {
		t.Errorf("expected empty-SET error, got %v", err)
	}
}
