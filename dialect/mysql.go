package dialect

import (
	"fmt"
	"strings"

	"github.com/Grandbusta/jone/types"
)

// MySQLDialect implements Dialect for MySQL.
type MySQLDialect struct{}

// Name returns "mysql".
func (d *MySQLDialect) Name() string {
	return "mysql"
}

// QuoteIdentifier quotes an identifier with backticks for MySQL.
func (d *MySQLDialect) QuoteIdentifier(name string) string {
	return fmt.Sprintf("`%s`", name)
}

// CreateTableSQL generates a CREATE TABLE statement for MySQL.
func (d *MySQLDialect) CreateTableSQL(table *types.Table) string {
	var columns []string
	for _, col := range table.Columns {
		columns = append(columns, d.ColumnDefinitionSQL(col))
	}

	return fmt.Sprintf(
		"CREATE TABLE %s (\n  %s\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
		d.QuoteIdentifier(table.Name),
		strings.Join(columns, ",\n  "),
	)
}

// DropTableSQL generates a DROP TABLE statement.
func (d *MySQLDialect) DropTableSQL(name string) string {
	return fmt.Sprintf("DROP TABLE %s;", d.QuoteIdentifier(name))
}

// DropTableIfExistsSQL generates a DROP TABLE IF EXISTS statement.
func (d *MySQLDialect) DropTableIfExistsSQL(name string) string {
	return fmt.Sprintf("DROP TABLE IF EXISTS %s;", d.QuoteIdentifier(name))
}

// ColumnDefinitionSQL generates the column definition SQL.
func (d *MySQLDialect) ColumnDefinitionSQL(col *types.Column) string {
	var parts []string

	parts = append(parts, d.QuoteIdentifier(col.Name))
	parts = append(parts, d.mapDataType(col))

	if col.IsUnsigned {
		parts = append(parts, "UNSIGNED")
	}
	if col.IsNotNull {
		parts = append(parts, "NOT NULL")
	}
	if col.HasDefault {
		parts = append(parts, fmt.Sprintf("DEFAULT %v", d.formatDefault(col.DefaultValue)))
	}
	if col.IsPrimaryKey {
		parts = append(parts, "PRIMARY KEY")
	}
	if col.IsUnique && !col.IsPrimaryKey {
		parts = append(parts, "UNIQUE")
	}

	return strings.Join(parts, " ")
}

// mapDataType maps generic types to MySQL-specific types.
func (d *MySQLDialect) mapDataType(col *types.Column) string {
	switch col.DataType {
	case "varchar":
		return "VARCHAR(255)"
	case "int":
		return "INT"
	case "bigint":
		return "BIGINT"
	case "smallint":
		return "SMALLINT"
	case "float":
		return "FLOAT"
	case "double":
		return "DOUBLE"
	case "decimal":
		return "DECIMAL(10,2)"
	case "boolean":
		return "TINYINT(1)"
	case "text":
		return "TEXT"
	case "date":
		return "DATE"
	case "time":
		return "TIME"
	case "timestamp":
		return "TIMESTAMP"
	case "uuid":
		return "CHAR(36)" // MySQL doesn't have native UUID
	case "json":
		return "JSON"
	case "jsonb":
		return "JSON" // MySQL uses JSON for both
	case "binary":
		return "BLOB"
	case "serial":
		return "INT AUTO_INCREMENT"
	case "bigserial":
		return "BIGINT AUTO_INCREMENT"
	default:
		return strings.ToUpper(col.DataType)
	}
}

// formatDefault formats a default value for SQL.
func (d *MySQLDialect) formatDefault(value any) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("'%s'", v)
	case bool:
		if v {
			return "1"
		}
		return "0"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// AlterTableSQL generates ALTER TABLE statements for all actions.
