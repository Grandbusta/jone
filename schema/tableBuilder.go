package schema

type Schema struct {
}

type Table struct {
	Name string
}

func (s *Schema) CreateTable(tableName string, callback func(table *Table)) {

}

func (s *Schema) DropTable(tableName string) {

}

func (s *Schema) Table(tableName string, callback func(table *Table)) {

}

// Renames a column in the table
func (t *Table) RenameColumn(oldName string, newName string) {

}

func (t *Table) DropColumn(name string) {

}

func (t *Table) Int(newName string) {

}

func (t *Table) String(newName string) {

}

func (t *Table) Bool(newName string) {
}

// func (j *Jone) Up() {
// 	j.Schema.CreateTable("users", func(table *Table) {
// 		table.Int("id")
// 		table.String("name")
// 		table.Bool("is_admin")
// 		table.DropColumn("food")
// 	})
// }

// func (j *Jone) Down() {
// 	j.Schema.DropTable("users")
// 	j.Schema.Table("users", func(table *Table) {
// 		table.RenameColumn("id", "user_id")
// 		table.RenameColumn("name", "username")
// 		table.RenameColumn("age", "user_age")
// 	})
// }
