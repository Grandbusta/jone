// Package jone provides a database migration and query building library.
//
// This package re-exports types from sub-packages for convenient access.
// For more control, import the specific sub-packages directly:
//
//	import "github.com/Grandbusta/jone/config"
//	import "github.com/Grandbusta/jone/schema"
//	import "github.com/Grandbusta/jone/migration"
//	import "github.com/Grandbusta/jone/dialect"
//	import "github.com/Grandbusta/jone/query"
package jone

import (
	"github.com/Grandbusta/jone/config"
	"github.com/Grandbusta/jone/dialect"
	"github.com/Grandbusta/jone/migration"
	"github.com/Grandbusta/jone/query"
	"github.com/Grandbusta/jone/schema"
	"github.com/Grandbusta/jone/types"
)

// Configuration types (re-exported from config package)
type Config = config.Config
type Connection = config.Connection
type Pool = config.Pool
type Migrations = config.Migrations

// Schema types (re-exported from schema package)
type Schema = schema.Schema
type TxSchema = schema.TxSchema
type Table = schema.Table
type Column = schema.Column

// WhereGroup collects conditions for a parenthesized WHERE sub-group
// (re-exported from query package):
//
//	db.Select("*").From("users").Where(func(g *jone.WhereGroup) {
//	    g.Where("role", "admin").OrWhere("age", ">", 65)
//	})
type WhereGroup = query.WhereGroup

// SelectBuilder builds SELECT queries (re-exported from query package).
type SelectBuilder = query.SelectBuilder

// Select starts a standalone SELECT builder, mainly for use as an EXISTS
// subquery or, named via As(), as a derived table:
//
//	db.Select("*").From("users").
//	    WhereExists(jone.Select("1").From("orders").WhereRaw("orders.user_id = users.id"))
//
//	sub := jone.Select("user_id").From("orders").GroupBy("user_id").As("t")
//	db.Select("*").From(sub).Exec()
//
// To run queries, use db.Select() on a connected instance instead.
var Select = query.Select

// Core types (re-exported from types package)
type CoreTable = types.Table
type CoreColumn = types.Column

// Fn provides SQL function helpers (e.g. jone.Fn.Now()).
var Fn = types.Fn

// Raw wraps a string as a raw SQL expression, bypassing parameterization.
// Use for SQL functions, expressions, or conflict targets:
//
//	jone.Raw("NOW()")
//	jone.Raw("(email) WHERE active")
func Raw(expr string) types.RawExpr {
	return types.RawExpr{Expr: expr}
}

// New creates a new database instance with the given config.
var New = schema.New

// Migration types (re-exported from migration package)
type Registration = migration.Registration
type RunParams = migration.RunParams
type RunOptions = migration.RunOptions

// RunLatest executes pending Up migrations in order.
var RunLatest = migration.RunLatest

// RunList displays all migrations with their status.
var RunList = migration.RunList

// RunUp runs the next pending migration or a specific one.
var RunUp = migration.RunUp

// RunDown rolls back the last single migration.
var RunDown = migration.RunDown

// RunRollback rolls back the last batch of migrations.
var RunRollback = migration.RunRollback

// Dialect types and functions (re-exported from dialect package)
type Dialect = dialect.Dialect

// GetDialect returns a dialect implementation by name.
var GetDialect = dialect.GetDialect
