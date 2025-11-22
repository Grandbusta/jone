package jone

type Config struct {
	Client     string
	Connection Connection
	Migrations Migrations
}

type Connection struct {
	User     string
	Password string
	Database string
	Port     string
	Host     string
}

type Migrations struct {
	TableName string
}

type CreateTableBuilder struct {
	Table *Table
}

type Table struct {
	Name    string
	Columns []*Column
}

type Column struct {
	name       string
	dataType   string
	primaryKey bool
	notNull    bool
	unique     bool
}

func (t *Table) String(columnName string) *Column {
	col := &Column{name: columnName, dataType: "varchar"}
	t.Columns = append(t.Columns, col)
	return col
}

func (t *Table) Int(columnName string) *Column {
	col := &Column{name: columnName, dataType: "int"}
	t.Columns = append(t.Columns, col)
	return col
}

func (c *Column) PrimaryKey() *Column {
	c.primaryKey = true
	return c
}

func (c *Column) NotNull() *Column {
	c.notNull = true
	return c
}

func (c *Column) Unique() *Column {
	c.unique = true
	return c
}

type Schema struct {
}

func (s *Schema) CreateTable(name string, builder func(t *Table)) error {
	t := Table{Name: name}
	builder(&t)
	return nil
}
