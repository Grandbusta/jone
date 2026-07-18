package query

import (
	"database/sql"
	"database/sql/driver"
	"strings"
	"testing"

	"github.com/Grandbusta/jone/dialect"
)

func TestRawQuery_ToSQL_RebindsPostgres(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sqlStr, args := NewRawBuilder("SELECT * FROM users WHERE age > ? AND status = ?", []any{21, "active"}, pg, nil, nil).
		ToSQL()
	want := "SELECT * FROM users WHERE age > $1 AND status = $2"
	if sqlStr != want {
		t.Errorf("SQL = %q, want %q", sqlStr, want)
	}
	if len(args) != 2 || args[0] != 21 || args[1] != "active" {
		t.Errorf("args = %v, want [21 active]", args)
	}
}

func TestRawQuery_ToSQL_MySQLKeepsQuestionMarks(t *testing.T) {
	my := &dialect.MySQLDialect{}

	sqlStr, args := NewRawBuilder("SELECT * FROM users WHERE age > ?", []any{21}, my, nil, nil).
		ToSQL()
	want := "SELECT * FROM users WHERE age > ?"
	if sqlStr != want {
		t.Errorf("SQL = %q, want %q", sqlStr, want)
	}
	if len(args) != 1 || args[0] != 21 {
		t.Errorf("args = %v, want [21]", args)
	}
}

func TestRawQuery_Exec_PassesReboundArgs(t *testing.T) {
	pg := &dialect.PostgresDialect{}
	rec := &recordingExecer{}

	_, err := NewRawBuilder("UPDATE users SET status = ? WHERE id = ?", []any{"legacy", 7}, pg, rec, nil).
		Exec()
	if err != nil {
		t.Fatalf("Exec() error: %v", err)
	}
	if rec.query != "UPDATE users SET status = $1 WHERE id = $2" {
		t.Errorf("query = %q", rec.query)
	}
	if len(rec.args) != 2 || rec.args[0] != "legacy" || rec.args[1] != 7 {
		t.Errorf("args = %v, want [legacy 7]", rec.args)
	}
}

func TestRawQuery_All_ReturnsRowsAsMaps(t *testing.T) {
	db, err := sql.Open("jone_stub", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	stubColumns = []string{"id", "name"}
	stubRows = [][]driver.Value{
		{int64(1), []byte("John")},
		{int64(2), []byte("Jane")},
	}

	pg := &dialect.PostgresDialect{}
	rows, err := NewRawBuilder("SELECT * FROM users WHERE age > ?", []any{21}, pg, db, nil).All()
	if err != nil {
		t.Fatalf("All() error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(rows))
	}
	if rows[0]["name"] != "John" || rows[1]["name"] != "Jane" {
		t.Errorf("rows = %v, want John and Jane", rows)
	}
	if stubLastQuery != "SELECT * FROM users WHERE age > $1" {
		t.Errorf("query = %q, want rebound placeholder", stubLastQuery)
	}
}

func TestRawQuery_Errors(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	_, err := NewRawBuilder("SELECT * FROM users WHERE a = ? AND b = ?", []any{1}, pg, &recordingExecer{}, nil).
		Exec()
	if err == nil || !strings.Contains(err.Error(), "placeholders") {
		t.Errorf("expected placeholder count error, got %v", err)
	}

	_, err = NewRawBuilder("SELECT 1", nil, pg, nil, nil).All()
	if err == nil || !strings.Contains(err.Error(), "no database connection") {
		t.Errorf("expected no-connection error, got %v", err)
	}
}
