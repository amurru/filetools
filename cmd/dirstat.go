package cmd

import (
	"fmt"
	"os"

	"amurru/filetools/internal/output"
	"amurru/filetools/internal/report"
	"amurru/filetools/internal/scan"
	"github.com/spf13/cobra"
)

// dirstatCmd represents the dirstat command
var dirstatCmd = &cobra.Command{
	Use:   "dirstat [directory]",
	Short: "Analyze directory and subdirectories for file statistics",
	Long: `Analyze a directory and its subdirectories to provide comprehensive file statistics.

This command will traverse the specified directory (or current directory if none provided)
and collect statistics about file sizes, types, and directory utilization. It provides:

- Total file count and size
- File type breakdown with counts and percentages
- Directory breakdown with file counts and size percentages
- Information about the largest file

The output includes percentages relative to the total directory utilization.

If the directory is not specified, the current directory will be used.
`,
	Run: runDirstat,
}

func init() {
	rootCmd.AddCommand(dirstatCmd)
}

// runDirstat executes the dirstat command
func runDirstat(cmd *cobra.Command, args []string) {
	// Create progress indicator
	progress := newProgressIndicator(cmd)
	defer progress.Finish()

	result, err := scan.AnalyzeDirStat(scan.DirStatOptions{
		Dir:         dirArg(args),
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

	result.Metadata = newMetadata("dirstat", flags)

	// Output the results
	if err := formatter.FormatDirStat(result, writer); err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
		os.Exit(1)
	}
}
