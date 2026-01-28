package dialect

import (
	"strings"
	"testing"

	"github.com/Grandbusta/jone/config"
	"github.com/Grandbusta/jone/types"
)

func TestPostgresDialect_Name(t *testing.T) {
	d := &PostgresDialect{}
	if got := d.Name(); got != "postgresql" {
		t.Errorf("Name() = %q, want %q", got, "postgresql")
	}
}

func TestPostgresDialect_QuoteIdentifier(t *testing.T) {
	d := &PostgresDialect{}
	tests := []struct {
		input string
		want  string
	}{
		{"users", `"users"`},
		{"user_id", `"user_id"`},
		{"CamelCase", `"CamelCase"`},
	}
	for _, tt := range tests {
		if got := d.QuoteIdentifier(tt.input); got != tt.want {
			t.Errorf("QuoteIdentifier(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestPostgresDialect_CreateTableSQL(t *testing.T) {
	d := &PostgresDialect{}
	table := &types.Table{
		Name: "users",
		Columns: []*types.Column{
			{Name: "id", DataType: "SERIAL", IsPrimaryKey: true},
			{Name: "email", DataType: "VARCHAR", Length: 255, IsNotNull: true, IsUnique: true},
			{Name: "name", DataType: "VARCHAR", Length: 100},
		},
	}

	sql := d.CreateTableSQL(table)

	// Check key parts are present
	if !strings.Contains(sql, `CREATE TABLE "users"`) {
		t.Error("CREATE TABLE missing table name")
	}
	if !strings.Contains(sql, `"id" SERIAL PRIMARY KEY`) {
		t.Errorf("id column definition incorrect, got: %s", sql)
	}
	if !strings.Contains(sql, `"email"`) && !strings.Contains(sql, "VARCHAR") {
		t.Errorf("email column definition incorrect, got: %s", sql)
	}
}

func TestPostgresDialect_CreateTableWithSchema(t *testing.T) {
	d := &PostgresDialect{}
	table := &types.Table{
		Name:   "users",
		Schema: "app",
		Columns: []*types.Column{
			{Name: "id", DataType: "SERIAL", IsPrimaryKey: true},
		},
	}

	sql := d.CreateTableSQL(table)

	if !strings.Contains(sql, `CREATE TABLE "app"."users"`) {
		t.Errorf("schema not included, got: %s", sql)
	}
}

func TestPostgresDialect_DropTableSQL(t *testing.T) {
	d := &PostgresDialect{}

	tests := []struct {
		schema string
		name   string
		want   string
	}{
		{"", "users", `DROP TABLE "users";`},
		{"app", "users", `DROP TABLE "app"."users";`},
	}

	for _, tt := range tests {
		got := d.DropTableSQL(tt.schema, tt.name)
		if got != tt.want {
			t.Errorf("DropTableSQL(%q, %q) = %q, want %q", tt.schema, tt.name, got, tt.want)
		}
	}
}

func TestPostgresDialect_DropTableIfExistsSQL(t *testing.T) {
	d := &PostgresDialect{}

	got := d.DropTableIfExistsSQL("", "users")
	want := `DROP TABLE IF EXISTS "users";`

	if got != want {
		t.Errorf("DropTableIfExistsSQL() = %q, want %q", got, want)
	}
}

func TestPostgresDialect_ColumnDefinitionSQL_PrimaryKey(t *testing.T) {
	d := &PostgresDialect{}
	col := &types.Column{Name: "id", DataType: "SERIAL", IsPrimaryKey: true}

	got := d.ColumnDefinitionSQL(col)

	if !strings.Contains(got, `"id"`) {
		t.Error("column name not quoted")
	}
	if !strings.Contains(got, "PRIMARY KEY") {
		t.Error("PRIMARY KEY not present")
	}
}

func TestPostgresDialect_ColumnDefinitionSQL_NotNull(t *testing.T) {
	d := &PostgresDialect{}
	col := &types.Column{Name: "email", DataType: "VARCHAR", IsNotNull: true}

	got := d.ColumnDefinitionSQL(col)

	if !strings.Contains(got, "NOT NULL") {
		t.Errorf("NOT NULL not present, got: %s", got)
	}
}

func TestPostgresDialect_ColumnDefinitionSQL_Unique(t *testing.T) {
	d := &PostgresDialect{}
	col := &types.Column{Name: "code", DataType: "VARCHAR", IsUnique: true}

	got := d.ColumnDefinitionSQL(col)

	if !strings.Contains(got, "UNIQUE") {
		t.Errorf("UNIQUE not present, got: %s", got)
	}
}

func TestPostgresDialect_ColumnDefinitionSQL_DefaultString(t *testing.T) {
	d := &PostgresDialect{}
	col := &types.Column{Name: "status", DataType: "VARCHAR", HasDefault: true, DefaultValue: "active"}

	got := d.ColumnDefinitionSQL(col)

	if !strings.Contains(got, "DEFAULT") {
		t.Errorf("DEFAULT not present, got: %s", got)
	}
	if !strings.Contains(got, "'active'") {
		t.Errorf("default value not correctly quoted, got: %s", got)
	}
}

func TestPostgresDialect_ColumnDefinitionSQL_DefaultBool(t *testing.T) {
	d := &PostgresDialect{}
	col := &types.Column{Name: "active", DataType: "BOOLEAN", HasDefault: true, DefaultValue: true}

	got := d.ColumnDefinitionSQL(col)

	if !strings.Contains(got, "DEFAULT") {
		t.Errorf("DEFAULT not present, got: %s", got)
	}
}

func TestPostgresDialect_ColumnDefinitionSQL_UUID(t *testing.T) {
	d := &PostgresDialect{}
	col := &types.Column{Name: "id", DataType: "UUID"}

	got := d.ColumnDefinitionSQL(col)

	if !strings.Contains(got, "UUID") {
		t.Errorf("UUID type not present, got: %s", got)
	}
}

func TestPostgresDialect_ColumnDefinitionSQL_Timestamp(t *testing.T) {
	d := &PostgresDialect{}
	col := &types.Column{Name: "created_at", DataType: "TIMESTAMP", IsNotNull: true}

	got := d.ColumnDefinitionSQL(col)

	if !strings.Contains(got, "TIMESTAMP") {
		t.Errorf("TIMESTAMP type not present, got: %s", got)
	}
	if !strings.Contains(got, "NOT NULL") {
		t.Errorf("NOT NULL not present, got: %s", got)
	}
}

func TestPostgresDialect_CreateMigrationsTableSQL(t *testing.T) {
	d := &PostgresDialect{}

	sql := d.CreateMigrationsTableSQL("jone_migrations")

	if !strings.Contains(sql, "jone_migrations") {
		t.Error("table name not in SQL")
	}
	if !strings.Contains(sql, "CREATE TABLE IF NOT EXISTS") {
		t.Error("CREATE TABLE IF NOT EXISTS not in SQL")
	}
	if !strings.Contains(sql, "migration") {
		t.Error("migration column not in SQL")
	}
	if !strings.Contains(sql, "batch") {
		t.Error("batch column not in SQL")
	}
}

func TestPostgresDialect_InsertMigrationSQL(t *testing.T) {
	d := &PostgresDialect{}

	sql := d.InsertMigrationSQL("jone_migrations")

	if !strings.Contains(sql, "INSERT INTO") {
		t.Error("INSERT INTO not in SQL")
	}
	if !strings.Contains(sql, "jone_migrations") {
		t.Error("table name not in SQL")
	}
}

func TestPostgresDialect_DeleteMigrationSQL(t *testing.T) {
	d := &PostgresDialect{}

	sql := d.DeleteMigrationSQL("jone_migrations")

	if !strings.Contains(sql, "DELETE FROM") {
		t.Error("DELETE FROM not in SQL")
	}
	if !strings.Contains(sql, "jone_migrations") {
		t.Error("table name not in SQL")
	}
}

func TestPostgresDialect_QualifyTable(t *testing.T) {
	d := &PostgresDialect{}

	tests := []struct {
		schema string
		name   string
		want   string
	}{
		{"", "users", `"users"`},
		{"public", "users", `"public"."users"`},
		{"app", "posts", `"app"."posts"`},
	}

	for _, tt := range tests {
		got := d.QualifyTable(tt.schema, tt.name)
		if got != tt.want {
			t.Errorf("QualifyTable(%q, %q) = %q, want %q", tt.schema, tt.name, got, tt.want)
		}
	}
}

func TestPostgresDialect_CommentColumnSQL(t *testing.T) {
	d := &PostgresDialect{}

	sql := d.CommentColumnSQL("users", "email", "User's email address")

	if !strings.Contains(sql, "COMMENT ON COLUMN") {
		t.Error("COMMENT ON COLUMN not in SQL")
	}
	if !strings.Contains(sql, "users") {
		t.Error("table name not in SQL")
	}
	if !strings.Contains(sql, "email") {
		t.Error("column name not in SQL")
	}
}

func TestPostgresDialect_DriverName(t *testing.T) {
	d := &PostgresDialect{}
	if got := d.DriverName(); got != "pgx" {
		t.Errorf("DriverName() = %q, want %q", got, "pgx")
	}
}

func TestPostgresDialect_FormatDSN(t *testing.T) {
	d := &PostgresDialect{}

	tests := []struct {
		name string
		conn config.Connection
		want string
	}{
		{
			name: "basic connection",
			conn: config.Connection{
				Host:     "localhost",
				Port:     "5432",
				User:     "postgres",
				Password: "secret",
				Database: "testdb",
			},
			want: "host=localhost port=5432 user=postgres password=secret dbname=testdb sslmode=disable",
		},
		{
			name: "with ssl mode",
			conn: config.Connection{
				Host:     "db.example.com",
				Port:     "5432",
				User:     "admin",
				Password: "pass",
				Database: "prod",
				SSLMode:  "require",
			},
			want: "host=db.example.com port=5432 user=admin password=pass dbname=prod sslmode=require",
		},
		{
			name: "empty ssl defaults to disable",
			conn: config.Connection{
				Host:     "localhost",
				Port:     "5432",
				User:     "user",
				Password: "pw",
				Database: "db",
				SSLMode:  "",
			},
			want: "host=localhost port=5432 user=user password=pw dbname=db sslmode=disable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.FormatDSN(tt.conn)
			if got != tt.want {
				t.Errorf("FormatDSN() = %q, want %q", got, tt.want)
			}
		})
	}
}
