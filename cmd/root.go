package cmd

import (
	"os"

	"amurru/filetools/internal/output"
	"github.com/spf13/cobra"
)

// outputFormat represents the desired output format
var outputFormat string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "filetools",
	Short: "Command-line tools for efficient file management and analysis",
	Long: `
A comprehensive suite of command-line utilities for efficient file management and analysis, built with Go for performance and reliability.

Filetools offers a range of commands to help with file operations such as searching, copying, moving, and analyzing files across different directories and formats. It is designed to be fast, lightweight, and easy to use for developers and system administrators.

`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.filetools.yaml)")

	// Output format flags
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "text", "Output format: text, json, xml, html")
	rootCmd.PersistentFlags().BoolP("json", "j", false, "Output in JSON format (shortcut for -o json)")
	rootCmd.PersistentFlags().BoolP("xml", "x", false, "Output in XML format (shortcut for -o xml)")
	rootCmd.PersistentFlags().BoolP("html", "w", false, "Output in HTML format (shortcut for -o html)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// getOutputFormat determines the output format based on flags
func getOutputFormat(cmd *cobra.Command) output.OutputFormat {
	// Check shortcut flags first (higher priority)
	if jsonFlag, _ := cmd.Flags().GetBool("json"); jsonFlag {
		return output.FormatJSON
	}
	if xmlFlag, _ := cmd.Flags().GetBool("xml"); xmlFlag {
		return output.FormatXML
	}
	if htmlFlag, _ := cmd.Flags().GetBool("html"); htmlFlag {
		return output.FormatHTML
	}

	// Fall back to the output flag
	switch outputFormat {
	case "json":
		return output.FormatJSON
	case "xml":
		return output.FormatXML
	case "html":
		return output.FormatHTML
	case "text":
		fallthrough
	default:
		return output.FormatText
	}
}
