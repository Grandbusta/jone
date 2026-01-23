package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var migrateListCmd = &cobra.Command{
	Use:   "migrate:list",
	Short: "Lists all migrations with their status",
	Long:  `Lists all registered migrations showing which are applied and which are pending.`,
	Run:   migrateListJone,
}

func migrateListJone(cmd *cobra.Command, args []string) {
	execParams := RunExecParams{
		Command: "migrate:list",
	}
	if err := runMigrations(execParams); err != nil {
		fmt.Printf("Error listing migrations: %v\n", err)
		os.Exit(1)
	}
}
