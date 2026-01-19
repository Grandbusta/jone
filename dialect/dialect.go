// Package dialect provides database-specific SQL generation.
package dialect

import "github.com/Grandbusta/jone/types"

// Dialect defines the interface for database-specific SQL generation.
type Dialect interface {
	// Name returns the dialect name (e.g., "postgresql", "mysql").
	Name() string

	// CreateTableSQL generates a CREATE TABLE statement.
	CreateTableSQL(table *types.Table) string

	// DropTableSQL generates a DROP TABLE statement.
	DropTableSQL(name string) string

	// DropTableIfExistsSQL generates a DROP TABLE IF EXISTS statement.
	DropTableIfExistsSQL(name string) string

	// CreateTableIfNotExistsSQL generates a CREATE TABLE IF NOT EXISTS statement.
	CreateTableIfNotExistsSQL(table *types.Table) string

	// ColumnDefinitionSQL generates the column definition for use in CREATE TABLE.
	ColumnDefinitionSQL(col *types.Column) string

	// QuoteIdentifier quotes an identifier (table/column name) for this dialect.
	QuoteIdentifier(name string) string

	// AlterTableSQL generates ALTER TABLE statements for all actions.
	AlterTableSQL(tableName string, actions []*types.TableAction) []string

	// DropColumnSQL generates an ALTER TABLE DROP COLUMN statement.
	DropColumnSQL(tableName, columnName string) string

	// AddColumnSQL generates an ALTER TABLE ADD COLUMN statement.
	AddColumnSQL(tableName string, column *types.Column) string

	// SetColumnNotNullSQL generates an ALTER TABLE ALTER COLUMN SET NOT NULL statement.
	SetColumnNotNullSQL(tableName, columnName string) string

	// DropColumnNotNullSQL generates an ALTER TABLE ALTER COLUMN DROP NOT NULL statement.
	DropColumnNotNullSQL(tableName, columnName string) string

	// SetColumnDefaultSQL generates an ALTER TABLE ALTER COLUMN SET DEFAULT statement.
	SetColumnDefaultSQL(tableName, columnName string, defaultValue any) string

	// DropColumnDefaultSQL generates an ALTER TABLE ALTER COLUMN DROP DEFAULT statement.
	DropColumnDefaultSQL(tableName, columnName string) string

	// RenameColumnSQL generates an ALTER TABLE RENAME COLUMN statement.
	RenameColumnSQL(tableName, oldName, newName string) string

	// CreateIndexSQL generates a CREATE INDEX statement.
	CreateIndexSQL(tableName string, index *types.Index) string

	// DropIndexSQL generates a DROP INDEX statement.
	// MySQL requires tableName; PostgreSQL ignores it.
	DropIndexSQL(tableName, indexName string) string

	// AddForeignKeySQL generates an ALTER TABLE ADD CONSTRAINT FOREIGN KEY statement.
	AddForeignKeySQL(tableName string, fk *types.ForeignKey) string

	// DropForeignKeySQL generates an ALTER TABLE DROP CONSTRAINT/FOREIGN KEY statement.
	DropForeignKeySQL(tableName, fkName string) string

	// HasTableSQL returns SQL to check if a table exists (returns count).
	HasTableSQL(tableName string) string

	// HasColumnSQL returns SQL to check if a column exists (returns count).
	HasColumnSQL(tableName, columnName string) string

	// CommentColumnSQL returns SQL to add a comment to a column.
	CommentColumnSQL(tableName, columnName, comment string) string
}

// GetDialect returns a dialect implementation by name.
func GetDialect(name string) Dialect {
	switch name {
	case "postgresql", "postgres", "pg":
		return &PostgresDialect{}
	// case "mysql":
	// 	return &MySQLDialect{}
	default:
		// Default to PostgreSQL
		return &PostgresDialect{}
	}
}
