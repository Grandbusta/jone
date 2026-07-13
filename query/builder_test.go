package query

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/Grandbusta/jone/dialect"
)

// recordingExecer records the last query and args passed to it.
type recordingExecer struct {
	query string
	args  []any
}

func (r *recordingExecer) Exec(query string, args ...any) (sql.Result, error) {
	r.query = query
	r.args = args
	return nil, nil
}

func (r *recordingExecer) Query(query string, args ...any) (*sql.Rows, error) {
	r.query = query
	r.args = args
	return nil, nil
}

func (r *recordingExecer) QueryRow(query string, args ...any) *sql.Row {
	r.query = query
	r.args = args
	return nil
}

func TestWhere_WrongArity(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	_, err := NewSelectBuilder(nil, pg, &recordingExecer{}, nil).
		From("users").Where("only-one").Exec()
	if err == nil || !strings.Contains(err.Error(), "Where expects") {
		t.Errorf("expected arity error, got %v", err)
	}

	_, err = NewSelectBuilder(nil, pg, &recordingExecer{}, nil).
		From("users").Where("a", "b", "c", "d").Exec()
	if err == nil || !strings.Contains(err.Error(), "Where expects") {
		t.Errorf("expected arity error, got %v", err)
	}
}

func TestWhere_NonStringColumnOrOp(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	_, err := NewSelectBuilder(nil, pg, &recordingExecer{}, nil).
		From("users").Where(1, 2).Exec()
	if err == nil || !strings.Contains(err.Error(), "column must be a string") {
		t.Errorf("expected column type error, got %v", err)
	}

	_, err = NewSelectBuilder(nil, pg, &recordingExecer{}, nil).
		From("users").Where("age", 1, 2).Exec()
	if err == nil || !strings.Contains(err.Error(), "operator must be a string") {
		t.Errorf("expected operator type error, got %v", err)
	}
}

func TestWhereRaw_ArgCountMismatch(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	_, err := NewDeleteBuilder("users", pg, &recordingExecer{}, nil).
		WhereRaw("a = ? AND b = ?", 1).Exec()
	if err == nil || !strings.Contains(err.Error(), "placeholders") {
		t.Errorf("expected placeholder count error, got %v", err)
	}
}

func TestDeferredConnectionErrorWins(t *testing.T) {
	pg := &dialect.PostgresDialect{}
	connErr := errors.New("connection refused")

	_, err := NewSelectBuilder(nil, pg, nil, connErr).
		From("users").Where("bad-arity").Exec()
	if err != connErr {
		t.Errorf("expected connection error to win, got %v", err)
	}
}

func TestWhereIn_SliceEqualsVariadic(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sliceSQL, sliceArgs := NewSelectBuilder(nil, pg, nil, nil).
		From("users").WhereIn("status", []string{"active", "pending"}).ToSQL()
	variadicSQL, variadicArgs := NewSelectBuilder(nil, pg, nil, nil).
		From("users").WhereIn("status", "active", "pending").ToSQL()

	if sliceSQL != variadicSQL {
		t.Errorf("slice form SQL %q != variadic form SQL %q", sliceSQL, variadicSQL)
	}
	if len(sliceArgs) != 2 || len(variadicArgs) != 2 {
		t.Errorf("args = %v and %v, want 2 each", sliceArgs, variadicArgs)
	}
}

func TestWhereGroup_BuildsParenthesizedSQL(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sql, args := NewSelectBuilder(nil, pg, nil, nil).
		From("users").
		Where("active", true).
		Where(func(g *WhereGroup) {
			g.Where("role", "admin").OrWhere("age", ">", 65)
		}).
		ToSQL()

	want := `SELECT * FROM "users" WHERE "active" = $1 AND ("role" = $2 OR "age" > $3);`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
	if len(args) != 3 || args[0] != true || args[1] != "admin" || args[2] != 65 {
		t.Errorf("args = %v, want [true admin 65]", args)
	}
}

func TestWhereGroup_Nested(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sql, _ := NewSelectBuilder(nil, pg, nil, nil).
		From("users").
		Where(func(g *WhereGroup) {
			g.Where("a", 1).
				OrWhere(func(g2 *WhereGroup) {
					g2.WhereNull("deleted_at").Where("plan", "pro")
				})
		}).
		ToSQL()

	want := `SELECT * FROM "users" WHERE ("a" = $1 OR ("deleted_at" IS NULL AND "plan" = $2));`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
}

func TestWhereGroup_OrWhereFunc(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sql, _ := NewDeleteBuilder("users", pg, nil, nil).
		Where("a", 1).
		OrWhere(func(g *WhereGroup) {
			g.Where("b", 2).Where("c", 3)
		}).
		ToSQL()

	want := `DELETE FROM "users" WHERE "a" = $1 OR ("b" = $2 AND "c" = $3);`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
}

