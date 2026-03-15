// Package dialect provides database-specific SQL generation.
package dialect

import (
	"github.com/Grandbusta/jone/config"
	"github.com/Grandbusta/jone/types"
)

// InsertOptions holds options for INSERT query generation.
type InsertOptions struct {
	OnConflictIgnore bool
	// ConflictColumns are the column names for the ON CONFLICT target.
	// e.g. ["email"] → ON CONFLICT ("email") DO NOTHING
	ConflictColumns []string
	// ConflictRaw is a raw conflict target expression used as-is.
	// e.g. "(email) WHERE active" → ON CONFLICT (email) WHERE active DO NOTHING
	ConflictRaw string
}

// Dialect defines the interface for database-specific SQL generation.
type Dialect interface {
	// Name returns the dialect name (e.g., "postgresql", "mysql").
	Name() string

	// DriverName returns the database/sql driver name for this dialect
	// (e.g., "pgx", "mysql", "sqlite3").
	DriverName() string

	// FormatDSN builds a connection string from the given connection parameters.
	FormatDSN(conn config.Connection) string

	// CreateTableSQL generates a CREATE TABLE statement.
	CreateTableSQL(table *types.Table) string

	// CreateTableIfNotExistsSQL generates a CREATE TABLE IF NOT EXISTS statement.
	CreateTableIfNotExistsSQL(table *types.Table) string

	// DropTableSQL generates a DROP TABLE statement.
	DropTableSQL(schema, name string) string

	// DropTableIfExistsSQL generates a DROP TABLE IF EXISTS statement.
	DropTableIfExistsSQL(schema, name string) string

	// AlterTableSQL generates ALTER TABLE statements for all actions.
	AlterTableSQL(schema, tableName string, actions []*types.TableAction) []string

	// HasTableSQL returns SQL to check if a table exists (returns count).
	HasTableSQL(schema, tableName string) string

	// HasColumnSQL returns SQL to check if a column exists (returns count).
	HasColumnSQL(schema, tableName, columnName string) string

	// ColumnDefinitionSQL generates the column definition for use in CREATE TABLE.
	ColumnDefinitionSQL(col *types.Column) string

	// CommentColumnSQL returns SQL to add a comment to a column.
	CommentColumnSQL(tableName, columnName, comment string) string

	// QuoteIdentifier quotes an identifier (table/column name) for this dialect.
	QuoteIdentifier(name string) string

	// QualifyTable returns a schema-qualified table name.
	// If schema is empty, returns just the quoted table name.
	QualifyTable(schema, tableName string) string

	// --- Query Builder Methods ---

	// InsertSQL generates a parameterized INSERT statement for a single row.
	InsertSQL(table string, data map[string]any, opts InsertOptions) (string, []any)

	// InsertManySQL generates a parameterized INSERT statement for multiple rows.
	InsertManySQL(table string, data []map[string]any, opts InsertOptions) (string, []any)

	// SelectSQL generates a SELECT statement. WHERE clauses are raw strings (no args).
	SelectSQL(table string, columns []string, wheres []string, orderBys []string, limit *int, offset *int) string

	// UpdateSQL generates a parameterized UPDATE statement.
	UpdateSQL(table string, set map[string]any, wheres []string) (string, []any)

	// DeleteSQL generates a DELETE statement. WHERE clauses are raw strings (no args).
	DeleteSQL(table string, wheres []string) string

	// --- Migration Tracking Methods ---

	// CreateMigrationsTableSQL returns SQL to create the migrations tracking table.
	CreateMigrationsTableSQL(tableName string) string

	// InsertMigrationSQL returns parameterized SQL to record a migration.
	// Parameters: $1=name, $2=batch
	InsertMigrationSQL(tableName string) string

	// DeleteMigrationSQL returns parameterized SQL to remove a migration record.
	// Parameters: $1=name
	DeleteMigrationSQL(tableName string) string

	// GetAppliedMigrationsSQL returns SQL to get all applied migration names.
	GetAppliedMigrationsSQL(tableName string) string

	// GetLastBatchSQL returns SQL to get the highest batch number.
	GetLastBatchSQL(tableName string) string

	// GetMigrationsByBatchSQL returns parameterized SQL to get migrations for a batch.
	// Parameters: $1=batch
	GetMigrationsByBatchSQL(tableName string) string
}

// GetDialect returns a dialect implementation by name.
func GetDialect(name string) Dialect {
	switch name {
	case "postgresql", "postgres", "pg":
		return &PostgresDialect{}
	case "mysql":
		return &MySQLDialect{}
	default:
		// Default to PostgreSQL
		return &PostgresDialect{}
	}
}
