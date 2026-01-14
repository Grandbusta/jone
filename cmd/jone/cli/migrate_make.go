package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Grandbusta/jone/cmd/jone/templates"
	"github.com/spf13/cobra"
)

/*
migrate:make needs an argument of the migration name e.g jone migrate:make create_users,
if there is no argument, it will ask the user to enter the name of the migration.
It checks if the jone folder exists and then check if the jonefile.go exists,
if not it tells the user to run jone init first.
If there is a jone folder and jonefile.go, it checks for the migrations folder
and if it does not exist, it creates it.
*/
var migrateMakeCmd = &cobra.Command{
	Use:   "migrate:make",
	Short: "Creates a new migration",
	Long:  `Creates a new migration file in the jone/migrations folder`,
	Run:   migrateMakeJone,
}

func migrateMakeJone(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		fmt.Println("Please provide a migration name")
		return
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get working directory: %v\n", err)
		os.Exit(1)
	}

	// Check jone folder exists
	if _, err := os.Stat(JoneFolderPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println("jone folder not found, please run jone init first")
			return
		}
		fmt.Printf("Error checking jone folder: %v\n", err)
		return
	}

	// Check jonefile.go exists
	if _, err := os.Stat(JoneFilePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println("jonefile.go not found in jone folder. Please run jone init first")
			return
		}
		fmt.Printf("Error checking jonefile.go: %v\n", err)
		return
	}

	// Create migrations folder if needed
	if _, err := os.Stat(MigrationsPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err := os.Mkdir(MigrationsPath, 0755); err != nil {
				fmt.Printf("Error creating migrations folder: %v\n", err)
				return
			}
		} else {
			fmt.Printf("Error checking migrations folder: %v\n", err)
			return
		}
	}

	if err := createMigration(cwd, args[0]); err != nil {
		fmt.Printf("Error creating migration: %v\n", err)
		return
	}

	if err := RegenerateRegistry(cwd); err != nil {
		fmt.Printf("Error regenerating registry: %v\n", err)
		return
	}

	fmt.Printf("Migration %s created successfully\n", args[0])
}

func createMigration(cwd string, name string) error {
	ts := time.Now().UTC().Format("20060102150405")
	folderName := fmt.Sprintf("%s_%s", ts, name)
	folderPath := filepath.Join(cwd, MigrationsPath, folderName)

	if err := os.Mkdir(folderPath, 0755); err != nil {
		return fmt.Errorf("creating migration folder: %w", err)
	}

	stub, err := templates.RenderMigration(templates.MigrationStubData{
		RuntimePackage: RuntimePackage,
	})
	if err != nil {
		return fmt.Errorf("rendering migration stub: %w", err)
	}

	migrationFile := filepath.Join(folderPath, "migration.go")
	if err := os.WriteFile(migrationFile, stub, 0o644); err != nil {
		return fmt.Errorf("writing migration file: %w", err)
	}

	return nil
}
