package schema

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/Grandbusta/jone/config"
	"github.com/Grandbusta/jone/dialect"
)

// Execer is an interface for executing SQL (both *sql.DB and *sql.Tx).
type Execer interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

// Schema provides methods for database schema operations.
type Schema struct {
	dialect dialect.Dialect
	db      *sql.DB // original connection (for Begin, Close)
	execer  Execer  // current executor (db or tx)
	config  *config.Config
	schema  string // current schema context
}

// fatal logs the error and exits. Used for unrecoverable schema errors during migrations.
func fatal(format string, args ...any) {
	err := fmt.Errorf("ERROR: "+format, args...)
	fmt.Println(err)
	// log.Printf("ERROR: "+format, args...)
	os.Exit(1)
}

// New creates a new Schema with the given config.
// It determines the dialect from the config and can optionally connect to the database.
func New(cfg *config.Config) *Schema {
	d := dialect.GetDialect(cfg.Client)
	return &Schema{
		dialect: d,
		config:  cfg,
	}
}

// WithSchema returns a new Schema that operates on the specified schema.
func (s *Schema) WithSchema(schemaName string) *Schema {
	return &Schema{
		dialect: s.dialect,
		db:      s.db,
		execer:  s.execer,
		config:  s.config,
		schema:  schemaName,
	}
}

// WithTx returns a new Schema that uses the given transaction.
func (s *Schema) WithTx(tx *sql.Tx) *Schema {
	return &Schema{
		dialect: s.dialect,
		db:      s.db,
		execer:  tx,
		config:  s.config,
		schema:  s.schema,
	}
}

// BeginTx starts a new transaction and returns it.
func (s *Schema) BeginTx() (*sql.Tx, error) {
	if s.db == nil {
		return nil, fmt.Errorf("no database connection")
	}
	return s.db.Begin()
}

// SchemaName returns the current schema name (empty = default).
func (s *Schema) SchemaName() string {
	return s.schema
}

// Dialect returns the current dialect.
func (s *Schema) Dialect() dialect.Dialect {
	return s.dialect
}

// DB returns the database connection, if set.
func (s *Schema) DB() *sql.DB {
	return s.db
}

// SetDB sets the database connection.
func (s *Schema) SetDB(db *sql.DB) {
	s.db = db
	s.execer = db
}

// Open opens a database connection using the config.
// It uses the dialect to determine the driver and DSN format, and applies
// any connection pool settings from the config.
func (s *Schema) Open() error {
	driver := s.dialect.DriverName()
	dsn := s.dialect.FormatDSN(s.config.Connection)
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w. Check your connection settings in jonefile.go", err)
	}
	if err := db.Ping(); err != nil {
		return fmt.Errorf("cannot connect to database: %w. Verify connection settings in jonefile.go", err)
	}

	// Apply connection pool settings
	pool := s.config.Pool
	if pool.MaxOpenConns > 0 {
		db.SetMaxOpenConns(pool.MaxOpenConns)
	}
	if pool.MaxIdleConns > 0 {
		db.SetMaxIdleConns(pool.MaxIdleConns)
	}
	if pool.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(pool.ConnMaxLifetime)
	}
	if pool.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(pool.ConnMaxIdleTime)
	}

	s.db = db
	s.execer = db
	return nil
}

// Close closes the database connection.
func (s *Schema) Close() error {
	if s.execer != nil {
		return s.db.Close()
	}
	return nil
}

// Raw executes a raw SQL statement with optional parameters.
// Use this for custom DDL, data migrations, or database-specific features.
func (s *Schema) Raw(sqlStmt string, args ...any) {
	if s.execer != nil {
		_, err := s.execer.Exec(sqlStmt, args...)
		if err != nil {
			fatal("executing raw SQL: %v", err)
		}
	} else {
		fmt.Println(sqlStmt)
	}
}

