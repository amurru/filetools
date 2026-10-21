package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"

	"amurru/filetools/internal/exclusions"
	"amurru/filetools/internal/output"
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

// sizeRange describes a bucket of files grouped by size
type sizeRange struct {
	label string
	min   int64
	max   int64
}

// sizeRanges are the buckets used for the size distribution, ordered from
// smallest to largest. The last range has no upper bound.
var sizeRanges = []sizeRange{
	{"< 1 KB", 0, 1024},
	{"1 KB - 10 KB", 1024, 10 * 1024},
	{"10 KB - 100 KB", 10 * 1024, 100 * 1024},
	{"100 KB - 1 MB", 100 * 1024, 1024 * 1024},
	{"1 MB - 10 MB", 1024 * 1024, 10 * 1024 * 1024},
	{"10 MB - 100 MB", 10 * 1024 * 1024, 100 * 1024 * 1024},
	{"> 100 MB", 100 * 1024 * 1024, 0},
}

// sizeBucketLabel returns the label of the size range a file belongs to
func sizeBucketLabel(size int64) string {
	for _, r := range sizeRanges {
		if size >= r.min && (r.max == 0 || size < r.max) {
			return r.label
		}
	}
	return sizeRanges[len(sizeRanges)-1].label
}

// analyzeSizes traverses the directory and collects size statistics
func analyzeSizes(rootDir string, top int, fileMatchers, dirMatchers []exclusions.ExclusionMatcher, progress output.ProgressIndicator) (*output.SizeReportResult, error) {
	totalFiles := 0
	totalDirs := 0
	totalSize := int64(0)
	var exclusionsList []output.Exclusion

	bucketCounts := make([]int, len(sizeRanges))
	bucketSizes := make([]int64, len(sizeRanges))
	var files []output.FileInfo

	progress.Start(-1, "Analyzing file sizes")

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not access %s: %v\n", path, err)
			return nil
		}

		// Skip the root directory itself
		if path == rootDir {
			return nil
		}

		isDir := d.IsDir()

		// Get relative path for exclusion checking
		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}

		// Check for exclusions
		if exclusion := exclusions.CheckExclusions(relPath, isDir, fileMatchers, dirMatchers); exclusion != nil {
			exclusionsList = append(exclusionsList, *exclusion)
			if isDir {
				return filepath.SkipDir
			}
			return nil
		}

		if isDir {
			totalDirs++
			return nil
		}

		info, err := d.Info()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not stat %s: %v\n", path, err)
			return nil
		}
		size := info.Size()

		totalFiles++
		totalSize += size

		files = append(files, output.FileInfo{
			Name: d.Name(),
			Size: size,
			Path: relPath,
		})

		for i, r := range sizeRanges {
			if size >= r.min && (r.max == 0 || size < r.max) {
				bucketCounts[i]++
				bucketSizes[i] += size
				break
			}
		}

		progress.Increment()
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Build the size distribution
	var buckets []output.SizeBucket
	for i, r := range sizeRanges {
		if bucketCounts[i] == 0 {
			continue
		}
		percentage := 0.0
		if totalSize > 0 {
			percentage = float64(bucketSizes[i]) / float64(totalSize) * 100
		}
		buckets = append(buckets, output.SizeBucket{
			Label:      r.label,
			FileCount:  bucketCounts[i],
			TotalSize:  bucketSizes[i],
			Percentage: percentage,
		})
	}

	// Sort the files by size, largest first, and keep the requested amount
	sort.Slice(files, func(i, j int) bool {
		if files[i].Size != files[j].Size {
			return files[i].Size > files[j].Size
		}
		return files[i].Path < files[j].Path
	})

	if top > 0 && len(files) > top {
		files = files[:top]
	}

	result := &output.SizeReportResult{
		TotalFiles:   totalFiles,
		TotalSize:    totalSize,
		TotalDirs:    totalDirs,
		LargestFiles: files,
		Buckets:      buckets,
		Exclusions:   exclusionsList,
	}

	return result, nil
}

// runSize executes the size command
func runSize(cmd *cobra.Command, args []string) {
	rootDir := "."
	if len(args) > 0 {
		rootDir = args[0]
	}

	// Validate the number of largest files to report
	if sizeTop < 0 {
		fmt.Fprintf(os.Stderr, "Error: --top must not be negative\n")
		os.Exit(1)
	}

	// Validate directory exists
	if info, err := os.Stat(rootDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	} else if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: %s is not a directory\n", rootDir)
		os.Exit(1)
	}

	// Parse exclusion patterns
	fileMatchers := exclusions.ParseExclusions(excludeFilePatterns, true)
	dirMatchers := exclusions.ParseExclusions(excludeDirPatterns, false)

	// Create progress indicator
	progress := output.NewProgressIndicator(cmd)
	defer progress.Finish()

	result, err := analyzeSizes(rootDir, sizeTop, fileMatchers, dirMatchers, progress)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error analyzing sizes: %v\n", err)
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
	flags := []output.Flag{
		{Name: "top", Value: fmt.Sprintf("%d", sizeTop)},
		{Name: "output", Value: string(format)},
	}

	// Add output flag
	if outputFile != "" {
		flags = append(flags, output.Flag{Name: "file", Value: outputFile})
	}

	// Add exclusion flags if specified
	if excludeFilePatterns != "" {
		flags = append(flags, output.Flag{Name: "exclude-file", Value: excludeFilePatterns})
	}
	if excludeDirPatterns != "" {
		flags = append(flags, output.Flag{Name: "exclude-dir", Value: excludeDirPatterns})
	}

	metadata := &output.Metadata{
		ToolName:    "filetools",
		SubCommand:  "size",
		Flags:       flags,
		Version:     version,
		GeneratedAt: time.Now().Format(time.RFC3339),
	}

	result.Metadata = metadata

	// Output the results
	if err := formatter.FormatSizeReport(result, writer); err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
		os.Exit(1)
	}
}
