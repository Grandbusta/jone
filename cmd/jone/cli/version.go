package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is the current version of jone.
// Update this before each release.
var Version = "v0.1.1"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of jone",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("jone version %s\n", Version)
	},
}
