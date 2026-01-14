package cli

import "regexp"

// Paths relative to project root
const (
	JoneFolderPath = "jone"
	JoneFilePath   = "jone/jonefile.go"
	MigrationsPath = "jone/migrations"
)

// RuntimePackage is the import path for the jone library
const RuntimePackage = "github.com/Grandbusta/jone"

// MigrationDirPattern matches migration folder names (e.g., "20260114035749_add_users")
var MigrationDirPattern = regexp.MustCompile(`^\d+_`)
