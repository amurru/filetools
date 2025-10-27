package cmd

import (
	"fmt"
	"io"
	"os"

	"amurru/filetools/internal/output"
	"github.com/spf13/cobra"
)

// outputFormat represents the desired output format
var outputFormat string

// outputFile represents the output file path (empty means stdout)
var outputFile string

// excludeFilePatterns represents file exclusion patterns (comma-separated)
var excludeFilePatterns string

// excludeDirPatterns represents directory exclusion patterns (comma-separated)
var excludeDirPatterns string

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

	// Output file flag
	rootCmd.PersistentFlags().StringVarP(&outputFile, "file", "f", "", "Output file (default: stdout)")

	// Exclusion flags
	rootCmd.PersistentFlags().StringVar(&excludeFilePatterns, "exclude-file", "", "Exclude files matching patterns (comma-separated globs or file types, e.g., '*.log,*.tmp')")
	rootCmd.PersistentFlags().StringVar(&excludeDirPatterns, "exclude-dir", "", "Exclude directories matching patterns (comma-separated globs, e.g., 'node_modules,*.git')")

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

// getOutputWriter returns an io.Writer for output (file or stdout) and a cleanup function
func getOutputWriter(cmd *cobra.Command) (io.Writer, func(), error) {
	fileFlag, _ := cmd.Flags().GetString("file")
	if fileFlag == "" {
		// No file specified, use stdout
		return os.Stdout, func() {}, nil
	}

	// Create output file
	file, err := os.Create(fileFlag)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create output file '%s': %w", fileFlag, err)
	}

	// Return file writer and cleanup function
	return file, func() { file.Close() }, nil
}
