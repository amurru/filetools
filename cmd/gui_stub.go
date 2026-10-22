//go:build !gui

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// guiCmd represents the gui command in builds without the graphical
// interface. It explains how to get a build that has one.
var guiCmd = &cobra.Command{
	Use:   "gui",
	Short: "Open the graphical interface",
	Long: `Open the graphical interface of filetools.

This binary was built without the graphical interface. Rebuild it with the
"gui" build tag to enable it:

    go build -tags gui .

On Linux the graphical interface also needs cgo and the X11, Wayland and EGL
development packages.
`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(os.Stderr, "This build of filetools has no graphical interface.")
		fmt.Fprintln(os.Stderr, "Rebuild it with: go build -tags gui .")
		os.Exit(1)
	},
}

func init() {
	rootCmd.AddCommand(guiCmd)
}
