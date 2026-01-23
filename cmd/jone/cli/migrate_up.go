package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var migrateUpCmd = &cobra.Command{
	Use:   "migrate:up [migration_name]",
	Short: "Runs the next pending migration or a specific one",
	Long:  `Runs the next pending migration. If a migration name is provided, runs that specific migration.`,
	Run:   migrateUpJone,
}

func migrateUpJone(cmd *cobra.Command, args []string) {
	execParams := RunExecParams{
		Command: "migrate:up",
		Args:    args,
	}
	if err := runMigrations(execParams); err != nil {
		fmt.Printf("Error running migration: %v\n", err)
		os.Exit(1)
	}
}
