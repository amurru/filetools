//go:build gui

package cmd

import (
	"fmt"
	"os"

	"github.com/amurru/filetools/internal/gui"
	"github.com/spf13/cobra"
)

// guiCmd represents the gui command
var guiCmd = &cobra.Command{
	Use:   "gui",
	Short: "Open the graphical interface",
	Long: `Open the graphical interface of filetools.

The interface offers the same tools as the command line subcommands: the
duplicate file search, the directory statistics, the size analysis, the
batch rename and the file organizer. Every tool shows a preview of what it
would do before it touches the filesystem.

This command is only available in builds made with the "gui" build tag.
`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := gui.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(guiCmd)
}
