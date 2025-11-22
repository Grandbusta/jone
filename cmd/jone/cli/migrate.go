package cli

import "github.com/spf13/cobra"

var migrateMakeCmd = &cobra.Command{
	Use:   "migrate:make",
	Short: "Migrates the database",
	Long:  `Migrates the database`,
	// Run:   migrateJone,
}
