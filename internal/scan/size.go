package scan

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"amurru/filetools/internal/exclusions"
	"amurru/filetools/internal/progress"
	"amurru/filetools/internal/report"
)

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

// SizeOptions configures a size analysis.
type SizeOptions struct {
	// Dir is the directory tree to analyze.
	Dir string
	// Top is the number of largest files to report. A value of zero or less
	// reports none.
	Top int
	// ExcludeFile and ExcludeDir are comma separated exclusion patterns.
	ExcludeFile string
	ExcludeDir  string
}

// AnalyzeSizes collects size statistics for a directory tree.
func AnalyzeSizes(opts SizeOptions, p progress.Indicator) (*report.SizeReportResult, error) {
	if err := validateDir(opts.Dir); err != nil {
		return nil, err
	}

	totalFiles := 0
	totalSize := int64(0)
	totalDirs := 0
	var exclusionsList []report.Exclusion

	bucketCounts := make([]int, len(sizeRanges))
	bucketSizes := make([]int64, len(sizeRanges))
	var files []report.FileInfo

	m := newMatchers(opts.ExcludeFile, opts.ExcludeDir)

	p.Start(-1, "Analyzing file sizes")

	err := filepath.WalkDir(opts.Dir, func(path string, d fs.DirEntry, err error) error {
		progress.WaitIfPaused(p)
		if progress.IsCancelled(p) {
			return progress.ErrCancelled
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not access %s: %v\n", path, err)
			return nil
		}

		// Skip the root directory itself
		if path == opts.Dir {
			return nil
		}

		isDir := d.IsDir()

		// Get relative path for exclusion checking
		relPath, err := filepath.Rel(opts.Dir, path)
		if err != nil {
			return err
		}

		// Check for exclusions
		if exclusion := exclusions.CheckExclusions(relPath, isDir, m.files, m.dirs); exclusion != nil {
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

		files = append(files, report.FileInfo{
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

		p.Increment()
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Build the size distribution
	var buckets []report.SizeBucket
	for i, r := range sizeRanges {
		if bucketCounts[i] == 0 {
			continue
		}
		percentage := 0.0
		if totalSize > 0 {
			percentage = float64(bucketSizes[i]) / float64(totalSize) * 100
		}
		buckets = append(buckets, report.SizeBucket{
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

	if opts.Top > 0 && len(files) > opts.Top {
		files = files[:opts.Top]
	}

	result := &report.SizeReportResult{
		TotalFiles:   totalFiles,
		TotalSize:    totalSize,
		TotalDirs:    totalDirs,
		LargestFiles: files,
		Buckets:      buckets,
		Exclusions:   exclusionsList,
	}

	return result, nil
}

// SizeBucketLabel returns the label of the size range a file belongs to
func SizeBucketLabel(size int64) string {
	for _, r := range sizeRanges {
		if size >= r.min && (r.max == 0 || size < r.max) {
			return r.label
		}
	}
	return sizeRanges[len(sizeRanges)-1].label
}

// ParseSize parses a human readable size such as "1024", "512K" or "10MB"
// into a number of bytes
func ParseSize(value string) (int64, error) {
	trimmed := strings.ToUpper(strings.TrimSpace(value))
	if trimmed == "" {
		return 0, nil
	}

	multipliers := []struct {
		suffix     string
		multiplier int64
	}{
		{"TIB", 1 << 40}, {"TB", 1 << 40}, {"T", 1 << 40},
		{"GIB", 1 << 30}, {"GB", 1 << 30}, {"G", 1 << 30},
		{"MIB", 1 << 20}, {"MB", 1 << 20}, {"M", 1 << 20},
		{"KIB", 1 << 10}, {"KB", 1 << 10}, {"K", 1 << 10},
		{"B", 1},
	}

	number := trimmed
	multiplier := int64(1)

	for _, m := range multipliers {
		if strings.HasSuffix(trimmed, m.suffix) {
			number = strings.TrimSpace(strings.TrimSuffix(trimmed, m.suffix))
			multiplier = m.multiplier
			break
		}
	}

	parsed, err := strconv.ParseFloat(number, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size '%s': expected a number optionally followed by K, M, G or T", value)
	}

	if parsed < 0 {
		return 0, fmt.Errorf("invalid size '%s': must not be negative", value)
	}

	return int64(parsed * float64(multiplier)), nil
}
