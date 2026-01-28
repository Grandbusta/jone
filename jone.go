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
type Table = schema.Table
type Column = schema.Column

// Core types (re-exported from types package)
type CoreTable = types.Table
type CoreColumn = types.Column

// NewSchema creates a new Schema with the given config.
var NewSchema = schema.New

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
