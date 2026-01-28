# Jone

A Go database migration tool with a fluent schema builder. Query builder coming soon!

## Installation

**1. Install the CLI** (one-time, adds `jone` command to your system):

```bash
go install github.com/Grandbusta/jone/cmd/jone@latest
```

**2. Add library to your project** (run in your project directory):

```bash
go get github.com/Grandbusta/jone
```

## Quick Start

```bash
# Initialize jone in your project (defaults to postgres)
jone init

# Create a migration
jone migrate:make create_users

# Run all pending migrations
jone migrate:latest

# View migration status
jone migrate:list
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `jone init` | Initialize jone project. Creates `jone/` folder and config. |
| `jone migrate:make <name>` | Create a new migration file. |
| `jone migrate:latest` | Run all pending migrations. |
| `jone migrate:up [name]` | Run next pending migration (or specific one). |
| `jone migrate:down [name]` | Rollback last migration (or specific one). |
| `jone migrate:rollback` | Rollback last batch of migrations. |
| `jone migrate:list` | List all migrations with status. |
| `jone migrate:status` | Alias for `migrate:list`. |

### Flags

**`jone init`**
- `--db`, `-d` — Database type: `postgres`, `mysql`, `sqlite` (default: `postgres`)

**`jone migrate:latest`**, **`migrate:up`**, **`migrate:down`**, **`migrate:rollback`**
- `--dry-run` — Show SQL that would be executed without running it

**`jone migrate:rollback`**
- `--all`, `-a` — Rollback all migrations (not just last batch)

## Configuration

After running `jone init`, edit `jone/jonefile.go`:

```go
package jone

import (
    "github.com/Grandbusta/jone"
    _ "github.com/jackc/pgx/v5/stdlib" // Driver
)

var Config = jone.Config{
    Client: "postgresql",
    Connection: jone.Connection{
        Host:     "localhost",
        Port:     "5432",
        User:     "postgres",
        Password: "password",
        Database: "my_db",
    },
    Migrations: jone.Migrations{
        TableName: "jone_migrations",
    },
}
```

## Connection Pooling

Jone leverages Go's built-in `database/sql` connection pool. You can configure pool behavior by adding a `Pool` field to your config:

```go
import "time"

var Config = jone.Config{
    Client: "postgresql",
    Connection: jone.Connection{
        Host:     "localhost",
        Port:     "5432",
        User:     "postgres",
        Password: "password",
        Database: "my_db",
    },
    Pool: jone.Pool{
        MaxOpenConns:    10,              // Max open connections (0 = unlimited)
        MaxIdleConns:    5,               // Max idle connections (0 = default 2)
        ConnMaxLifetime: 30 * time.Minute, // Max connection reuse time (0 = no limit)
        ConnMaxIdleTime: 5 * time.Minute,  // Max idle time before close (0 = no limit)
    },
    Migrations: jone.Migrations{
        TableName: "jone_migrations",
    },
}
```

All fields are optional. Omitting `Pool` (or using zero values) preserves the `database/sql` defaults.

## Schema Builder

### Creating Tables

```go
func Up(s *jone.Schema) {
    s.CreateTable("users", func(t *jone.Table) {
        t.Increments("id")
        t.String("name").Length(100).NotNullable()
        t.String("email").Length(255).NotNullable().Unique()
        t.Text("bio").Nullable()
        t.Boolean("active").Default(true)
        t.Timestamps() // created_at, updated_at
    })
}
```

### Column Types

| Method | SQL Type |
|--------|----------|
| `Increments(name)` | SERIAL / AUTO_INCREMENT PRIMARY KEY |
| `String(name)` | VARCHAR(255) |
| `Text(name)` | TEXT |
| `Int(name)` | INTEGER |
| `BigInt(name)` | BIGINT |
| `SmallInt(name)` | SMALLINT |
| `Boolean(name)` | BOOLEAN |
| `Float(name)` | REAL / FLOAT |
| `Double(name)` | DOUBLE PRECISION |
| `Decimal(name)` | DECIMAL |
| `Date(name)` | DATE |
| `Time(name)` | TIME |
| `Timestamp(name)` | TIMESTAMP |
| `UUID(name)` | UUID |
| `JSON(name)` | JSON |
| `JSONB(name)` | JSONB |
| `Binary(name)` | BYTEA / BLOB |
| `Enum(name, ...values)` | ENUM type |

### Column Modifiers

```go
t.String("name").Length(100)         // Set length
t.String("email").NotNullable()      // NOT NULL
t.String("status").Default("new")    // Default value
t.String("code").Unique()            // Unique constraint
t.String("notes").Nullable()         // Explicitly nullable
t.String("title").Comment("...")     // Column comment
t.BigInt("user_id").Unsigned()       // Unsigned (MySQL)
t.BigInt("id").Primary()             // Primary key
t.Decimal("price").Precision(10).Scale(2) // DECIMAL(10,2)
```

### Indexes

```go
// Create index
t.Index("email")
t.Index("first_name", "last_name").Name("idx_full_name")
t.Index("data").Using("gin")  // Index method (btree, hash, gin, gist)

