// Package types provides core types used across the jone library.
// This package has no internal dependencies to prevent import cycles.
package types

// RawExpr represents a raw SQL expression that should not be parameterized or quoted.
type RawExpr struct {
	Expr string
}

// Functions provides SQL function helpers.
type Functions struct{}

// Now returns a raw CURRENT_TIMESTAMP expression.
func (f *Functions) Now() RawExpr {
	return RawExpr{Expr: "CURRENT_TIMESTAMP"}
}

// Fn is the global functions instance for use as jone.Fn.Now(), etc.
var Fn = &Functions{}

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
	ActionCreateIndex       ActionType = "create_index"
	ActionDropIndex         ActionType = "drop_index"
	ActionAddForeignKey     ActionType = "add_foreign_key"
	ActionDropForeignKey    ActionType = "drop_foreign_key"
	ActionDropPrimary       ActionType = "drop_primary"
)

// Index represents a database index definition.
type Index struct {
	Name      string   // Index name (auto-generated if empty)
	Columns   []string // Columns to index
	IsUnique  bool     // UNIQUE constraint
	Method    string   // btree, hash, gin, gist (PostgreSQL)
	TableName string   // For auto-generating name
}

// ForeignKey represents a database foreign key constraint.
type ForeignKey struct {
	Name      string // FK constraint name (auto-generated if empty)
	Column    string // Local column
	RefTable  string // Referenced table
	RefColumn string // Referenced column
	OnDelete  string // CASCADE, SET NULL, RESTRICT, NO ACTION
	OnUpdate  string // CASCADE, SET NULL, RESTRICT, NO ACTION
	TableName string // For auto-generating name
}

// TableAction represents a single alteration operation on a table.
type TableAction struct {
	Type         ActionType
	Column       *Column // For add/modify operations
	Name         string  // Column name for drop, old name for rename
	NewName      string  // New name for rename operations
	DefaultValue any
	Index        *Index      // For index operations
	ForeignKey   *ForeignKey // For foreign key operations
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
	RefOnDelete  string // CASCADE, SET NULL, RESTRICT, NO ACTION
	RefOnUpdate  string // CASCADE, SET NULL, RESTRICT, NO ACTION
	Comment      string // Column comment/description
}

// Table represents a database table definition.
type Table struct {
	Name    string
	Schema  string // Database schema (e.g., "public", "app")
	Columns []*Column
	Actions []*TableAction
}
