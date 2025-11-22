package cli

import (
	"errors"
	"fmt"
	"os"

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
	joneFolderPath = "jone"
	jonefilePath   = "jone/jonefile.go"
	migrationsPath = "jone/migrations"
)

func migrateJone(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		fmt.Println("Please provide a migration name")
		return
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
			os.Mkdir("jone/migrations", 0755)
		} else {
			fmt.Printf("Error checking migrations folder: %v\n", err)
			return
		}
	}

	fmt.Printf("migrate:make running perfectly now for %s\n", args[0])

}
