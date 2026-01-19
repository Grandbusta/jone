package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var migrateLatestCmd = &cobra.Command{
	Use:   "migrate:latest",
	Short: "Runs all pending migrations",
	Long:  `Runs all pending migrations by generating and executing a runner`,
	Run:   migrateLatestJone,
}

func migrateLatestJone(cmd *cobra.Command, args []string) {
	if err := runMigrations("migrate:latest"); err != nil {
		fmt.Printf("Error running migrations: %v\n", err)
		os.Exit(1)
	}
}