func (d *MySQLDialect) AlterTableSQL(tableName string, actions []*types.TableAction) []string {
	var statements []string
	for _, action := range actions {
		switch action.Type {
		case types.ActionDropColumn:
			statements = append(statements, d.DropColumnSQL(tableName, action.Name))
		case types.ActionAddColumn:
			statements = append(statements, d.AddColumnSQL(tableName, action.Column))
		case types.ActionRenameColumn:
			statements = append(statements, d.RenameColumnSQL(tableName, action.Name, action.NewName))
		case types.ActionChangeColumnType:
			statements = append(statements, d.ChangeColumnTypeSQL(tableName, action.Column))
		case types.ActionSetColumnNotNull:
			statements = append(statements, d.SetColumnNotNullSQL(tableName, action.Name))
		case types.ActionDropColumnNotNull:
			statements = append(statements, d.DropColumnNotNullSQL(tableName, action.Name))
		case types.ActionSetColumnDefault:
			statements = append(statements, d.SetColumnDefaultSQL(tableName, action.Name, action.DefaultValue))
		case types.ActionDropColumnDefault:
			statements = append(statements, d.DropColumnDefaultSQL(tableName, action.Name))
		}
	}
	return statements
}

// DropColumnSQL generates an ALTER TABLE DROP COLUMN statement.
func (d *MySQLDialect) DropColumnSQL(tableName, columnName string) string {
	return fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;",
		d.QuoteIdentifier(tableName),
		d.QuoteIdentifier(columnName))
}

// AddColumnSQL generates an ALTER TABLE ADD COLUMN statement.
func (d *MySQLDialect) AddColumnSQL(tableName string, column *types.Column) string {
	return fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s;",
		d.QuoteIdentifier(tableName),
		d.ColumnDefinitionSQL(column))
}

// RenameColumnSQL generates an ALTER TABLE RENAME COLUMN statement.
// Uses RENAME COLUMN syntax (MySQL 8.0+).
func (d *MySQLDialect) RenameColumnSQL(tableName, oldName, newName string) string {
	return fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s;",
		d.QuoteIdentifier(tableName),
		d.QuoteIdentifier(oldName),
		d.QuoteIdentifier(newName))
}

// ChangeColumnTypeSQL generates an ALTER TABLE MODIFY COLUMN statement to change column type.
// Note: MySQL uses MODIFY COLUMN which requires the full column definition.
func (d *MySQLDialect) ChangeColumnTypeSQL(tableName string, column *types.Column) string {
	return fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s %s;",
		d.QuoteIdentifier(tableName),
		d.QuoteIdentifier(column.Name),
		d.mapDataType(column))
}

// SetColumnNotNullSQL generates an ALTER TABLE MODIFY COLUMN statement to set NOT NULL.
// Note: MySQL requires knowing the column type to modify constraints.
// This is a simplified version that may need the full column definition in practice.
func (d *MySQLDialect) SetColumnNotNullSQL(tableName, columnName string) string {
	// MySQL doesn't have a direct "SET NOT NULL" - you need to use MODIFY with the full definition.
	// This is a workaround that works for simple cases.
	return fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s %s NOT NULL;",
		d.QuoteIdentifier(tableName),
		d.QuoteIdentifier(columnName),
		"VARCHAR(255)") // TODO: This needs the actual column type
}

// DropColumnNotNullSQL generates an ALTER TABLE MODIFY COLUMN statement to drop NOT NULL.
// Note: MySQL requires knowing the column type to modify constraints.
func (d *MySQLDialect) DropColumnNotNullSQL(tableName, columnName string) string {
	return fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s %s NULL;",
		d.QuoteIdentifier(tableName),
		d.QuoteIdentifier(columnName),
		"VARCHAR(255)") // TODO: This needs the actual column type
}

// SetColumnDefaultSQL generates an ALTER TABLE ALTER COLUMN SET DEFAULT statement.
func (d *MySQLDialect) SetColumnDefaultSQL(tableName, columnName string, defaultValue any) string {
	return fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s;",
		d.QuoteIdentifier(tableName),
		d.QuoteIdentifier(columnName),
		d.formatDefault(defaultValue))
}

// DropColumnDefaultSQL generates an ALTER TABLE ALTER COLUMN DROP DEFAULT statement.
func (d *MySQLDialect) DropColumnDefaultSQL(tableName, columnName string) string {
	return fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP DEFAULT;",
		d.QuoteIdentifier(tableName),
		d.QuoteIdentifier(columnName))
}
