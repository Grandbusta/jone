// Package query provides query building for DML operations.
// This is a stub for future implementation.
package query

// Builder is the main query builder interface.
type Builder interface {
	// ToSQL generates the SQL string and arguments.
	ToSQL() (string, []any)
}

// SelectBuilder builds SELECT queries.
type SelectBuilder struct {
	table   string
	columns []string
	where   []string
	orderBy []string
	limit   *int
	offset  *int
}

// Select starts building a SELECT query.
func Select(columns ...string) *SelectBuilder {
	return &SelectBuilder{columns: columns}
}

// From sets the table to select from.
func (s *SelectBuilder) From(table string) *SelectBuilder {
	s.table = table
	return s
}

// Where adds a WHERE condition.
func (s *SelectBuilder) Where(condition string) *SelectBuilder {
	s.where = append(s.where, condition)
	return s
}

// OrderBy adds an ORDER BY clause.
func (s *SelectBuilder) OrderBy(column string) *SelectBuilder {
	s.orderBy = append(s.orderBy, column)
	return s
}

// Limit sets the LIMIT clause.
func (s *SelectBuilder) Limit(n int) *SelectBuilder {
	s.limit = &n
	return s
}

// Offset sets the OFFSET clause.
func (s *SelectBuilder) Offset(n int) *SelectBuilder {
	s.offset = &n
	return s
}

// ToSQL generates the SELECT SQL. (stub implementation)
func (s *SelectBuilder) ToSQL() (string, []any) {
	// TODO: Implement full SQL generation
	return "", nil
}


// UpdateBuilder builds UPDATE queries.
type UpdateBuilder struct {
	table string
	set   map[string]any
	where []string
}

// Update starts building an UPDATE query.
func Update(table string) *UpdateBuilder {
	return &UpdateBuilder{table: table, set: make(map[string]any)}
}

// Set adds a column=value pair to update.
func (u *UpdateBuilder) Set(column string, value any) *UpdateBuilder {
	u.set[column] = value
	return u
}

// Where adds a WHERE condition.
func (u *UpdateBuilder) Where(condition string) *UpdateBuilder {
	u.where = append(u.where, condition)
	return u
}

// ToSQL generates the UPDATE SQL. (stub implementation)
func (u *UpdateBuilder) ToSQL() (string, []any) {
	// TODO: Implement full SQL generation
	return "", nil
}

// DeleteBuilder builds DELETE queries.
type DeleteBuilder struct {
	table string
	where []string
}

// Delete starts building a DELETE query.
func Delete(table string) *DeleteBuilder {
	return &DeleteBuilder{table: table}
}

// Where adds a WHERE condition.
func (d *DeleteBuilder) Where(condition string) *DeleteBuilder {
	d.where = append(d.where, condition)
	return d
}

// ToSQL generates the DELETE SQL. (stub implementation)
func (d *DeleteBuilder) ToSQL() (string, []any) {
	// TODO: Implement full SQL generation
	return "", nil
}
