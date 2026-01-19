package dialect

import (
	"fmt"
	"strings"

	"github.com/Grandbusta/jone/types"
)

// PostgresDialect implements Dialect for PostgreSQL.
type PostgresDialect struct{}

// Name returns "postgresql".
func (d *PostgresDialect) Name() string {
	return "postgresql"
}

// QuoteIdentifier quotes an identifier with double quotes for PostgreSQL.
func (d *PostgresDialect) QuoteIdentifier(name string) string {
	return fmt.Sprintf(`"%s"`, name)
}

// CreateTableSQL generates a CREATE TABLE statement for PostgreSQL.
func (d *PostgresDialect) CreateTableSQL(table *types.Table) string {
	var columns []string
	for _, col := range table.Columns {
		columns = append(columns, d.ColumnDefinitionSQL(col))
	}

	return fmt.Sprintf(
		"CREATE TABLE %s (\n  %s\n);",
		d.QuoteIdentifier(table.Name),
		strings.Join(columns, ",\n  "),
	)
}

// DropTableSQL generates a DROP TABLE statement.
func (d *PostgresDialect) DropTableSQL(name string) string {
	return fmt.Sprintf("DROP TABLE %s;", d.QuoteIdentifier(name))
}

// DropTableIfExistsSQL generates a DROP TABLE IF EXISTS statement.
func (d *PostgresDialect) DropTableIfExistsSQL(name string) string {
	return fmt.Sprintf("DROP TABLE IF EXISTS %s;", d.QuoteIdentifier(name))
}

// CreateTableIfNotExistsSQL generates a CREATE TABLE IF NOT EXISTS statement.
func (d *PostgresDialect) CreateTableIfNotExistsSQL(table *types.Table) string {
	var columns []string
	for _, col := range table.Columns {
		columns = append(columns, d.ColumnDefinitionSQL(col))
	}

	return fmt.Sprintf(
		"CREATE TABLE IF NOT EXISTS %s (\n  %s\n);",
		d.QuoteIdentifier(table.Name),
		strings.Join(columns, ",\n  "),
	)
}

// ColumnDefinitionSQL generates the column definition SQL.
func (d *PostgresDialect) ColumnDefinitionSQL(col *types.Column) string {
	var parts []string

	parts = append(parts, d.QuoteIdentifier(col.Name))
	parts = append(parts, d.mapDataType(col))

	if col.IsPrimaryKey {
		parts = append(parts, "PRIMARY KEY")
	}
	if col.IsNotNull && !col.IsPrimaryKey {
		parts = append(parts, "NOT NULL")
	}
	if col.IsUnique && !col.IsPrimaryKey {
		parts = append(parts, "UNIQUE")
	}
	if col.HasDefault {
		parts = append(parts, fmt.Sprintf("DEFAULT %v", d.formatDefault(col.DefaultValue)))
	}
	if col.RefTable != "" && col.RefColumn != "" {
		parts = append(parts, fmt.Sprintf(
			"REFERENCES %s(%s)",
			d.QuoteIdentifier(col.RefTable),
			d.QuoteIdentifier(col.RefColumn),
		))
	}

	return strings.Join(parts, " ")
}

// mapDataType maps generic types to PostgreSQL-specific types.
func (d *PostgresDialect) mapDataType(col *types.Column) string {
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
		return "INTEGER"
	case "bigint":
		return "BIGINT"
	case "smallint":
		return "SMALLINT"
	case "float":
		if col.Precision > 0 {
			return fmt.Sprintf("FLOAT(%d)", col.Precision)
		}
		return "REAL"
	case "double":
		return "DOUBLE PRECISION"
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
		return "BOOLEAN"
	case "text":
		return "TEXT"
	case "date":
		return "DATE"
	case "time":
		return "TIME"
	case "timestamp":
		return "TIMESTAMP"
	case "uuid":
		return "UUID"
	case "json":
		return "JSON"
	case "jsonb":
		return "JSONB"
	case "binary":
		return "BYTEA"
	case "serial":
		return "SERIAL"
	case "bigserial":
		return "BIGSERIAL"
	default:
		return strings.ToUpper(col.DataType)
	}
}

// formatDefault formats a default value for SQL.
func (d *PostgresDialect) formatDefault(value any) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("'%s'", v)
	case bool:
		if v {
			return "TRUE"
		}
		return "FALSE"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// AlterTableSQL generates ALTER TABLE statements for all actions.
func (d *PostgresDialect) AlterTableSQL(tableName string, actions []*types.TableAction) []string {
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
		case types.ActionCreateIndex:
			statements = append(statements, d.CreateIndexSQL(tableName, action.Index))
		case types.ActionDropIndex:
			statements = append(statements, d.DropIndexSQL(tableName, action.Index.Name))
		case types.ActionAddForeignKey:
			statements = append(statements, d.AddForeignKeySQL(tableName, action.ForeignKey))
		case types.ActionDropForeignKey:
			statements = append(statements, d.DropForeignKeySQL(tableName, action.ForeignKey.Name))
		}
	}
	return statements
}