func TestWhereGroup_ErrorPropagates(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	_, err := NewSelectBuilder(nil, pg, &recordingExecer{}, nil).
		From("users").
		Where(func(g *WhereGroup) {
			g.Where("only-one") // bad arity inside the group
		}).
		Exec()
	if err == nil || !strings.Contains(err.Error(), "Where expects") {
		t.Errorf("expected group parse error at Exec, got %v", err)
	}

	_, err = NewSelectBuilder(nil, pg, &recordingExecer{}, nil).
		From("users").
		Where(func(g *WhereGroup) {
			g.WhereRaw("a = ?", 1, 2) // mismatch inside the group
		}).
		Exec()
	if err == nil || !strings.Contains(err.Error(), "placeholders") {
		t.Errorf("expected group WhereRaw error at Exec, got %v", err)
	}
}

func TestOrderBy_InvalidDirection(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	_, err := NewSelectBuilder(nil, pg, &recordingExecer{}, nil).
		From("users").OrderBy("id", "sideways").Exec()
	if err == nil || !strings.Contains(err.Error(), "OrderBy direction") {
		t.Errorf("expected direction error, got %v", err)
	}

	_, err = NewSelectBuilder(nil, pg, &recordingExecer{}, nil).
		From("users").OrderBy("id", "asc", "desc").Exec()
	if err == nil || !strings.Contains(err.Error(), "OrderBy expects") {
		t.Errorf("expected arity error, got %v", err)
	}
}

func TestOrderBy_QuotesAndNormalizes(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sql, _ := NewSelectBuilder(nil, pg, nil, nil).
		From("users").OrderBy("created_at", "desc").OrderByRaw("lower(name)").ToSQL()
	want := `SELECT * FROM "users" ORDER BY "created_at" DESC, lower(name);`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
}

func TestDeleteExec_PassesArgs(t *testing.T) {
	pg := &dialect.PostgresDialect{}
	rec := &recordingExecer{}

	_, err := NewDeleteBuilder("users", pg, rec, nil).Where("id", 7).Exec()
	if err != nil {
		t.Fatalf("Exec() error: %v", err)
	}
	if rec.query != `DELETE FROM "users" WHERE "id" = $1;` {
		t.Errorf("query = %q", rec.query)
	}
	if len(rec.args) != 1 || rec.args[0] != 7 {
		t.Errorf("args = %v, want [7]", rec.args)
	}
}

// --- First() tests using a minimal in-memory driver ---

// stubDriver serves fixed columns/rows and records the last query.
type stubDriver struct{}

var (
	stubColumns   []string
	stubRows      [][]driver.Value
	stubLastQuery string
)

func (stubDriver) Open(string) (driver.Conn, error) { return &stubConn{}, nil }

type stubConn struct{}

func (*stubConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("not implemented") }
func (*stubConn) Close() error                        { return nil }
func (*stubConn) Begin() (driver.Tx, error)           { return nil, errors.New("not implemented") }

func (*stubConn) Query(query string, args []driver.Value) (driver.Rows, error) {
	stubLastQuery = query
	rows := make([][]driver.Value, len(stubRows))
	copy(rows, stubRows)
	return &stubResultRows{rows: rows}, nil
}

type stubResultRows struct {
	rows [][]driver.Value
	idx  int
}

func (*stubResultRows) Columns() []string { return stubColumns }
func (*stubResultRows) Close() error      { return nil }

func (r *stubResultRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.idx])
	r.idx++
	return nil
}

func init() {
	sql.Register("jone_stub", stubDriver{})
}

func TestFirst_ReturnsRowAsMap(t *testing.T) {
	db, err := sql.Open("jone_stub", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	stubColumns = []string{"id", "name"}
	stubRows = [][]driver.Value{{int64(1), []byte("John")}}

	pg := &dialect.PostgresDialect{}
	row, err := NewSelectBuilder([]string{"*"}, pg, db, nil).
		From("users").Where("id", 1).First()
	if err != nil {
		t.Fatalf("First() error: %v", err)
	}

	if row["id"] != int64(1) {
		t.Errorf("id = %v (%T), want int64(1)", row["id"], row["id"])
	}
	if row["name"] != "John" {
		t.Errorf("name = %v (%T), want string \"John\" ([]byte should be converted)", row["name"], row["name"])
	}
	if !strings.Contains(stubLastQuery, "LIMIT 1") {
		t.Errorf("query %q missing LIMIT 1", stubLastQuery)
	}
}

func TestFirst_NoRows(t *testing.T) {
	db, err := sql.Open("jone_stub", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	stubColumns = []string{"id"}
	stubRows = nil

	pg := &dialect.PostgresDialect{}
	_, err = NewSelectBuilder(nil, pg, db, nil).From("users").First()
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}
