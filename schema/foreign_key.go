package schema

import (
	"github.com/Grandbusta/jone/types"
)

// ForeignKeyBuilder provides a fluent interface for creating foreign keys.
type ForeignKeyBuilder struct {
	table    *Table
	column   string
	name     string
	refTable string
	refCol   string
	onDelete string
	onUpdate string
}

// References sets the referenced table and column.
func (b *ForeignKeyBuilder) References(table, column string) *ForeignKeyBuilder {
	b.refTable = table
	b.refCol = column
	b.updateAction()
	return b
}

// OnDelete sets the ON DELETE action (CASCADE, SET NULL, RESTRICT, NO ACTION).
func (b *ForeignKeyBuilder) OnDelete(action string) *ForeignKeyBuilder {
	b.onDelete = action
	b.updateAction()
	return b
}

// OnUpdate sets the ON UPDATE action (CASCADE, SET NULL, RESTRICT, NO ACTION).
func (b *ForeignKeyBuilder) OnUpdate(action string) *ForeignKeyBuilder {
	b.onUpdate = action
	b.updateAction()
	return b
}

// Name sets a custom name for the foreign key constraint.
func (b *ForeignKeyBuilder) Name(n string) *ForeignKeyBuilder {
	b.name = n
	b.updateAction()
	return b
}

// build creates the ForeignKey struct with auto-generated name if needed.
func (b *ForeignKeyBuilder) build() *types.ForeignKey {
	name := b.name
	if name == "" {
		name = b.generateName()
	}
	return &types.ForeignKey{
		Name:      name,
		Column:    b.column,
		RefTable:  b.refTable,
		RefColumn: b.refCol,
		OnDelete:  b.onDelete,
		OnUpdate:  b.onUpdate,
		TableName: b.table.Name,
	}
}

// generateName creates an auto-generated foreign key name.
func (b *ForeignKeyBuilder) generateName() string {
	return "fk_" + b.table.Name + "_" + b.column
}

// updateAction updates the last action with the current builder state.
func (b *ForeignKeyBuilder) updateAction() {
	if len(b.table.Actions) > 0 {
		lastAction := b.table.Actions[len(b.table.Actions)-1]
		if lastAction.Type == types.ActionAddForeignKey {
			lastAction.ForeignKey = b.build()
		}
	}
}
