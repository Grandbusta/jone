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

	// RenameColumnSQL generates an ALTER TABLE RENAME COLUMN statement.
	RenameColumnSQL(tableName, oldName, newName string) string
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
