package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var migrateUpCmd = &cobra.Command{
	Use:   "migrate:up",
	Short: "Runs all pending migrations",
	Long:  `Runs all pending migrations by generating and executing a runner`,
	Run:   migrateUpJone,
}

func migrateUpJone(cmd *cobra.Command, args []string) {
	if err := runMigrations("migrate:up"); err != nil {
		fmt.Printf("Error running migrations: %v\n", err)
		os.Exit(1)
	}
}
