package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

/*
migrate:make needs an argument of the migration name e.g jone migrate:make create_users, if there is no argument, it will ask the user to enter the name of the migration
It checks if the jone folder exists and then check if the jonefile.go exists, if not it tells the user to run jone init first.
If there is a jone folder and jonefile.go, it checks for the migrations folder and if it does not exists, it creates it
*/
var migrateMakeCmd = &cobra.Command{
	Use:   "migrate:make",
	Short: "Migrates the database",
	Long:  `Migrates the database`,
	Run:   migrateJone,
}

const (
	joneFolderPath     = "jone"
	jonefilePath       = "jone/jonefile.go"
	migrationsPath     = "jone/migrations"
	registryFolderPath = "jone/migrations/registry"
	registryFilePath   = "jone/migrations/registry/registry.go"
)

func migrateJone(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		fmt.Println("Please provide a migration name")
		return
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get working directory: %v\n", err)
		os.Exit(1)
	}

	if _, err := os.Stat(joneFolderPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println("jone folder not found, please run jone init first")
			return
		} else {
			fmt.Printf("Error checking jone folder: %v\n", err)
			return
		}
	}

	if _, err := os.Stat(jonefilePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println("jonefile.go not found in jone folder. Please run jone init first")
			return
		} else {
			fmt.Printf("Error checking jonefile.go: %v\n", err)
			return
		}
	}

	if _, err := os.Stat(migrationsPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			os.Mkdir(migrationsPath, 0755)
		} else {
			fmt.Printf("Error checking migrations folder: %v\n", err)
			return
		}
	}

	createMigration(cwd, args[0])

	// if _, err := os.Stat(registryFolderPath); err != nil {
	// 	if errors.Is(err, os.ErrNotExist) {
	// 		os.Mkdir(registryFolderPath, 0755)
	// 	} else {
	// 		fmt.Printf("Error checking registry folder: %v\n", err)
	// 		return
	// 	}
	// }

	// if _, err := os.Stat(registryFilePath); err != nil {
	// 	if errors.Is(err, os.ErrNotExist) {
	// 		os.Create(registryFilePath)
	// 	} else {
	// 		fmt.Printf("Error checking registry file: %v\n", err)
	// 		return
	// 	}
	// }

	// Write registry.go contents
	// file, err := os.OpenFile(registryFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	// if err != nil {
	// 	fmt.Printf("Error opening registry.go: %v\n", err)
	// 	return
	// }
	// defer file.Close()

	// registryFileContents := `

	// `

}

func createMigration(cwd string, name string) {
	ts := time.Now().UTC().Format("20060102150405")
	folderName := fmt.Sprintf("%s_%s", ts, name)
	folderPath := filepath.Join(cwd, migrationsPath, folderName)

	if err := os.Mkdir(folderPath, 0755); err != nil {
		fmt.Printf("Error creating migration: %v\n", err)
		return
	}

	stub := fmt.Sprintf(`package mig

import (
	"context"

	%q
)

func Up(ctx context.Context, s jone.Schema) error {
	// TODO: implement Up migration
	return nil
}

func Down(ctx context.Context, s jone.Schema) error {
	// TODO: implement Down migration
	return nil
}
`, "github.com/Grandbusta/jone")

	if err := os.WriteFile(filepath.Join(folderPath, "mig.go"), []byte(stub), 0o644); err != nil {
		fmt.Printf("Error writing migration file: %v\n", err)
		return
	}

}
