package dialect

import (
	"fmt"
	"strings"

	"github.com/Grandbusta/jone/config"
	"github.com/Grandbusta/jone/types"
)

// MySQLDialect implements Dialect for MySQL.
type MySQLDialect struct{}

// Name returns "mysql".
func (d *MySQLDialect) Name() string {
	return "mysql"
}

// DriverName returns "mysql" for the go-sql-driver/mysql driver.
func (d *MySQLDialect) DriverName() string {
	return "mysql"
}

// FormatDSN builds a MySQL connection string in DSN format.
func (d *MySQLDialect) FormatDSN(conn config.Connection) string {
	// Format: user:password@tcp(host:port)/database
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
		conn.User, conn.Password, conn.Host, conn.Port, conn.Database)
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
		d.QualifyTable(table.Schema, table.Name),
		strings.Join(columns, ",\n  "),
	)
}

// DropTableSQL generates a DROP TABLE statement.
func (d *MySQLDialect) DropTableSQL(schema, name string) string {
	return fmt.Sprintf("DROP TABLE %s;", d.QualifyTable(schema, name))
}

// DropTableIfExistsSQL generates a DROP TABLE IF EXISTS statement.
func (d *MySQLDialect) DropTableIfExistsSQL(schema, name string) string {
	return fmt.Sprintf("DROP TABLE IF EXISTS %s;", d.QualifyTable(schema, name))
}

// CreateTableIfNotExistsSQL generates a CREATE TABLE IF NOT EXISTS statement.
func (d *MySQLDialect) CreateTableIfNotExistsSQL(table *types.Table) string {
	var columns []string
	for _, col := range table.Columns {
		columns = append(columns, d.ColumnDefinitionSQL(col))
	}

	return fmt.Sprintf(
		"CREATE TABLE IF NOT EXISTS %s (\n  %s\n);",
		d.QualifyTable(table.Schema, table.Name),
		strings.Join(columns, ",\n  "),
	)
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
	if col.Comment != "" {
		parts = append(parts, fmt.Sprintf("COMMENT '%s'", col.Comment))
	}
	if col.RefTable != "" && col.RefColumn != "" {
		refPart := fmt.Sprintf(
			"REFERENCES %s(%s)",
			d.QuoteIdentifier(col.RefTable),
			d.QuoteIdentifier(col.RefColumn),
		)
		if col.RefOnDelete != "" {
			refPart += " ON DELETE " + col.RefOnDelete
		}
		if col.RefOnUpdate != "" {
			refPart += " ON UPDATE " + col.RefOnUpdate
		}
		parts = append(parts, refPart)
	}

	return strings.Join(parts, " ")
}

