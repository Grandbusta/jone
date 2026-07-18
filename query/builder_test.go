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

func TestTruncate_BothDialects(t *testing.T) {
	pg := &dialect.PostgresDialect{}
	my := &dialect.MySQLDialect{}

	sql, args := NewTruncateBuilder("users", pg, nil, nil).ToSQL()
	if sql != `TRUNCATE TABLE "users";` || len(args) != 0 {
		t.Errorf("SQL = %q, args = %v", sql, args)
	}

	sql, _ = NewTruncateBuilder("users", my, nil, nil).ToSQL()
	if sql != "TRUNCATE TABLE `users`;" {
		t.Errorf("SQL = %q", sql)
	}
}

func TestTruncate_ExecAndErrors(t *testing.T) {
	pg := &dialect.PostgresDialect{}
	rec := &recordingExecer{}

	_, err := NewTruncateBuilder("users", pg, rec, nil).Exec()
	if err != nil {
		t.Fatalf("Exec() error: %v", err)
	}
	if rec.query != `TRUNCATE TABLE "users";` {
		t.Errorf("query = %q", rec.query)
	}

	_, err = NewTruncateBuilder("users", pg, nil, nil).Exec()
	if err == nil || !strings.Contains(err.Error(), "no database connection") {
		t.Errorf("expected no-connection error, got %v", err)
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

func TestAll_ReturnsAllRowsAsMaps(t *testing.T) {
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
	rows, err := NewSelectBuilder([]string{"*"}, pg, db, nil).
		From("users").Where("active", true).All()
	if err != nil {
		t.Fatalf("All() error: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(rows))
	}
	if rows[0]["id"] != int64(1) || rows[0]["name"] != "John" {
		t.Errorf("rows[0] = %v, want id=1 name=John", rows[0])
	}
	if rows[1]["id"] != int64(2) || rows[1]["name"] != "Jane" {
		t.Errorf("rows[1] = %v, want id=2 name=Jane", rows[1])
	}
	if strings.Contains(stubLastQuery, "LIMIT") {
		t.Errorf("query %q should not have a LIMIT", stubLastQuery)
	}
}

func TestAll_NoRows(t *testing.T) {
	db, err := sql.Open("jone_stub", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	stubColumns = []string{"id"}
	stubRows = nil

	pg := &dialect.PostgresDialect{}
	rows, err := NewSelectBuilder(nil, pg, db, nil).From("users").All()
	if err != nil {
		t.Fatalf("All() error: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("rows = %v, want none", rows)
	}
}

// --- Aggregate tests ---

func TestCount_ReturnsScalar(t *testing.T) {
	db, err := sql.Open("jone_stub", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	stubColumns = []string{"count"}
	stubRows = [][]driver.Value{{int64(42)}}

	pg := &dialect.PostgresDialect{}
	n, err := NewSelectBuilder(nil, pg, db, nil).
		From("users").Where("active", true).Count()
	if err != nil {
		t.Fatalf("Count() error: %v", err)
	}
	if n != 42 {
		t.Errorf("Count() = %d, want 42", n)
	}
	if stubLastQuery != `SELECT COUNT(*) FROM "users" WHERE "active" = $1;` {
		t.Errorf("query = %q", stubLastQuery)
	}
}

func TestSum_NullReturnsZero(t *testing.T) {
	db, err := sql.Open("jone_stub", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	stubColumns = []string{"sum"}
	stubRows = [][]driver.Value{{nil}}

	pg := &dialect.PostgresDialect{}
	total, err := NewSelectBuilder(nil, pg, db, nil).From("orders").Sum("amount")
	if err != nil {
		t.Fatalf("Sum() error: %v", err)
	}
	if total != 0 {
		t.Errorf("Sum() on NULL = %v, want 0", total)
	}
	if stubLastQuery != `SELECT SUM("amount") FROM "orders";` {
		t.Errorf("query = %q", stubLastQuery)
	}
}

func TestAvg_ReturnsFloat(t *testing.T) {
	db, err := sql.Open("jone_stub", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	stubColumns = []string{"avg"}
	stubRows = [][]driver.Value{{float64(3.5)}}

	pg := &dialect.PostgresDialect{}
	avg, err := NewSelectBuilder(nil, pg, db, nil).From("reviews").Avg("rating")
	if err != nil {
		t.Fatalf("Avg() error: %v", err)
	}
	if avg != 3.5 {
		t.Errorf("Avg() = %v, want 3.5", avg)
	}
}

func TestMin_ConvertsBytesToString(t *testing.T) {
	db, err := sql.Open("jone_stub", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	stubColumns = []string{"min"}
	stubRows = [][]driver.Value{{[]byte("alice")}}

	pg := &dialect.PostgresDialect{}
	min, err := NewSelectBuilder(nil, pg, db, nil).From("users").Min("name")
	if err != nil {
		t.Fatalf("Min() error: %v", err)
	}
	if min != "alice" {
		t.Errorf("Min() = %v (%T), want string \"alice\"", min, min)
	}
}

func TestMax_NullReturnsNil(t *testing.T) {
	db, err := sql.Open("jone_stub", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	stubColumns = []string{"max"}
	stubRows = [][]driver.Value{{nil}}

	pg := &dialect.PostgresDialect{}
	max, err := NewSelectBuilder(nil, pg, db, nil).From("users").Max("age")
	if err != nil {
		t.Fatalf("Max() error: %v", err)
	}
	if max != nil {
		t.Errorf("Max() on NULL = %v, want nil", max)
	}
}

func TestAggregate_Errors(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	_, err := NewSelectBuilder(nil, pg, &recordingExecer{}, nil).From("users").Sum("")
	if err == nil || !strings.Contains(err.Error(), "requires a column") {
		t.Errorf("expected column-required error, got %v", err)
	}

	_, err = NewSelectBuilder(nil, pg, &recordingExecer{}, nil).Count()
	if err == nil || !strings.Contains(err.Error(), "no table specified") {
		t.Errorf("expected no-table error, got %v", err)
	}

	_, err = Select().From("users").Count()
	if err == nil || !strings.Contains(err.Error(), "no database connection") {
		t.Errorf("expected no-connection error, got %v", err)
	}
}

// --- Increment / Decrement tests ---

func TestIncrement_DefaultAmount(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sql, args := NewUpdateBuilder("posts", nil, pg, nil, nil).
		Increment("views").Where("id", 10).ToSQL()
	want := `UPDATE "posts" SET "views" = "views" + 1 WHERE "id" = $1;`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
	if len(args) != 1 || args[0] != 10 {
		t.Errorf("args = %v, want [10]", args)
	}
}

func TestDecrement_ExplicitAmount_MySQL(t *testing.T) {
	my := &dialect.MySQLDialect{}

	sql, args := NewUpdateBuilder("accounts", nil, my, nil, nil).
		Decrement("balance", 50).Where("id", 3).ToSQL()
	want := "UPDATE `accounts` SET `balance` = `balance` - 50 WHERE `id` = ?;"
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
	if len(args) != 1 || args[0] != 3 {
		t.Errorf("args = %v, want [3]", args)
	}
}

func TestIncrement_MixedWithData_ParamNumbering(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sql, args := NewUpdateBuilder("posts", []any{"title", "hi"}, pg, nil, nil).
		Increment("views", 5).Where("id", 10).ToSQL()
	// set keys are sorted: title before views; the raw increment consumes no param
	want := `UPDATE "posts" SET "title" = $1, "views" = "views" + 5 WHERE "id" = $2;`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
	if len(args) != 2 || args[0] != "hi" || args[1] != 10 {
		t.Errorf("args = %v, want [hi 10]", args)
	}
}

func TestIncrement_NoDialectDefersError(t *testing.T) {
	_, err := NewUpdateBuilder("posts", nil, nil, &recordingExecer{}, nil).
		Increment("views").Exec()
	if err == nil || !strings.Contains(err.Error(), "no dialect") {
		t.Errorf("expected no-dialect error, got %v", err)
	}
}

// --- Update data-form tests ---

func TestUpdate_MapForm(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sql, args := NewUpdateBuilder("users", []any{map[string]any{"name": "Alice", "age": 30}}, pg, nil, nil).
		Where("id", 1).ToSQL()
	want := `UPDATE "users" SET "age" = $1, "name" = $2 WHERE "id" = $3;`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
	if len(args) != 3 || args[0] != 30 || args[1] != "Alice" || args[2] != 1 {
		t.Errorf("args = %v, want [30 Alice 1]", args)
	}
}

func TestUpdate_MapForm_CallerMapNotMutated(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	data := map[string]any{"title": "hi"}
	NewUpdateBuilder("posts", []any{data}, pg, nil, nil).Increment("views")
	if len(data) != 1 {
		t.Errorf("caller map mutated: %v", data)
	}
}

func TestUpdate_StructForm(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	type user struct {
		FullName string `db:"name"`
		Age      int
		Secret   string `db:"-"`
		internal bool
	}
	sql, args := NewUpdateBuilder("users", []any{user{FullName: "Alice", Secret: "x"}}, pg, nil, nil).
		Where("id", 1).ToSQL()
	// db tag, snake_case fallback, db:"-" and unexported skipped;
	// zero-valued Age is included.
	want := `UPDATE "users" SET "age" = $1, "name" = $2 WHERE "id" = $3;`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
	if len(args) != 3 || args[0] != 0 || args[1] != "Alice" || args[2] != 1 {
		t.Errorf("args = %v, want [0 Alice 1]", args)
	}
}

func TestUpdate_PairForm(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sql, args := NewUpdateBuilder("books", []any{"title", "Slaughterhouse Five"}, pg, nil, nil).
		Where("id", 42).ToSQL()
	want := `UPDATE "books" SET "title" = $1 WHERE "id" = $2;`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
	if len(args) != 2 || args[0] != "Slaughterhouse Five" || args[1] != 42 {
		t.Errorf("args = %v, want [Slaughterhouse Five 42]", args)
	}
}

func TestUpdate_DataErrors(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	_, err := NewUpdateBuilder("users", []any{42}, pg, &recordingExecer{}, nil).Exec()
	if err == nil || !strings.Contains(err.Error(), "Update expects") {
		t.Errorf("expected data type error, got %v", err)
	}

	_, err = NewUpdateBuilder("users", []any{7, "x"}, pg, &recordingExecer{}, nil).Exec()
	if err == nil || !strings.Contains(err.Error(), "column must be a string") {
		t.Errorf("expected column type error, got %v", err)
	}

	_, err = NewUpdateBuilder("users", []any{"a", 1, "b"}, pg, &recordingExecer{}, nil).Exec()
	if err == nil || !strings.Contains(err.Error(), "Update expects") {
		t.Errorf("expected arity error, got %v", err)
	}

	_, err = NewUpdateBuilder("users", nil, pg, &recordingExecer{}, nil).Exec()
	if err == nil || !strings.Contains(err.Error(), "no values to update") {
		t.Errorf("expected empty-set error, got %v", err)
	}
}

// --- WhereNot tests ---

func TestWhereNot_BuildsNegatedSQL(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sql, args := NewSelectBuilder(nil, pg, nil, nil).
		From("users").WhereNot("role", "admin").ToSQL()
	want := `SELECT * FROM "users" WHERE NOT "role" = $1;`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
	if len(args) != 1 || args[0] != "admin" {
		t.Errorf("args = %v, want [admin]", args)
	}
}

func TestWhereNot_GroupAndOrForms(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sql, args := NewSelectBuilder(nil, pg, nil, nil).
		From("users").
		Where("active", true).
		WhereNot(func(g *WhereGroup) {
			g.Where("role", "admin").OrWhereNot("age", ">", 65)
		}).
		OrWhereNot("plan", "free").
		ToSQL()
	want := `SELECT * FROM "users" WHERE "active" = $1 AND NOT ("role" = $2 OR NOT "age" > $3) OR NOT "plan" = $4;`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
	if len(args) != 4 {
		t.Errorf("args = %v, want 4 args", args)
	}
}

func TestWhereNot_OnUpdateAndDelete(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sql, _ := NewUpdateBuilder("users", []any{"active", false}, pg, nil, nil).
		WhereNot("role", "admin").ToSQL()
	want := `UPDATE "users" SET "active" = $1 WHERE NOT "role" = $2;`
	if sql != want {
		t.Errorf("update SQL = %q, want %q", sql, want)
	}

	sql, _ = NewDeleteBuilder("users", pg, nil, nil).
		WhereNot("protected", true).ToSQL()
	want = `DELETE FROM "users" WHERE NOT "protected" = $1;`
	if sql != want {
		t.Errorf("delete SQL = %q, want %q", sql, want)
	}
}

func TestWhereNot_WrongArityDefersError(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	_, err := NewSelectBuilder(nil, pg, &recordingExecer{}, nil).
		From("users").WhereNot("only-one").Exec()
	if err == nil || !strings.Contains(err.Error(), "Where expects") {
		t.Errorf("expected arity error, got %v", err)
	}
}

// --- Or-variant tests ---

func TestOrVariants_JoinWithOr(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sql, args := NewSelectBuilder(nil, pg, nil, nil).
		From("users").
		Where("active", true).
		OrWhereIn("status", "vip", "staff").
		OrWhereNotIn("plan", "free").
		OrWhereNull("deleted_at").
		OrWhereNotNull("verified_at").
		OrWhereRaw("lower(name) = ?", "john").
		ToSQL()
	want := `SELECT * FROM "users" WHERE "active" = $1` +
		` OR "status" IN ($2, $3)` +
		` OR "plan" NOT IN ($4)` +
		` OR "deleted_at" IS NULL` +
		` OR "verified_at" IS NOT NULL` +
		` OR lower(name) = $5;`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
	if len(args) != 5 {
		t.Errorf("args = %v, want 5 args", args)
	}
}

func TestOrVariants_InsideGroup(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sql, _ := NewDeleteBuilder("users", pg, nil, nil).
		Where(func(g *WhereGroup) {
			g.WhereNull("deleted_at").
				OrWhereIn("status", "banned").
				OrWhereNotNull("purged_at").
				OrWhereRaw("age < ?", 13)
		}).
		ToSQL()
	want := `DELETE FROM "users" WHERE ("deleted_at" IS NULL OR "status" IN ($1) OR "purged_at" IS NOT NULL OR age < $2);`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
}

func TestOrVariants_OnUpdate(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sql, _ := NewUpdateBuilder("users", []any{"active", false}, pg, nil, nil).
		WhereNull("confirmed_at").
		OrWhereNotIn("role", "admin", "owner").
		ToSQL()
	want := `UPDATE "users" SET "active" = $1 WHERE "confirmed_at" IS NULL OR "role" NOT IN ($2, $3);`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
}

func TestOrWhereRaw_ArgCountMismatchDefersError(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	_, err := NewSelectBuilder(nil, pg, &recordingExecer{}, nil).
		From("users").OrWhereRaw("a = ? AND b = ?", 1).Exec()
	if err == nil || !strings.Contains(err.Error(), "placeholders") {
		t.Errorf("expected placeholder count error, got %v", err)
	}
}

// --- Between / Exists tests ---

func TestWhereBetween_Variants(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sql, args := NewSelectBuilder(nil, pg, nil, nil).
		From("users").
		WhereBetween("age", 18, 65).
		OrWhereNotBetween("score", 0, 50).
		ToSQL()
	want := `SELECT * FROM "users" WHERE "age" BETWEEN $1 AND $2 OR "score" NOT BETWEEN $3 AND $4;`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
	if len(args) != 4 || args[0] != 18 || args[3] != 50 {
		t.Errorf("args = %v, want [18 65 0 50]", args)
	}
}

func TestWhereExists_SubBuilder(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sub := Select("1").From("orders").
		WhereRaw("orders.user_id = users.id").
		Where("status", "paid")
	sql, args := NewSelectBuilder(nil, pg, nil, nil).
		From("users").
		Where("active", true).
		WhereExists(sub).
		Where("plan", "pro").
		ToSQL()
	want := `SELECT * FROM "users" WHERE "active" = $1 AND EXISTS (SELECT "1" FROM "orders" WHERE orders.user_id = users.id AND "status" = $2) AND "plan" = $3;`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
	if len(args) != 3 || args[0] != true || args[1] != "paid" || args[2] != "pro" {
		t.Errorf("args = %v, want [true paid pro]", args)
	}
}

func TestWhereNotExists_RawForm(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sql, args := NewDeleteBuilder("users", pg, nil, nil).
		WhereNotExists("SELECT 1 FROM orders WHERE user_id = users.id AND status = ?", "open").
		ToSQL()
	want := `DELETE FROM "users" WHERE NOT EXISTS (SELECT 1 FROM orders WHERE user_id = users.id AND status = $1);`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
	if len(args) != 1 || args[0] != "open" {
		t.Errorf("args = %v, want [open]", args)
	}
}

func TestWhereExists_InGroupAndOrVariants(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sql, _ := NewUpdateBuilder("users", []any{"active", false}, pg, nil, nil).
		Where(func(g *WhereGroup) {
			g.WhereBetween("age", 0, 12).
				OrWhereExists(Select("1").From("bans").WhereRaw("bans.user_id = users.id"))
		}).
		OrWhereNotExists("SELECT 1 FROM payments WHERE user_id = users.id").
		ToSQL()
	want := `UPDATE "users" SET "active" = $1 WHERE ("age" BETWEEN $2 AND $3 OR EXISTS (SELECT "1" FROM "bans" WHERE bans.user_id = users.id)) OR NOT EXISTS (SELECT 1 FROM payments WHERE user_id = users.id);`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
}

func TestWhereExists_Errors(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	// raw form: placeholder/arg mismatch
	_, err := NewSelectBuilder(nil, pg, &recordingExecer{}, nil).
		From("users").WhereExists("SELECT 1 WHERE a = ? AND b = ?", 1).Exec()
	if err == nil || !strings.Contains(err.Error(), "placeholders") {
		t.Errorf("expected placeholder count error, got %v", err)
	}

	// unsupported subquery type
	_, err = NewSelectBuilder(nil, pg, &recordingExecer{}, nil).
		From("users").WhereExists(42).Exec()
	if err == nil || !strings.Contains(err.Error(), "expects a *SelectBuilder or a raw SQL string") {
		t.Errorf("expected type error, got %v", err)
	}

	// sub-builder without From()
	_, err = NewSelectBuilder(nil, pg, &recordingExecer{}, nil).
		From("users").WhereExists(Select("1")).Exec()
	if err == nil || !strings.Contains(err.Error(), "call From()") {
		t.Errorf("expected missing-From error, got %v", err)
	}

	// args passed alongside a builder subquery
	_, err = NewSelectBuilder(nil, pg, &recordingExecer{}, nil).
		From("users").WhereExists(Select("1").From("orders"), "stray").Exec()
	if err == nil || !strings.Contains(err.Error(), "raw SQL subquery") {
		t.Errorf("expected stray-args error, got %v", err)
	}

	// deferred error inside the sub-builder propagates
	_, err = NewSelectBuilder(nil, pg, &recordingExecer{}, nil).
		From("users").WhereExists(Select("1").From("orders").Where("bad-arity")).Exec()
	if err == nil || !strings.Contains(err.Error(), "Where expects") {
		t.Errorf("expected propagated sub-builder error, got %v", err)
	}
}

// --- Like / ILike tests ---

func TestWhereLike_Variants(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sql, args := NewSelectBuilder(nil, pg, nil, nil).
		From("users").
		WhereLike("name", "Jo%").
		OrWhereILike("email", "%@x.com").
		ToSQL()
	want := `SELECT * FROM "users" WHERE "name" LIKE $1 OR "email" ILIKE $2;`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
	if len(args) != 2 || args[0] != "Jo%" || args[1] != "%@x.com" {
		t.Errorf("args = %v, want [Jo%% %%@x.com]", args)
	}
}

func TestWhereILike_MySQLAndGroup(t *testing.T) {
	my := &dialect.MySQLDialect{}

	sql, _ := NewDeleteBuilder("users", my, nil, nil).
		Where(func(g *WhereGroup) {
			g.WhereILike("name", "spam%").OrWhereLike("email", "Bot@%")
		}).
		ToSQL()
	want := "DELETE FROM `users` WHERE (`name` LIKE ? OR `email` LIKE BINARY ?);"
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
}

func TestWhereLike_OnUpdateParamNumbering(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sql, args := NewUpdateBuilder("users", []any{"flagged", true}, pg, nil, nil).
		WhereILike("bio", "%crypto%").
		ToSQL()
	want := `UPDATE "users" SET "flagged" = $1 WHERE "bio" ILIKE $2;`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
	if len(args) != 2 || args[1] != "%crypto%" {
		t.Errorf("args = %v, want [true %%crypto%%]", args)
	}
}

// --- Distinct / GroupBy tests ---

func TestDistinctAndGroupBy_Chain(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sql, args := NewSelectBuilder(nil, pg, nil, nil).
		From("orders").
		Distinct().
		Where("total", ">", 100).
		GroupBy("status", "region").
		GroupByRaw("DATE(created_at)").
		OrderBy("status").
		ToSQL()
	want := `SELECT DISTINCT * FROM "orders" WHERE "total" > $1 GROUP BY "status", "region", DATE(created_at) ORDER BY "status";`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
	if len(args) != 1 || args[0] != 100 {
		t.Errorf("args = %v, want [100]", args)
	}
}

func TestDistinct_WithColumnsSetsSelectList(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sql, _ := NewSelectBuilder(nil, pg, nil, nil).
		From("users").Distinct("city", "state").ToSQL()
	want := `SELECT DISTINCT "city", "state" FROM "users";`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
}

func TestDistinctOn_Postgres(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sql, _ := NewSelectBuilder(nil, pg, nil, nil).
		From("logins").
		DistinctOn("user_id").
		OrderBy("user_id").
		OrderBy("created_at", "desc").
		ToSQL()
	want := `SELECT DISTINCT ON ("user_id") * FROM "logins" ORDER BY "user_id", "created_at" DESC;`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
}

func TestDistinctOn_Errors(t *testing.T) {
	my := &dialect.MySQLDialect{}
	pg := &dialect.PostgresDialect{}

	_, err := NewSelectBuilder(nil, my, &recordingExecer{}, nil).
		From("logins").DistinctOn("user_id").Exec()
	if err == nil || !strings.Contains(err.Error(), "not supported by mysql") {
		t.Errorf("expected mysql unsupported error, got %v", err)
	}

	_, err = NewSelectBuilder(nil, pg, &recordingExecer{}, nil).
		From("logins").DistinctOn().Exec()
	if err == nil || !strings.Contains(err.Error(), "at least one column") {
		t.Errorf("expected no-columns error, got %v", err)
	}
}

func TestWhereExists_SubqueryCarriesGroupBy(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sql, _ := NewSelectBuilder(nil, pg, nil, nil).
		From("users").
		WhereExists(
			Select("user_id").From("orders").
				WhereRaw("orders.user_id = users.id").
				GroupBy("user_id"),
		).
		ToSQL()
	want := `SELECT * FROM "users" WHERE EXISTS (SELECT "user_id" FROM "orders" WHERE orders.user_id = users.id GROUP BY "user_id");`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
}

// --- Join tests ---

func TestJoin_InnerWithAliasAndQualifiedColumns(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sql, args := NewSelectBuilder([]string{"users.name", "o.total"}, pg, nil, nil).
		From("users").
		Join("orders as o", "users.id", "o.user_id").
		Where("o.total", ">", 100).
		ToSQL()
	want := `SELECT "users"."name", "o"."total" FROM "users" INNER JOIN "orders" AS "o" ON "users"."id" = "o"."user_id" WHERE "o"."total" > $1;`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
	if len(args) != 1 || args[0] != 100 {
		t.Errorf("args = %v, want [100]", args)
	}
}

func TestLeftJoin_OperatorForm_MySQL(t *testing.T) {
	my := &dialect.MySQLDialect{}

	sql, _ := NewSelectBuilder([]string{"users.*"}, my, nil, nil).
		From("users").
		LeftJoin("profiles", "users.id", "=", "profiles.user_id").
		ToSQL()
	want := "SELECT `users`.* FROM `users` LEFT JOIN `profiles` ON `users`.`id` = `profiles`.`user_id`;"
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
}

func TestRightJoin_And_SelfJoinAliases(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sql, _ := NewSelectBuilder([]string{"e.name", "m.name"}, pg, nil, nil).
		From("employees as e").
		RightJoin("employees as m", "e.manager_id", "m.id").
		ToSQL()
	want := `SELECT "e"."name", "m"."name" FROM "employees" AS "e" RIGHT JOIN "employees" AS "m" ON "e"."manager_id" = "m"."id";`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
}

func TestJoinRaw_ArgsNumberBeforeWhere(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sql, args := NewSelectBuilder(nil, pg, nil, nil).
		From("users").
		JoinRaw("LEFT JOIN orders o ON o.user_id = users.id AND o.total > ?", 100).
		Where("users.active", true).
		ToSQL()
	want := `SELECT * FROM "users" LEFT JOIN orders o ON o.user_id = users.id AND o.total > $1 WHERE "users"."active" = $2;`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
	if len(args) != 2 || args[0] != 100 || args[1] != true {
		t.Errorf("args = %v, want [100 true]", args)
	}
}

func TestJoin_OnDerivedTable(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sub := Select("user_id").From("orders").GroupBy("user_id").As("t")
	sql, _ := NewSelectBuilder(nil, pg, nil, nil).
		From(sub).
		Join("users", "t.user_id", "users.id").
		ToSQL()
	want := `SELECT * FROM (SELECT "user_id" FROM "orders" GROUP BY "user_id") AS "t" INNER JOIN "users" ON "t"."user_id" = "users"."id";`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
}

func TestOuterJoinVariants(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sql, _ := NewSelectBuilder(nil, pg, nil, nil).
		From("users").
		LeftOuterJoin("orders", "users.id", "orders.user_id").
		RightOuterJoin("profiles", "users.id", "profiles.user_id").
		FullOuterJoin("audits", "users.id", "!=", "audits.user_id").
		ToSQL()
	want := `SELECT * FROM "users" LEFT OUTER JOIN "orders" ON "users"."id" = "orders"."user_id" RIGHT OUTER JOIN "profiles" ON "users"."id" = "profiles"."user_id" FULL OUTER JOIN "audits" ON "users"."id" != "audits"."user_id";`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
}

func TestCrossJoin(t *testing.T) {
	my := &dialect.MySQLDialect{}

	sql, args := NewSelectBuilder(nil, my, nil, nil).
		From("sizes").
		CrossJoin("colors as c").
		ToSQL()
	want := "SELECT * FROM `sizes` CROSS JOIN `colors` AS `c`;"
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
	if len(args) != 0 {
		t.Errorf("args = %v, want none", args)
	}
}

func TestFullOuterJoin_MySQLUnsupported(t *testing.T) {
	my := &dialect.MySQLDialect{}

	_, err := NewSelectBuilder(nil, my, &recordingExecer{}, nil).
		From("users").
		FullOuterJoin("orders", "users.id", "orders.user_id").
		Exec()
	if err == nil || !strings.Contains(err.Error(), "FULL OUTER JOIN is not supported") {
		t.Errorf("expected unsupported error, got %v", err)
	}
}

func TestJoin_Errors(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	_, err := NewSelectBuilder(nil, pg, &recordingExecer{}, nil).
		From("users").Join("orders", "users.id").Exec()
	if err == nil || !strings.Contains(err.Error(), "Join expects") {
		t.Errorf("expected ON arity error, got %v", err)
	}

	_, err = NewSelectBuilder(nil, pg, &recordingExecer{}, nil).
		From("users").JoinRaw("JOIN x ON a = ? AND b = ?", 1).Exec()
	if err == nil || !strings.Contains(err.Error(), "placeholders") {
		t.Errorf("expected placeholder count error, got %v", err)
	}
}

// --- From subquery / As tests ---

func TestFromSubquery_Postgres_ParamNumbering(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sub := Select("user_id").From("orders").Where("status", "paid").GroupBy("user_id").As("t")
	sql, args := NewSelectBuilder(nil, pg, nil, nil).
		From(sub).
		Where("user_id", ">", 10).
		ToSQL()
	want := `SELECT * FROM (SELECT "user_id" FROM "orders" WHERE "status" = $1 GROUP BY "user_id") AS "t" WHERE "user_id" > $2;`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
	if len(args) != 2 || args[0] != "paid" || args[1] != 10 {
		t.Errorf("args = %v, want [paid 10]", args)
	}
}

func TestFromSubquery_MySQL(t *testing.T) {
	my := &dialect.MySQLDialect{}

	sub := Select("user_id").From("orders").Where("total", ">", 100).As("big")
	sql, args := NewSelectBuilder([]string{"user_id"}, my, nil, nil).
		From(sub).
		ToSQL()
	want := "SELECT `user_id` FROM (SELECT `user_id` FROM `orders` WHERE `total` > ?) AS `big`;"
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
	if len(args) != 1 || args[0] != 100 {
		t.Errorf("args = %v, want [100]", args)
	}
}

func TestFromSubquery_HavingNumberingAfterFromArgs(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sub := Select("user_id").From("orders").Where("status", "paid").As("t")
	sql, args := NewSelectBuilder([]string{"user_id"}, pg, nil, nil).
		From(sub).
		GroupBy("user_id").
		Having("count", ">", 5).
		ToSQL()
	want := `SELECT "user_id" FROM (SELECT "user_id" FROM "orders" WHERE "status" = $1) AS "t" GROUP BY "user_id" HAVING "count" > $2;`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
	if len(args) != 2 || args[0] != "paid" || args[1] != 5 {
		t.Errorf("args = %v, want [paid 5]", args)
	}
}

func TestFromSubquery_Errors(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	_, err := NewSelectBuilder(nil, pg, &recordingExecer{}, nil).
		From(Select("1").From("orders")).Exec() // no As()
	if err == nil || !strings.Contains(err.Error(), "requires an alias") {
		t.Errorf("expected missing-alias error, got %v", err)
	}

	_, err = NewSelectBuilder(nil, pg, &recordingExecer{}, nil).
		From(Select("1").As("t")).Exec() // subquery has no table
	if err == nil || !strings.Contains(err.Error(), "has no table") {
		t.Errorf("expected no-table error, got %v", err)
	}

	_, err = NewSelectBuilder(nil, pg, &recordingExecer{}, nil).
		From(42).Exec()
	if err == nil || !strings.Contains(err.Error(), "From expects") {
		t.Errorf("expected type error, got %v", err)
	}

	sub := Select("user_id").From("orders").As("t")
	_, err = NewSelectBuilder(nil, pg, &recordingExecer{}, nil).
		From(sub).Count()
	if err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Errorf("expected aggregate-unsupported error, got %v", err)
	}
}

// --- Having tests ---

func TestHaving_ComparisonForm(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sql, args := NewSelectBuilder([]string{"status"}, pg, nil, nil).
		From("orders").
		GroupBy("status").
		Having("count", ">", 5).
		ToSQL()
	want := `SELECT "status" FROM "orders" GROUP BY "status" HAVING "count" > $1;`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
	if len(args) != 1 || args[0] != 5 {
		t.Errorf("args = %v, want [5]", args)
	}
}

func TestHavingRaw_And_FullChainNumbering(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sql, args := NewSelectBuilder([]string{"status"}, pg, nil, nil).
		From("orders").
		Where("region", "eu").
		GroupBy("status").
		Having("count", ">", 5).
		HavingRaw("SUM(total) < ?", 999).
		OrderBy("status").
		ToSQL()
	want := `SELECT "status" FROM "orders" WHERE "region" = $1 GROUP BY "status" HAVING "count" > $2 AND SUM(total) < $3 ORDER BY "status";`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
	if len(args) != 3 || args[0] != "eu" || args[1] != 5 || args[2] != 999 {
		t.Errorf("args = %v, want [eu 5 999]", args)
	}
}

func TestHavingIn_And_NotIn(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sql, args := NewSelectBuilder([]string{"status"}, pg, nil, nil).
		From("orders").
		GroupBy("status").
		HavingIn("status", []string{"open", "paid"}).
		ToSQL()
	want := `SELECT "status" FROM "orders" GROUP BY "status" HAVING "status" IN ($1, $2);`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
	if len(args) != 2 || args[0] != "open" || args[1] != "paid" {
		t.Errorf("args = %v, want [open paid]", args)
	}

	sql, args = NewSelectBuilder([]string{"status"}, pg, nil, nil).
		From("orders").
		GroupBy("status").
		HavingNotIn("status", "void", "failed").
		ToSQL()
	want = `SELECT "status" FROM "orders" GROUP BY "status" HAVING "status" NOT IN ($1, $2);`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
	if len(args) != 2 || args[0] != "void" || args[1] != "failed" {
		t.Errorf("args = %v, want [void failed]", args)
	}
}

func TestHavingNull_And_NotNull(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sql, args := NewSelectBuilder([]string{"status"}, pg, nil, nil).
		From("orders").
		GroupBy("status").
		HavingNull("closed_at").
		HavingNotNull("status").
		ToSQL()
	want := `SELECT "status" FROM "orders" GROUP BY "status" HAVING "closed_at" IS NULL AND "status" IS NOT NULL;`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
	if len(args) != 0 {
		t.Errorf("args = %v, want none", args)
	}
}

func TestHavingBetween_NumberingAfterWhere(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sql, args := NewSelectBuilder([]string{"status"}, pg, nil, nil).
		From("orders").
		Where("region", "eu").
		GroupBy("status").
		HavingBetween("total", 10, 100).
		HavingNotBetween("count", 0, 2).
		ToSQL()
	want := `SELECT "status" FROM "orders" WHERE "region" = $1 GROUP BY "status" HAVING "total" BETWEEN $2 AND $3 AND "count" NOT BETWEEN $4 AND $5;`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
	if len(args) != 5 || args[0] != "eu" || args[1] != 10 || args[2] != 100 || args[3] != 0 || args[4] != 2 {
		t.Errorf("args = %v, want [eu 10 100 0 2]", args)
	}
}

func TestHavingExists_And_NotExists(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	sql, args := NewSelectBuilder([]string{"status"}, pg, nil, nil).
		From("orders").
		GroupBy("status").
		HavingExists(
			Select("1").From("refunds").WhereRaw("refunds.status = orders.status"),
		).
		ToSQL()
	want := `SELECT "status" FROM "orders" GROUP BY "status" HAVING EXISTS (SELECT "1" FROM "refunds" WHERE refunds.status = orders.status);`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
	if len(args) != 0 {
		t.Errorf("args = %v, want none", args)
	}

	sql, args = NewSelectBuilder([]string{"status"}, pg, nil, nil).
		From("orders").
		GroupBy("status").
		HavingNotExists("SELECT 1 FROM audits WHERE audits.kind = ?", "fraud").
		ToSQL()
	want = `SELECT "status" FROM "orders" GROUP BY "status" HAVING NOT EXISTS (SELECT 1 FROM audits WHERE audits.kind = $1);`
	if sql != want {
		t.Errorf("SQL = %q, want %q", sql, want)
	}
	if len(args) != 1 || args[0] != "fraud" {
		t.Errorf("args = %v, want [fraud]", args)
	}
}

func TestHavingExists_Errors(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	_, err := NewSelectBuilder(nil, pg, &recordingExecer{}, nil).
		From("orders").GroupBy("status").HavingExists(42).Exec()
	if err == nil || !strings.Contains(err.Error(), "expects a *SelectBuilder") {
		t.Errorf("expected subquery type error, got %v", err)
	}

	_, err = NewSelectBuilder(nil, pg, &recordingExecer{}, nil).
		From("orders").GroupBy("status").
		HavingNotExists("SELECT 1 WHERE a = ? AND b = ?", 1).Exec()
	if err == nil || !strings.Contains(err.Error(), "placeholders") {
		t.Errorf("expected placeholder count error, got %v", err)
	}
}

func TestHaving_Errors(t *testing.T) {
	pg := &dialect.PostgresDialect{}

	_, err := NewSelectBuilder(nil, pg, &recordingExecer{}, nil).
		From("orders").GroupBy("status").Having("only-one").Exec()
	if err == nil || !strings.Contains(err.Error(), "Where expects") {
		t.Errorf("expected arity error, got %v", err)
	}

	_, err = NewSelectBuilder(nil, pg, &recordingExecer{}, nil).
		From("orders").HavingRaw("COUNT(*) > ? AND SUM(x) > ?", 5).Exec()
	if err == nil || !strings.Contains(err.Error(), "placeholders") {
		t.Errorf("expected placeholder count error, got %v", err)
	}
}
