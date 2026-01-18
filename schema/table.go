package schema

import (
	"github.com/Grandbusta/jone/types"
)

// Table wraps types.Table and provides builder methods.
type Table struct {
	*types.Table
}

// NewTable creates a new Table with the given name.
func NewTable(name string) *Table {
	return &Table{
		Table: &types.Table{Name: name},
	}
}

// addColumn is a helper that creates a column with the given name and type.
// It appends to Columns (for CreateTable) and records an ActionAddColumn (for Table/ALTER).
func (t *Table) addColumn(name, dataType string) *Column {
	col := &types.Column{Name: name, DataType: dataType}
	t.Columns = append(t.Columns, col)
	t.Actions = append(t.Actions, &types.TableAction{
		Type:   types.ActionAddColumn,
		Column: col,
	})
	return &Column{Column: col}
}

// DropColumn drops a column.
func (t *Table) DropColumn(name string) *Table {
	t.Actions = append(t.Actions, &types.TableAction{
		Type: types.ActionDropColumn,
		Name: name,
	})
	return t
}

// DropColumns drops multiple columns.
func (t *Table) DropColumns(names ...string) *Table {
	for _, name := range names {
		t.DropColumn(name)
	}
	return t
}

// SetNullable sets a column as nullable.
func (t *Table) SetNullable(column string) *Table {
	t.Actions = append(t.Actions, &types.TableAction{
		Type: types.ActionDropColumnNotNull,
		Name: column,
	})
	return t
}

// DropNullable sets a column as not nullable.
func (t *Table) DropNullable(column string) *Table {
	t.Actions = append(t.Actions, &types.TableAction{
		Type: types.ActionSetColumnNotNull,
		Name: column,
	})
	return t
}

// SetDefault sets the default value for a column.
func (t *Table) SetDefault(column string, defaultValue any) *Table {
	t.Actions = append(t.Actions, &types.TableAction{
		Type:         types.ActionSetColumnDefault,
		Name:         column,
		DefaultValue: defaultValue,
	})
	return t
}

// DropDefault drops the default value for a column.
func (t *Table) DropDefault(column string) *Table {
	t.Actions = append(t.Actions, &types.TableAction{
		Type: types.ActionDropColumnDefault,
		Name: column,
	})
	return t
}

// RenameColumn renames a column.
func (t *Table) RenameColumn(oldName, newName string) *Table {
	t.Actions = append(t.Actions, &types.TableAction{
		Type:    types.ActionRenameColumn,
		Name:    oldName,
		NewName: newName,
	})
	return t
}

// String creates a VARCHAR column.
func (t *Table) String(name string) *Column {
	// TODO: Support length
	return t.addColumn(name, "varchar")
}

// Text creates a TEXT column (for longer strings).
func (t *Table) Text(name string) *Column {
	return t.addColumn(name, "text")
}

// Int creates an INT column.
func (t *Table) Int(name string) *Column {
	return t.addColumn(name, "int")
}

// BigInt creates a BIGINT column.
func (t *Table) BigInt(name string) *Column {
	return t.addColumn(name, "bigint")
}

// SmallInt creates a SMALLINT column.
func (t *Table) SmallInt(name string) *Column {
	return t.addColumn(name, "smallint")
}

// Boolean creates a BOOLEAN column.
func (t *Table) Boolean(name string) *Column {
	return t.addColumn(name, "boolean")
}

// Float creates a FLOAT column.
func (t *Table) Float(name string) *Column {
	return t.addColumn(name, "float")
}

// Double creates a DOUBLE PRECISION column.
func (t *Table) Double(name string) *Column {
	return t.addColumn(name, "double")
}

// Decimal creates a DECIMAL column.
func (t *Table) Decimal(name string) *Column {
	return t.addColumn(name, "decimal")
}

// Date creates a DATE column.
func (t *Table) Date(name string) *Column {
	return t.addColumn(name, "date")
}

// Time creates a TIME column.
func (t *Table) Time(name string) *Column {
	return t.addColumn(name, "time")
}

// Timestamp creates a TIMESTAMP column.
func (t *Table) Timestamp(name string) *Column {
	return t.addColumn(name, "timestamp")
}

// Timestamps creates created_at and updated_at timestamp columns.
func (t *Table) Timestamps() {
	t.Timestamp("created_at").NotNullable()
	t.Timestamp("updated_at").NotNullable()
}

// UUID creates a UUID column.
func (t *Table) UUID(name string) *Column {
	return t.addColumn(name, "uuid")
}

// JSON creates a JSON column.
func (t *Table) JSON(name string) *Column {
	return t.addColumn(name, "json")
}

// JSONB creates a JSONB column (PostgreSQL-specific, falls back to JSON).
func (t *Table) JSONB(name string) *Column {
	return t.addColumn(name, "jsonb")
}

// Binary creates a BINARY/BLOB column.
func (t *Table) Binary(name string) *Column {
	return t.addColumn(name, "binary")
}

// Increments creates an auto-incrementing primary key column.
func (t *Table) Increments(name string) *Column {
	return t.addColumn(name, "serial").Primary().NotNullable()
}