// mapDataType maps generic types to MySQL-specific types.
func (d *MySQLDialect) mapDataType(col *types.Column) string {
	switch col.DataType {
	case "varchar":
		if col.Length > 0 {
			return fmt.Sprintf("VARCHAR(%d)", col.Length)
		}
		return "VARCHAR(255)"
	case "char":
		if col.Length > 0 {
			return fmt.Sprintf("CHAR(%d)", col.Length)
		}
		return "CHAR(1)"
	case "int":
		return "INT"
	case "bigint":
		return "BIGINT"
	case "smallint":
		return "SMALLINT"
	case "float":
		if col.Precision > 0 {
			return fmt.Sprintf("FLOAT(%d)", col.Precision)
		}
		return "FLOAT"
	case "double":
		return "DOUBLE"
	case "decimal":
		p := col.Precision
		if p == 0 {
			p = 10
		}
		s := col.Scale
		if s == 0 {
			s = 2
		}
		return fmt.Sprintf("DECIMAL(%d,%d)", p, s)
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
		if col.Length > 0 {
			return fmt.Sprintf("VARBINARY(%d)", col.Length)
		}
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
func (d *MySQLDialect) AlterTableSQL(schema, tableName string, actions []*types.TableAction) []string {
	qualifiedTable := d.QualifyTable(schema, tableName)
	var statements []string
	for _, action := range actions {
		switch action.Type {
		case types.ActionDropColumn:
			statements = append(statements, d.dropColumnSQL(qualifiedTable, action.Name))
		case types.ActionAddColumn:
			statements = append(statements, d.addColumnSQL(qualifiedTable, action.Column))
		case types.ActionRenameColumn:
			statements = append(statements, d.renameColumnSQL(qualifiedTable, action.Name, action.NewName))
		case types.ActionChangeColumnType:
			statements = append(statements, d.changeColumnTypeSQL(qualifiedTable, action.Column))
		case types.ActionSetColumnNotNull:
			statements = append(statements, d.setColumnNotNullSQL(qualifiedTable, action.Name))
		case types.ActionDropColumnNotNull:
			statements = append(statements, d.dropColumnNotNullSQL(qualifiedTable, action.Name))
		case types.ActionSetColumnDefault:
			statements = append(statements, d.setColumnDefaultSQL(qualifiedTable, action.Name, action.DefaultValue))
		case types.ActionDropColumnDefault:
			statements = append(statements, d.dropColumnDefaultSQL(qualifiedTable, action.Name))
		case types.ActionCreateIndex:
			statements = append(statements, d.createIndexSQL(qualifiedTable, action.Index))
		case types.ActionDropIndex:
			statements = append(statements, d.dropIndexSQL(qualifiedTable, action.Index.Name))
		case types.ActionAddForeignKey:
			statements = append(statements, d.addForeignKeySQL(qualifiedTable, action.ForeignKey))
		case types.ActionDropForeignKey:
			statements = append(statements, d.dropForeignKeySQL(qualifiedTable, action.ForeignKey.Name))
		case types.ActionDropPrimary:
			statements = append(statements, d.dropPrimarySQL(qualifiedTable))
		}
	}
	return statements
}

// createIndexSQL generates a CREATE INDEX statement.
func (d *MySQLDialect) createIndexSQL(tableName string, idx *types.Index) string {
	unique := ""
	if idx.IsUnique {
		unique = "UNIQUE "
	}

	cols := make([]string, len(idx.Columns))
	for i, c := range idx.Columns {
		cols[i] = d.QuoteIdentifier(c)
	}

	return fmt.Sprintf("CREATE %sINDEX %s ON %s (%s);",
		unique,
		d.QuoteIdentifier(idx.Name),
		tableName,
		strings.Join(cols, ", "))
}

// dropIndexSQL generates a DROP INDEX statement for MySQL.
// MySQL requires the table name for DROP INDEX.
// tableName should be pre-qualified (e.g., from QualifyTable).
func (d *MySQLDialect) dropIndexSQL(tableName, indexName string) string {
	return fmt.Sprintf("DROP INDEX %s ON %s;",
		d.QuoteIdentifier(indexName),
		tableName)
}

// addForeignKeySQL generates an ALTER TABLE ADD CONSTRAINT FOREIGN KEY statement.
// tableName should be pre-qualified (e.g., from QualifyTable).
func (d *MySQLDialect) addForeignKeySQL(tableName string, fk *types.ForeignKey) string {
	sql := fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s(%s)",
		tableName,
		d.QuoteIdentifier(fk.Name),
		d.QuoteIdentifier(fk.Column),
		d.QuoteIdentifier(fk.RefTable),
		d.QuoteIdentifier(fk.RefColumn))

	if fk.OnDelete != "" {
		sql += " ON DELETE " + fk.OnDelete
	}
	if fk.OnUpdate != "" {
		sql += " ON UPDATE " + fk.OnUpdate
	}
	return sql + ";"
}

// dropForeignKeySQL generates an ALTER TABLE DROP FOREIGN KEY statement.
// MySQL uses DROP FOREIGN KEY instead of DROP CONSTRAINT.
// tableName should be pre-qualified (e.g., from QualifyTable).
func (d *MySQLDialect) dropForeignKeySQL(tableName, fkName string) string {
	return fmt.Sprintf("ALTER TABLE %s DROP FOREIGN KEY %s;",
		tableName,
		d.QuoteIdentifier(fkName))
}

// dropColumnSQL generates an ALTER TABLE DROP COLUMN statement.
// dropColumnSQL generates an ALTER TABLE DROP COLUMN statement.
// tableName should be pre-qualified (e.g., from QualifyTable).
func (d *MySQLDialect) dropColumnSQL(tableName, columnName string) string {
	return fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;",
		tableName,
		d.QuoteIdentifier(columnName))
}

// addColumnSQL generates an ALTER TABLE ADD COLUMN statement.
// tableName should be pre-qualified (e.g., from QualifyTable).
func (d *MySQLDialect) addColumnSQL(tableName string, column *types.Column) string {
	return fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s;",
		tableName,
		d.ColumnDefinitionSQL(column))
}

// renameColumnSQL generates an ALTER TABLE RENAME COLUMN statement.
// Uses RENAME COLUMN syntax (MySQL 8.0+).
// tableName should be pre-qualified (e.g., from QualifyTable).
func (d *MySQLDialect) renameColumnSQL(tableName, oldName, newName string) string {
	return fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s;",
		tableName,
		d.QuoteIdentifier(oldName),
		d.QuoteIdentifier(newName))
}

// changeColumnTypeSQL generates an ALTER TABLE MODIFY COLUMN statement to change column type.
// Note: MySQL uses MODIFY COLUMN which requires the full column definition.
// tableName should be pre-qualified (e.g., from QualifyTable).
func (d *MySQLDialect) changeColumnTypeSQL(tableName string, column *types.Column) string {
	return fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s %s;",
		tableName,
		d.QuoteIdentifier(column.Name),
		d.mapDataType(column))
}

// setColumnNotNullSQL generates an ALTER TABLE MODIFY COLUMN statement to set NOT NULL.
// Note: MySQL requires knowing the column type to modify constraints.
// tableName should be pre-qualified (e.g., from QualifyTable).
func (d *MySQLDialect) setColumnNotNullSQL(tableName, columnName string) string {
	// MySQL doesn't have a direct "SET NOT NULL" - you need to use MODIFY with the full definition.
	// This is a workaround that works for simple cases.
	return fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s %s NOT NULL;",
		tableName,
		d.QuoteIdentifier(columnName),
		"VARCHAR(255)") // TODO: This needs the actual column type
}

