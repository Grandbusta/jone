package jone

import "github.com/Grandbusta/jone/schema"

type Table struct {
	*schema.Table
}

type Jone struct {
	Schema *schema.Schema
	Table  *Table
}
