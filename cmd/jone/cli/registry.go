package cli

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"

	"github.com/Grandbusta/jone/cmd/jone/templates"
)

// RegenerateRegistry scans the migrations folder and regenerates registry/registry.go.
func RegenerateRegistry(projectRoot string) error {
	migrationsRoot := filepath.Join(projectRoot, MigrationsPath)
	entries, err := os.ReadDir(migrationsRoot)
	if err != nil {
		return fmt.Errorf("reading migrations directory: %w", err)
	}

	modulePath := ReadModulePath(projectRoot)
	if modulePath == "" {
		return fmt.Errorf("could not read module path from go.mod")
	}

	var migrations []templates.MigrationInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == "registry" {
			continue
		}
		if MigrationDirPattern.MatchString(name) {
			migrations = append(migrations, templates.MigrationInfo{
				Name:       name,
				Alias:      aliasFromFolder(name),
				ImportPath: modulePath + "/" + MigrationsPath + "/" + name,
			})
		}
	}

	return writeRegistryFile(migrationsRoot, migrations)
}

func writeRegistryFile(migrationsRoot string, migrations []templates.MigrationInfo) error {
	regDir := filepath.Join(migrationsRoot, "registry")
	if err := os.MkdirAll(regDir, 0o755); err != nil {
		return fmt.Errorf("mkdir registry: %w", err)
	}

	content, err := templates.RenderRegistry(templates.RegistryData{
		RuntimePackage: RuntimePackage,
		Migrations:     migrations,
	})
	if err != nil {
		return fmt.Errorf("rendering registry template: %w", err)
	}

	// gofmt the output
	formatted, err := format.Source(content)
	if err != nil {
		// If formatting fails, write unformatted for debugging
		dest := filepath.Join(regDir, "registry.go")
		if writeErr := os.WriteFile(dest, content, 0o644); writeErr != nil {
			return fmt.Errorf("format error: %v; write fallback error: %v", err, writeErr)
		}
		return fmt.Errorf("format error: %w (wrote unformatted file)", err)
	}

	dest := filepath.Join(regDir, "registry.go")
	if err := os.WriteFile(dest, formatted, 0o644); err != nil {
		return fmt.Errorf("write registry.go: %w", err)
	}

	return nil
}

// aliasFromFolder extracts a valid Go identifier from a migration folder name.
// e.g., "20260114035749_add_users" -> "m20260114035749"
func aliasFromFolder(folder string) string {
	if idx := strings.Index(folder, "_"); idx > 0 {
		return "m" + folder[:idx]
	}
	return "m" + folder
}
