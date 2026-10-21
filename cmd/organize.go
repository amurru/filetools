package cmd

import (
	"fmt"
	"os"

	"amurru/filetools/internal/output"
	"amurru/filetools/internal/report"
	"amurru/filetools/internal/scan"
	"github.com/spf13/cobra"
)

// organizeCmd represents the organize command
var organizeCmd = &cobra.Command{
	Use:   "organize [directory]",
	Short: "Organize files into subdirectories by type, extension, date or size",
	Long: `Organize files into subdirectories grouped by a chosen criteria.

This command traverses the specified directory (or current directory if none
provided) and moves every file into a subdirectory named after the category
it belongs to. Categories are computed from the file extension, its
modification date or its size.

Supported criteria (--by):
  type       Broad categories such as images, video, audio, documents,
             archives, code, data and other
  extension  One directory per file extension, for example "png" or "jpg"
  date       One directory per month of the modification date, for example
             "2024-03"
  size       One directory per size range, for example "1 MB - 10 MB"

Files are moved into <destination>/<category>/<name>. The destination
defaults to the scanned directory. Paths in the report are relative to the
scanned directory for the source and to the destination for the target.

Dry-run mode is enabled by default for safety. Use --force to actually move
the files and --overwrite to replace files that already exist at the target.
`,
	Run: runOrganize,
}

var (
	organizeBy            string
	organizeDest          string
	organizeForce         bool
	organizeOverwrite     bool
	organizeIncludeHidden bool
	organizeCategory      string
)

func init() {
	rootCmd.AddCommand(organizeCmd)

	organizeCmd.Flags().StringVar(&organizeBy, "by", "type", "Group files by: type, extension, date, size")
	organizeCmd.Flags().StringVar(&organizeDest, "dest", "", "Destination directory for organized files (default: the scanned directory)")
	organizeCmd.Flags().BoolVar(&organizeForce, "force", false, "Move the files (disables dry-run)")
	organizeCmd.Flags().BoolVar(&organizeOverwrite, "overwrite", false, "Replace files that already exist at the target")
	organizeCmd.Flags().BoolVar(&organizeIncludeHidden, "include-hidden", false, "Also organize hidden files")
	organizeCmd.Flags().StringVar(&organizeCategory, "category", "", "Only organize files belonging to this category")
}

// runOrganize executes the organize command
func runOrganize(cmd *cobra.Command, args []string) {
	// Create progress indicator
	progress := newProgressIndicator(cmd)
	defer progress.Finish()

	result, err := scan.Organize(scan.OrganizeOptions{
		Source:        dirArg(args),
		Dest:          organizeDest,
		By:            organizeBy,
		DryRun:        !organizeForce,
		Overwrite:     organizeOverwrite,
		IncludeHidden: organizeIncludeHidden,
		Category:      organizeCategory,
		ExcludeFile:   excludeFilePatterns,
		ExcludeDir:    excludeDirPatterns,
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
		{Name: "by", Value: organizeBy},
		{Name: "dest", Value: organizeDest},
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

	result.Metadata = newMetadata("organize", flags)

	// Output the results
	if err := formatter.FormatOrganize(result, writer); err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
		os.Exit(1)
	}
}
