package cmd

import (
	"fmt"
	"os"

	"github.com/amurru/filetools/internal/output"
	"github.com/amurru/filetools/internal/report"
	"github.com/amurru/filetools/internal/scan"
	"github.com/spf13/cobra"
)

// dupfindCmd represents the dupfind command
var dupfindCmd = &cobra.Command{
	Use:   "dupfind",
	Short: "Find duplicate files",
	Long: `Find duplicate files in a directory tree.

This command will find duplicate files in a directory tree. It will
compare the hashes of each file and report any files that are
identical. File names are not important.

The output will be a list of files that are identical. The files will
be listed in the order that they were found, with the first file
listed being the first duplicate. The files will be listed in
alphabetical order by filename.

Large files are fingerprinted on their first bytes first, so only the
files that can still be duplicates are read completely.

If the directory is not specified, the current directory will be used.
`,
	Run: runDupfind,
}

var (
	hashAlgorithm string
	minSize       string
	maxSize       string
)

func init() {
	rootCmd.AddCommand(dupfindCmd)

	// Add hash algorithm flag
	dupfindCmd.Flags().StringVarP(&hashAlgorithm, "hash", "H", "md5", "Hash algorithm to use (md5, sha1, sha256)")

	// Add size range flags
	dupfindCmd.Flags().StringVar(&minSize, "min-size", "", "Only consider files of at least this size (e.g. 1MB, 512K)")
	dupfindCmd.Flags().StringVar(&maxSize, "max-size", "", "Only consider files of at most this size (e.g. 100MB)")
}

// runDupfind executes the dupfind command
func runDupfind(cmd *cobra.Command, args []string) {
	// Parse the size range filters
	minSizeBytes, err := scan.ParseSize(minSize)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: --min-size: %v\n", err)
		os.Exit(1)
	}
	maxSizeBytes, err := scan.ParseSize(maxSize)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: --max-size: %v\n", err)
		os.Exit(1)
	}

	// Create progress indicator
	progress := newProgressIndicator(cmd)
	defer progress.Finish()

	result, err := scan.FindDuplicates(scan.DuplicateOptions{
		Dir:         dirArg(args),
		Algorithm:   hashAlgorithm,
		MinSize:     minSizeBytes,
		MaxSize:     maxSizeBytes,
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
		{Name: "hash", Value: hashAlgorithm},
		{Name: "output", Value: string(format)},
	}

	// Add size range flags if specified
	if minSize != "" {
		flags = append(flags, report.Flag{Name: "min-size", Value: minSize})
	}
	if maxSize != "" {
		flags = append(flags, report.Flag{Name: "max-size", Value: maxSize})
	}

	// Add file flag if specified
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

	result.Metadata = newMetadata("dupfind", flags)

	// Output the results
	if err := formatter.FormatDuplicates(result, writer); err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
		os.Exit(1)
	}
}
