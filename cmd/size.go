package cmd

import (
	"fmt"
	"os"

	"amurru/filetools/internal/output"
	"amurru/filetools/internal/report"
	"amurru/filetools/internal/scan"
	"github.com/spf13/cobra"
)

// sizeCmd represents the size command
var sizeCmd = &cobra.Command{
	Use:   "size [directory]",
	Short: "Analyze file sizes and report the largest files",
	Long: `Analyze the size of the files in a directory tree.

This command traverses the specified directory (or current directory if none
provided) and reports how much space the files take up. It provides:

- Total file count, total size and directory count
- A distribution of the files over size ranges
- The largest files in the tree

Use --top to control how many of the largest files are listed.

If the directory is not specified, the current directory will be used.
`,
	Run: runSize,
}

var sizeTop int

func init() {
	rootCmd.AddCommand(sizeCmd)

	sizeCmd.Flags().IntVar(&sizeTop, "top", 10, "Number of largest files to report")
}

// runSize executes the size command
func runSize(cmd *cobra.Command, args []string) {
	// Validate the number of largest files to report
	if sizeTop < 0 {
		fmt.Fprintf(os.Stderr, "Error: --top must not be negative\n")
		os.Exit(1)
	}

	// Create progress indicator
	progress := newProgressIndicator(cmd)
	defer progress.Finish()

	result, err := scan.AnalyzeSizes(scan.SizeOptions{
		Dir:         dirArg(args),
		Top:         sizeTop,
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
		{Name: "top", Value: fmt.Sprintf("%d", sizeTop)},
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

	result.Metadata = newMetadata("size", flags)

	// Output the results
	if err := formatter.FormatSizeReport(result, writer); err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
		os.Exit(1)
	}
}
