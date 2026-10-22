package cmd

import (
	"fmt"
	"io"
	"os"
	"time"

	"amurru/filetools/internal/output"
	"amurru/filetools/internal/progress"
	"amurru/filetools/internal/report"
	"amurru/filetools/internal/version"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// outputFormat represents the desired output format
var outputFormat string

// outputFile represents the output file path (empty means stdout)
var outputFile string

// excludeFilePatterns represents file exclusion patterns (comma-separated)
var excludeFilePatterns string

// excludeDirPatterns represents directory exclusion patterns (comma-separated)
var excludeDirPatterns string

// noProgress disables progress bar indicators
var noProgress bool

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

	// Progress flag
	rootCmd.PersistentFlags().BoolVar(&noProgress, "no-progress", false, "Disable progress bar indicators")
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

// newProgressIndicator creates the progress reporter for a command.
//
// Progress is only reported when human readable output goes to a terminal:
// a structured format written to a file would be broken by the extra lines
// and a redirected stderr is usually consumed by another program.
func newProgressIndicator(cmd *cobra.Command) progress.Indicator {
	if disabled, _ := cmd.Flags().GetBool("no-progress"); disabled {
		return progress.Nop{}
	}

	// Only disable progress for structured file output (to avoid breaking parsers)
	// Allow progress for text file output since it's human-readable
	if outputFile != "" && getOutputFormat(cmd) != output.FormatText {
		return progress.Nop{}
	}

	// Disable progress bars if stderr is not a terminal
	if !isTerminal(os.Stderr) {
		return progress.Nop{}
	}

	return progress.NewText(os.Stderr)
}

// isTerminal reports whether the given file is attached to a terminal.
func isTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// newMetadata builds the metadata block describing a command run.
func newMetadata(subCommand string, flags []report.Flag) *report.Metadata {
	return &report.Metadata{
		ToolName:    "filetools",
		SubCommand:  subCommand,
		Flags:       flags,
		Version:     version.String(),
		GeneratedAt: time.Now().Format(time.RFC3339),
	}
}

// dirArg returns the directory to operate on, defaulting to the current
// directory when no argument was given.
func dirArg(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return "."
}
