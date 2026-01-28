// Package config provides configuration types for the jone migration tool.
package config

import (
	"time"
)

// Config holds the main configuration for jone.
type Config struct {
	Client     string
	Connection Connection
	Pool       Pool
	Migrations Migrations
}

// Pool holds connection pool configuration.
// Zero values preserve database/sql defaults.
type Pool struct {
	MaxOpenConns    int           // Maximum number of open connections. 0 means unlimited.
	MaxIdleConns    int           // Maximum number of idle connections. 0 means default (2).
	ConnMaxLifetime time.Duration // Maximum time a connection may be reused. 0 means no limit.
	ConnMaxIdleTime time.Duration // Maximum time a connection may be idle. 0 means no limit.
}

// Connection holds database connection parameters.
type Connection struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
	SSLMode  string // disable, require, verify-full
}

// Migrations holds migration-specific configuration.
type Migrations struct {
	TableName string
}