// dropColumnNotNullSQL generates an ALTER TABLE MODIFY COLUMN statement to drop NOT NULL.
// Note: MySQL requires knowing the column type to modify constraints.
// tableName should be pre-qualified (e.g., from QualifyTable).
func (d *MySQLDialect) dropColumnNotNullSQL(tableName, columnName string) string {
	return fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s %s NULL;",
		tableName,
		d.QuoteIdentifier(columnName),
		"VARCHAR(255)") // TODO: This needs the actual column type
}

// setColumnDefaultSQL generates an ALTER TABLE ALTER COLUMN SET DEFAULT statement.
// tableName should be pre-qualified (e.g., from QualifyTable).
func (d *MySQLDialect) setColumnDefaultSQL(tableName, columnName string, defaultValue any) string {
	return fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s;",
		tableName,
		d.QuoteIdentifier(columnName),
		d.formatDefault(defaultValue))
}

// dropColumnDefaultSQL generates an ALTER TABLE ALTER COLUMN DROP DEFAULT statement.
// tableName should be pre-qualified (e.g., from QualifyTable).
func (d *MySQLDialect) dropColumnDefaultSQL(tableName, columnName string) string {
	return fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP DEFAULT;",
		tableName,
		d.QuoteIdentifier(columnName))
}

// HasTableSQL returns SQL to check if a table exists in MySQL.
func (d *MySQLDialect) HasTableSQL(schema, tableName string) string {
	if schema != "" {
		return fmt.Sprintf(`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = '%s' AND table_name = '%s'`, schema, tableName)
	}
	return fmt.Sprintf(`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = '%s'`, tableName)
}

// HasColumnSQL returns SQL to check if a column exists in MySQL.
func (d *MySQLDialect) HasColumnSQL(schema, tableName, columnName string) string {
	if schema != "" {
		return fmt.Sprintf(`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = '%s' AND table_name = '%s' AND column_name = '%s'`, schema, tableName, columnName)
	}
	return fmt.Sprintf(`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = '%s' AND column_name = '%s'`, tableName, columnName)
}

// CommentColumnSQL returns SQL to add a comment to a column in MySQL.
// Note: MySQL supports inline COMMENT in CREATE TABLE.
func (d *MySQLDialect) CommentColumnSQL(tableName, columnName, comment string) string {
	return ""
}

// dropPrimarySQL returns SQL to drop the primary key constraint in MySQL.
// Note: MySQL doesn't use constraint names for primary keys.
// tableName should be pre-qualified (e.g., from QualifyTable).
func (d *MySQLDialect) dropPrimarySQL(tableName string) string {
	return fmt.Sprintf("ALTER TABLE %s DROP PRIMARY KEY;", tableName)
}

// QualifyTable returns a schema-qualified table name.
// MySQL uses database.table syntax.
func (d *MySQLDialect) QualifyTable(schema, tableName string) string {
	if schema == "" {
		return d.QuoteIdentifier(tableName)
	}
	return fmt.Sprintf("%s.%s", d.QuoteIdentifier(schema), d.QuoteIdentifier(tableName))
}

// --- Migration Tracking Methods ---

// CreateMigrationsTableSQL returns SQL to create the migrations tracking table.
func (d *MySQLDialect) CreateMigrationsTableSQL(tableName string) string {
	return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
	id INT AUTO_INCREMENT PRIMARY KEY,
	name VARCHAR(255) NOT NULL UNIQUE,
	batch INT NOT NULL,
	applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`, d.QuoteIdentifier(tableName))
}

// InsertMigrationSQL returns parameterized SQL to record a migration.
func (d *MySQLDialect) InsertMigrationSQL(tableName string) string {
	return fmt.Sprintf("INSERT INTO %s (name, batch) VALUES (?, ?);",
		d.QuoteIdentifier(tableName))
}

// DeleteMigrationSQL returns parameterized SQL to remove a migration record.
func (d *MySQLDialect) DeleteMigrationSQL(tableName string) string {
	return fmt.Sprintf("DELETE FROM %s WHERE name = ?;",
		d.QuoteIdentifier(tableName))
}

// GetAppliedMigrationsSQL returns SQL to get all applied migration names ordered by id.
func (d *MySQLDialect) GetAppliedMigrationsSQL(tableName string) string {
	return fmt.Sprintf("SELECT name FROM %s ORDER BY id;",
		d.QuoteIdentifier(tableName))
}

// GetLastBatchSQL returns SQL to get the highest batch number.
func (d *MySQLDialect) GetLastBatchSQL(tableName string) string {
	return fmt.Sprintf("SELECT COALESCE(MAX(batch), 0) FROM %s;",
		d.QuoteIdentifier(tableName))
}

// GetMigrationsByBatchSQL returns parameterized SQL to get migrations for a batch.
func (d *MySQLDialect) GetMigrationsByBatchSQL(tableName string) string {
	return fmt.Sprintf("SELECT name FROM %s WHERE batch = ? ORDER BY id DESC;",
		d.QuoteIdentifier(tableName))
}
