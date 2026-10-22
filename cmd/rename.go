package cmd

import (
	"fmt"
	"os"

	"github.com/amurru/filetools/internal/output"
	"github.com/amurru/filetools/internal/report"
	"github.com/amurru/filetools/internal/scan"
	"github.com/spf13/cobra"
)

// renameCmd represents the rename command
var renameCmd = &cobra.Command{
	Use:   "rename [directory]",
	Short: "Rename files in a directory using pattern matching and sed-like replacements",
	Long: `Rename files in a directory using pattern matching and sed-like replacements.

This command will traverse the specified directory (or current directory if none provided)
and rename files that match the given pattern using a sed-style replacement expression.

Note: Glob patterns must be quoted to prevent shell expansion.

Examples:
    # Add prefix to all JPG files
    filetools rename --match "*.jpg" --sed "s/^/vacation_/" /photos

    # Remove suffix from files
    filetools rename --match "*_old.jpg" --sed "s/_old//" /photos

    # General replacement
    filetools rename --match "*.txt" --sed "s/draft/final/g" /docs

Dry-run mode is enabled by default for safety. Use --force to perform actual renames.
`,
	Run: runRename,
}

var (
	matchPattern   string
	sedExpression  string
	forceOverwrite bool
)

func init() {
	rootCmd.AddCommand(renameCmd)

	// Command-specific flags
	renameCmd.Flags().StringVar(&matchPattern, "match", "", "File pattern to match (glob, required)")
	renameCmd.Flags().StringVar(&sedExpression, "sed", "", "Sed-style replacement expression (e.g., s/old/new/g, required)")
	renameCmd.Flags().BoolVar(&forceOverwrite, "force", false, "Perform actual renames (disables dry-run)")
	_ = renameCmd.MarkFlagRequired("match")
	_ = renameCmd.MarkFlagRequired("sed")
}

// runRename executes the rename command
func runRename(cmd *cobra.Command, args []string) {
	// Create progress indicator
	progress := newProgressIndicator(cmd)
	defer progress.Finish()

	result, err := scan.Rename(scan.RenameOptions{
		Dir:         dirArg(args),
		Match:       matchPattern,
		Sed:         sedExpression,
		DryRun:      !forceOverwrite,
		Overwrite:   forceOverwrite,
		ExcludeFile: excludeFilePatterns,
		ExcludeDir:  excludeDirPatterns,
	}, progress)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Get output writer (file or stdout)
	writer, cleanup, err := getOutputWriter(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	// Get output format and create formatter
	format := getOutputFormat(cmd)
	formatter := output.NewFormatter(format)

	// Create metadata
	flags := []report.Flag{
		{Name: "match", Value: matchPattern},
		{Name: "sed", Value: sedExpression},
		{Name: "dry-run", Value: fmt.Sprintf("%t", result.DryRun)},
		{Name: "output", Value: string(format)},
	}

	// Add output flag if specified
	if outputFile != "" {
		flags = append(flags, report.Flag{Name: "file", Value: outputFile})
	}

	// Add exclusion flags if specified
	if excludeFilePatterns != "" {
		flags = append(flags, report.Flag{Name: "exclude-file", Value: excludeFilePatterns})
	}
	if excludeDirPatterns != "" {
		flags = append(flags, report.Flag{Name: "exclude-dir", Value: excludeDirPatterns})
	}

	result.Metadata = newMetadata("rename", flags)

	// Output the results
	if err := formatter.FormatRename(result, writer); err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
		os.Exit(1)
	}
}