// Unique index
t.Unique("email")
t.Unique("org_id", "slug").Name("uq_org_slug")
```

### Foreign Keys

```go
t.BigInt("user_id").NotNullable()
t.Foreign("user_id").References("users", "id").OnDelete("CASCADE")
t.Foreign("org_id").References("orgs", "id").OnDelete("SET NULL").OnUpdate("CASCADE")
t.Foreign("custom").References("table", "col").Name("fk_custom_name")
```

### Timestamps

```go
t.Timestamps() // Adds created_at and updated_at
```

### Altering Tables

```go
func Up(s *jone.Schema) {
    s.Table("users", func(t *jone.Table) {
        t.String("phone").Nullable()        // Add column
        t.DropColumn("legacy_field")        // Drop column
        t.RenameColumn("name", "full_name") // Rename column
        t.SetNullable("bio")                // Make nullable
        t.DropNullable("email")             // Make not nullable
        t.SetDefault("active", true)        // Set default
        t.DropDefault("active")             // Drop default
        t.DropIndex("email")                // Drop index
        t.DropForeign("user_id")            // Drop foreign key
    })
}
```

### Dropping Tables

```go
func Down(s *jone.Schema) {
    s.DropTable("users")
    // or
    s.DropTableIfExists("users")
}
```

### Other Operations

```go
s.RenameTable("old_name", "new_name")
s.HasTable("users")           // Check if table exists
s.HasColumn("users", "email") // Check if column exists
```

### Raw SQL

For custom statements the schema builder doesn't support:

```go
// DDL statements
s.Raw("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"")
s.Raw("CREATE INDEX CONCURRENTLY idx_users_email ON users(email)")

// Data migrations with parameters
s.Raw("INSERT INTO settings (key, value) VALUES ($1, $2)", "version", "1.0")
s.Raw("UPDATE users SET status = $1 WHERE created_at < $2", "legacy", "2020-01-01")
```

## Migration Example

```go
// jone/migrations/20260123000000_create_users/migration.go
package m20260123000000

import "github.com/Grandbusta/jone"

func Up(s *jone.Schema) {
    s.CreateTable("users", func(t *jone.Table) {
        t.Increments("id")
        t.String("email").NotNullable().Unique()
        t.String("password_hash").NotNullable()
        t.String("name").Length(100)
        t.Boolean("verified").Default(false)
        t.Timestamps()
    })

    s.CreateTable("posts", func(t *jone.Table) {
        t.Increments("id")
        t.BigInt("user_id").NotNullable()
        t.String("title").NotNullable()
        t.Text("content")
        t.Timestamps()

        t.Foreign("user_id").References("users", "id").OnDelete("CASCADE")
        t.Index("user_id")
    })
}

func Down(s *jone.Schema) {
    s.DropTableIfExists("posts")
    s.DropTableIfExists("users")
}
```

## Supported Databases

| Database | Driver Package | Status |
|----------|----------------|--------|
| PostgreSQL | `github.com/jackc/pgx/v5/stdlib` | ✅ Supported |
| MySQL | `github.com/go-sql-driver/mysql` | 🚧 In Progress |
| SQLite | `github.com/mattn/go-sqlite3` | 🚧 In Progress |

## Contributing

Contributions are welcome! Please see our [Contributing Guide](CONTRIBUTING.md) for more details.

## License

MIT
