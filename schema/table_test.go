package schema

import (
	"testing"
)

func TestTable_String(t *testing.T) {
	table := NewTable("users")
	col := table.String("name")

	if col.Name != "name" {
		t.Errorf("column name = %q, want %q", col.Name, "name")
	}
	if col.DataType != "varchar" {
		t.Errorf("column type = %q, want %q", col.DataType, "varchar")
	}
}

func TestTable_StringWithModifiers(t *testing.T) {
	table := NewTable("users")
	col := table.String("email")
	col.Length(255)
	col.NotNullable()
	col.Unique()

	if col.Column.Length != 255 {
		t.Errorf("length = %d, want %d", col.Column.Length, 255)
	}
	if !col.IsNotNull {
		t.Error("expected IsNotNull to be true")
	}
	if !col.IsUnique {
		t.Error("expected IsUnique to be true")
	}
}

func TestTable_Int(t *testing.T) {
	table := NewTable("users")
	col := table.Int("age")

	if col.DataType != "int" {
		t.Errorf("column type = %q, want %q", col.DataType, "int")
	}
}

func TestTable_BigInt(t *testing.T) {
	table := NewTable("users")
	col := table.BigInt("views")

	if col.DataType != "bigint" {
		t.Errorf("column type = %q, want %q", col.DataType, "bigint")
	}
}

func TestTable_Boolean(t *testing.T) {
	table := NewTable("users")
	col := table.Boolean("active")
	col.Default(true)

	if col.DataType != "boolean" {
		t.Errorf("column type = %q, want %q", col.DataType, "boolean")
	}
	if !col.HasDefault {
		t.Error("expected HasDefault to be true")
	}
	if col.DefaultValue != true {
		t.Errorf("default value = %v, want %v", col.DefaultValue, true)
	}
}

func TestTable_UUID(t *testing.T) {
	table := NewTable("users")
	col := table.UUID("id")
	col.Primary()

	if col.DataType != "uuid" {
		t.Errorf("column type = %q, want %q", col.DataType, "uuid")
	}
	if !col.IsPrimaryKey {
		t.Error("expected IsPrimaryKey to be true")
	}
}

func TestTable_Decimal(t *testing.T) {
	table := NewTable("products")
	col := table.Decimal("price")
	col.Precision(10)
	col.Scale(2)
	col.NotNullable()

	if col.DataType != "decimal" {
		t.Errorf("column type = %q, want %q", col.DataType, "decimal")
	}
	if col.Column.Precision != 10 {
		t.Errorf("precision = %d, want %d", col.Column.Precision, 10)
	}
	if col.Column.Scale != 2 {
		t.Errorf("scale = %d, want %d", col.Column.Scale, 2)
	}
}

func TestTable_Timestamps(t *testing.T) {
	table := NewTable("users")
	table.Timestamps()

	// Should create 2 columns: created_at and updated_at
	if len(table.Columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(table.Columns))
	}

	var createdAt, updatedAt bool
	for _, col := range table.Columns {
		if col.Name == "created_at" {
			createdAt = true
			if col.DataType != "timestamp" {
				t.Errorf("created_at type = %q, want %q", col.DataType, "timestamp")
			}
			if !col.IsNotNull {
				t.Error("expected created_at to be NOT NULL")
			}
		}
		if col.Name == "updated_at" {
			updatedAt = true
		}
	}

	if !createdAt {
		t.Error("expected created_at column")
	}
	if !updatedAt {
		t.Error("expected updated_at column")
	}
}

func TestTable_Increments(t *testing.T) {
	table := NewTable("users")
	col := table.Increments("id")

	if col.DataType != "serial" {
		t.Errorf("column type = %q, want %q", col.DataType, "serial")
	}
	if !col.IsPrimaryKey {
		t.Error("expected IsPrimaryKey to be true")
	}
}

func TestTable_DropColumn(t *testing.T) {
	table := NewTable("users")
	table.DropColumn("legacy_field")

	if len(table.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(table.Actions))
	}
	if table.Actions[0].Type != "drop_column" {
		t.Errorf("action type = %q, want %q", table.Actions[0].Type, "drop_column")
	}
	if table.Actions[0].Name != "legacy_field" {
		t.Errorf("action name = %q, want %q", table.Actions[0].Name, "legacy_field")
	}
}

func TestTable_RenameColumn(t *testing.T) {
	table := NewTable("users")
	table.RenameColumn("old_name", "new_name")

	if len(table.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(table.Actions))
	}
	if table.Actions[0].Type != "rename_column" {
		t.Errorf("action type = %q, want %q", table.Actions[0].Type, "rename_column")
	}
	if table.Actions[0].Name != "old_name" {
		t.Errorf("old name = %q, want %q", table.Actions[0].Name, "old_name")
	}
	if table.Actions[0].NewName != "new_name" {
		t.Errorf("new name = %q, want %q", table.Actions[0].NewName, "new_name")
	}
}

func TestTable_Index(t *testing.T) {
	table := NewTable("users")
	table.Index("email")

	if len(table.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(table.Actions))
	}
	if table.Actions[0].Type != "create_index" {
		t.Errorf("action type = %q, want %q", table.Actions[0].Type, "create_index")
	}
	if table.Actions[0].Index == nil {
		t.Fatal("expected Index to be set")
	}
	if len(table.Actions[0].Index.Columns) != 1 {
		t.Errorf("expected 1 column in index, got %d", len(table.Actions[0].Index.Columns))
	}
}

func TestTable_UniqueIndex(t *testing.T) {
	table := NewTable("users")
	table.Unique("email")

	if len(table.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(table.Actions))
	}
	if !table.Actions[0].Index.IsUnique {
		t.Error("expected unique index")
	}
}

func TestTable_Foreign(t *testing.T) {
	table := NewTable("posts")
	table.Foreign("user_id").References("users", "id").OnDelete("CASCADE")

	if len(table.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(table.Actions))
	}
	if table.Actions[0].Type != "add_foreign_key" {
		t.Errorf("action type = %q, want %q", table.Actions[0].Type, "add_foreign_key")
	}

	fk := table.Actions[0].ForeignKey
	if fk == nil {
		t.Fatal("expected ForeignKey to be set")
	}
	if fk.Column != "user_id" {
		t.Errorf("column = %q, want %q", fk.Column, "user_id")
	}
	if fk.RefTable != "users" {
		t.Errorf("ref table = %q, want %q", fk.RefTable, "users")
	}
	if fk.RefColumn != "id" {
		t.Errorf("ref column = %q, want %q", fk.RefColumn, "id")
	}
	if fk.OnDelete != "CASCADE" {
		t.Errorf("on delete = %q, want %q", fk.OnDelete, "CASCADE")
	}
}

func TestColumn_ChainedModifiers(t *testing.T) {
	table := NewTable("products")
	col := table.String("sku")
	col.Length(50)
	col.NotNullable()
	col.Unique()
	col.Default("")
	col.Comment("Stock Keeping Unit")

	if col.Column.Length != 50 {
		t.Errorf("length = %d, want 50", col.Column.Length)
	}
	if !col.IsNotNull {
		t.Error("expected NotNull")
	}
	if !col.IsUnique {
		t.Error("expected Unique")
	}
	if !col.HasDefault {
		t.Error("expected HasDefault")
	}
	if col.Column.Comment != "Stock Keeping Unit" {
		t.Errorf("comment = %q, want %q", col.Column.Comment, "Stock Keeping Unit")
	}
}