func (s *Schema) Table(name string, builder func(t *Table)) {
	t := NewTable(name)
	t.Schema = s.schema // Set schema context
	builder(t)

	// Generate SQL for each action
	statements := s.dialect.AlterTableSQL(s.schema, name, t.Actions)

	for _, sqlStmt := range statements {
		if s.execer != nil {
			_, err := s.execer.Exec(sqlStmt)
			if err != nil {
				fatal("executing ALTER TABLE: %v", err)
			}
		} else {
			fmt.Println(sqlStmt)
		}
	}
}

// CreateTable creates a new table with the given name using the builder function.
func (s *Schema) CreateTable(name string, builder func(t *Table)) {
	t := NewTable(name)
	t.Schema = s.schema // Set schema context
	builder(t)

	sqlStmt := s.dialect.CreateTableSQL(t.Table)

	// Execute if we have a database connection
	if s.execer != nil {
		_, err := s.execer.Exec(sqlStmt)
		if err != nil {
			fatal("executing CREATE TABLE: %v", err)
		}

		// Execute COMMENT ON COLUMN for columns with comments (PostgreSQL needs separate statement)
		qualifiedTable := s.dialect.QualifyTable(s.schema, name)
		for _, col := range t.Columns {
			if col.Comment != "" {
				commentSQL := s.dialect.CommentColumnSQL(qualifiedTable, col.Name, col.Comment)
				if _, err := s.execer.Exec(commentSQL); err != nil {
					fatal("executing COMMENT ON COLUMN: %v", err)
				}
			}
		}
	} else {
		fmt.Println(sqlStmt)
	}
}

// CreateTableIfNotExists creates a new table if it doesn't already exist.
func (s *Schema) CreateTableIfNotExists(name string, builder func(t *Table)) {
	t := NewTable(name)
	t.Schema = s.schema // Set schema context
	builder(t)

	sqlStmt := s.dialect.CreateTableIfNotExistsSQL(t.Table)
	if s.execer != nil {
		_, err := s.execer.Exec(sqlStmt)
		if err != nil {
			fatal("executing CREATE TABLE IF NOT EXISTS: %v", err)
		}
	} else {
		fmt.Println(sqlStmt)
	}
}

// DropTable drops a table by name.
func (s *Schema) DropTable(name string) {
	sqlStmt := s.dialect.DropTableSQL(s.schema, name)

	if s.execer != nil {
		_, err := s.execer.Exec(sqlStmt)
		if err != nil {
			fatal("executing DROP TABLE: %v", err)
		}
	} else {
		fmt.Println(sqlStmt)
	}
}

// DropTableIfExists drops a table if it exists.
func (s *Schema) DropTableIfExists(name string) {
	sqlStmt := s.dialect.DropTableIfExistsSQL(s.schema, name)

	if s.execer != nil {
		_, err := s.execer.Exec(sqlStmt)
		if err != nil {
			fatal("executing DROP TABLE IF EXISTS: %v", err)
		}
	} else {
		fmt.Println(sqlStmt)
	}
}

// RenameTable renames a table from oldName to newName.
func (s *Schema) RenameTable(oldName, newName string) {
	sqlStmt := fmt.Sprintf("ALTER TABLE %s RENAME TO %s;",
		s.dialect.QualifyTable(s.schema, oldName),
		s.dialect.QuoteIdentifier(newName))

	if s.execer != nil {
		_, err := s.execer.Exec(sqlStmt)
		if err != nil {
			fatal("executing RENAME TABLE: %v", err)
		}
	} else {
		fmt.Println(sqlStmt)
	}
}

// HasTable checks if a table exists.
func (s *Schema) HasTable(name string) bool {
	if s.db == nil {
		return false
	}
	sql := s.dialect.HasTableSQL(s.schema, name)
	var count int
	if err := s.execer.QueryRow(sql).Scan(&count); err != nil {
		return false
	}
	return count > 0
}

// HasColumn checks if a column exists in a table.
func (s *Schema) HasColumn(table, column string) bool {
	if s.db == nil {
		return false
	}
	sql := s.dialect.HasColumnSQL(s.schema, table, column)
	var count int
	if err := s.execer.QueryRow(sql).Scan(&count); err != nil {
		return false
	}
	return count > 0
}