// CreateIndexSQL generates a CREATE INDEX statement.
func (d *PostgresDialect) CreateIndexSQL(tableName string, idx *types.Index) string {
	unique := ""
	if idx.IsUnique {
		unique = "UNIQUE "
	}

	using := ""
	if idx.Method != "" {
		using = fmt.Sprintf(" USING %s", idx.Method)
	}

	cols := make([]string, len(idx.Columns))
	for i, c := range idx.Columns {
		cols[i] = d.QuoteIdentifier(c)
	}

	return fmt.Sprintf("CREATE %sINDEX %s ON %s%s (%s);",
		unique,
		d.QuoteIdentifier(idx.Name),
		d.QuoteIdentifier(tableName),
		using,
		strings.Join(cols, ", "))
}

// DropIndexSQL generates a DROP INDEX statement.
// PostgreSQL doesn't require tableName but accepts it for interface compatibility.
func (d *PostgresDialect) DropIndexSQL(tableName, name string) string {
	return fmt.Sprintf("DROP INDEX %s;", d.QuoteIdentifier(name))
}

// AddForeignKeySQL generates an ALTER TABLE ADD CONSTRAINT FOREIGN KEY statement.
func (d *PostgresDialect) AddForeignKeySQL(tableName string, fk *types.ForeignKey) string {
	sql := fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s(%s)",
		d.QuoteIdentifier(tableName),
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

// DropForeignKeySQL generates an ALTER TABLE DROP CONSTRAINT statement.
func (d *PostgresDialect) DropForeignKeySQL(tableName, fkName string) string {
	return fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s;",
		d.QuoteIdentifier(tableName),
		d.QuoteIdentifier(fkName))
}

// DropColumnSQL generates an ALTER TABLE DROP COLUMN statement.
func (d *PostgresDialect) DropColumnSQL(tableName, columnName string) string {
	return fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;",
		d.QuoteIdentifier(tableName),
		d.QuoteIdentifier(columnName))
}

// AddColumnSQL generates an ALTER TABLE ADD COLUMN statement.
func (d *PostgresDialect) AddColumnSQL(tableName string, column *types.Column) string {
	return fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s;",
		d.QuoteIdentifier(tableName),
		d.ColumnDefinitionSQL(column))
}

// RenameColumnSQL generates an ALTER TABLE RENAME COLUMN statement.
func (d *PostgresDialect) RenameColumnSQL(tableName, oldName, newName string) string {
	return fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s;",
		d.QuoteIdentifier(tableName),
		d.QuoteIdentifier(oldName),
		d.QuoteIdentifier(newName))
}

// ChangeColumnTypeSQL generates an ALTER TABLE ALTER COLUMN TYPE statement.
func (d *PostgresDialect) ChangeColumnTypeSQL(tableName string, column *types.Column) string {
	return fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s;",
		d.QuoteIdentifier(tableName),
		d.QuoteIdentifier(column.Name),
		d.mapDataType(column))
}

// SetColumnNotNullSQL generates an ALTER TABLE ALTER COLUMN SET NOT NULL statement.
func (d *PostgresDialect) SetColumnNotNullSQL(tableName, columnName string) string {
	return fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL;",
		d.QuoteIdentifier(tableName),
		d.QuoteIdentifier(columnName))
}

// DropColumnNotNullSQL generates an ALTER TABLE ALTER COLUMN DROP NOT NULL statement.
func (d *PostgresDialect) DropColumnNotNullSQL(tableName, columnName string) string {
	return fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL;",
		d.QuoteIdentifier(tableName),
		d.QuoteIdentifier(columnName))
}

// SetColumnDefaultSQL generates an ALTER TABLE ALTER COLUMN SET DEFAULT statement.
func (d *PostgresDialect) SetColumnDefaultSQL(tableName, columnName string, defaultValue any) string {
	return fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s;",
		d.QuoteIdentifier(tableName),
		d.QuoteIdentifier(columnName),
		d.formatDefault(defaultValue))
}

// DropColumnDefaultSQL generates an ALTER TABLE ALTER COLUMN DROP DEFAULT statement.
func (d *PostgresDialect) DropColumnDefaultSQL(tableName, columnName string) string {
	return fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP DEFAULT;",
		d.QuoteIdentifier(tableName),
		d.QuoteIdentifier(columnName))
}

// HasTableSQL returns SQL to check if a table exists in PostgreSQL.
func (d *PostgresDialect) HasTableSQL(tableName string) string {
	return fmt.Sprintf(`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name = '%s'`, tableName)
}

// HasColumnSQL returns SQL to check if a column exists in PostgreSQL.
func (d *PostgresDialect) HasColumnSQL(tableName, columnName string) string {
	return fmt.Sprintf(`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = 'public' AND table_name = '%s' AND column_name = '%s'`, tableName, columnName)
}

// CommentColumnSQL returns SQL to add a comment to a column in PostgreSQL.
func (d *PostgresDialect) CommentColumnSQL(tableName, columnName, comment string) string {
	return fmt.Sprintf("COMMENT ON COLUMN %s.%s IS '%s';",
		d.QuoteIdentifier(tableName),
		d.QuoteIdentifier(columnName),
		comment)
}
