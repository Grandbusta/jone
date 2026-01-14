package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var migrateDownCmd = &cobra.Command{
	Use:   "migrate:down",
	Short: "Rolls back migrations",
	Long:  `Rolls back migrations by generating and executing a runner`,
	Run:   migrateDownJone,
}

func migrateDownJone(cmd *cobra.Command, args []string) {
	if err := runMigrations("migrate:down"); err != nil {
		fmt.Printf("Error rolling back migrations: %v\n", err)
		os.Exit(1)
	}
}
