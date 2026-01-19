// Package types provides core types used across the jone library.
// This package has no internal dependencies to prevent import cycles.
package types

// ActionType represents the type of table alteration action.
type ActionType string

const (
	ActionDropColumn        ActionType = "drop_column"
	ActionAddColumn         ActionType = "add_column"
	ActionRenameColumn      ActionType = "rename_column"
	ActionChangeColumnType  ActionType = "change_column_type"
	ActionSetColumnNotNull  ActionType = "set_column_not_null"
	ActionDropColumnNotNull ActionType = "drop_column_not_null"
	ActionSetColumnDefault  ActionType = "set_column_default"
	ActionDropColumnDefault ActionType = "drop_column_default"
)

// TableAction represents a single alteration operation on a table.
type TableAction struct {
	Type         ActionType
	Column       *Column // For add/modify operations
	Name         string  // Column name for drop, old name for rename
	NewName      string  // New name for rename operations
	DefaultValue any
}

// Column represents a database column definition.
type Column struct {
	Name         string
	DataType     string
	Length       int // For VARCHAR, CHAR, BINARY
	Precision    int // For DECIMAL, NUMERIC, FLOAT
	Scale        int // For DECIMAL, NUMERIC
	IsPrimaryKey bool
	IsNotNull    bool
	IsUnique     bool
	IsUnsigned   bool
	DefaultValue any
	HasDefault   bool
	RefTable     string
	RefColumn    string
}

// Table represents a database table definition.
type Table struct {
	Name    string
	Columns []*Column
	Actions []*TableAction
}
