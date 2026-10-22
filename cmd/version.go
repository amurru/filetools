package cmd

import (
	"fmt"

	"amurru/filetools/internal/version"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints the version",
	Long:  "Prints the version of the filetools command line tool.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf(
			"version: %s\ncommit: %s\ncommit-date: %s\ntree-state: %s\n",
			version.String(),
			version.CommitString(),
			version.CommitDateString(),
			version.TreeStateString(),
		)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
