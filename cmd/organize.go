package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"amurru/filetools/internal/exclusions"
	"amurru/filetools/internal/output"
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

// fileTypeCategories maps a file extension to a broad category
var fileTypeCategories = map[string]string{
	// Images
	".jpg": "images", ".jpeg": "images", ".png": "images", ".gif": "images",
	".bmp": "images", ".webp": "images", ".svg": "images", ".tiff": "images",
	".tif": "images", ".ico": "images", ".heic": "images", ".raw": "images",
	// Video
	".mp4": "video", ".mkv": "video", ".avi": "video", ".mov": "video",
	".wmv": "video", ".flv": "video", ".webm": "video", ".m4v": "video",
	".mpg": "video", ".mpeg": "video",
	// Audio
	".mp3": "audio", ".wav": "audio", ".flac": "audio", ".aac": "audio",
	".ogg": "audio", ".m4a": "audio", ".wma": "audio", ".opus": "audio",
	// Documents
	".pdf": "documents", ".doc": "documents", ".docx": "documents",
	".txt": "documents", ".rtf": "documents", ".odt": "documents",
	".md": "documents", ".pages": "documents", ".epub": "documents",
	".ppt": "documents", ".pptx": "documents", ".xls": "documents",
	".xlsx": "documents", ".csv": "documents",
	// Archives
	".zip": "archives", ".tar": "archives", ".gz": "archives",
	".bz2": "archives", ".xz": "archives", ".7z": "archives",
	".rar": "archives", ".tgz": "archives",
	// Code
	".go": "code", ".py": "code", ".js": "code", ".ts": "code",
	".tsx": "code", ".jsx": "code", ".java": "code", ".c": "code",
	".h": "code", ".cpp": "code", ".hpp": "code", ".cs": "code",
	".rs": "code", ".rb": "code", ".php": "code", ".sh": "code",
	".html": "code", ".css": "code", ".vue": "code", ".swift": "code",
	".kt": "code", ".sql": "code", ".lua": "code",
	// Data
	".db": "data", ".sqlite": "data", ".sqlite3": "data", ".parquet": "data",
	".json": "data", ".xml": "data", ".yaml": "data", ".yml": "data",
	".toml": "data", ".ini": "data", ".log": "data",
}

// organizeOptions holds the settings of a single organize run
type organizeOptions struct {
	By            string
	DestDir       string
	DryRun        bool
	Overwrite     bool
	IncludeHidden bool
	Category      string
}

// categorizeFile determines the target category of a file for the given criteria
func categorizeFile(path string, info os.FileInfo, by string) string {
	ext := strings.ToLower(filepath.Ext(path))

	switch by {
	case "extension":
		if ext == "" {
			return "no-extension"
		}
		return strings.TrimPrefix(ext, ".")

	case "date":
		return info.ModTime().Format("2006-01")

	case "size":
		return sizeBucketLabel(info.Size())

	default: // "type"
		if category, ok := fileTypeCategories[ext]; ok {
			return category
		}
		return "other"
	}
}

// isSamePath reports whether two paths point at the same location
func isSamePath(a, b string) bool {
	absA, err := filepath.Abs(a)
	if err != nil {
		return false
	}
	absB, err := filepath.Abs(b)
	if err != nil {
		return false
	}
	return filepath.Clean(absA) == filepath.Clean(absB)
}

// organizeFiles traverses the directory and moves files into per-category subdirectories
func organizeFiles(rootDir string, opts organizeOptions, fileMatchers, dirMatchers []exclusions.ExclusionMatcher, progress output.ProgressIndicator) ([]output.OrganizeOperation, []output.OrganizeCategory, []output.Exclusion, error) {
	var operations []output.OrganizeOperation
	var exclusionsList []output.Exclusion

	categoryStats := make(map[string]*output.OrganizeCategory)

	destDir := opts.DestDir
	if destDir == "" {
		destDir = rootDir
	}

	progress.Start(-1, "Organizing files")

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

		// Never descend into the destination directory, otherwise organized
		// files would be picked up again on the next run
		if isDir && isSamePath(path, destDir) {
			return filepath.SkipDir
		}

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
			return nil
		}

		// Skip hidden files unless explicitly requested
		if !opts.IncludeHidden && strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not stat %s: %v\n", path, err)
			return nil
		}

		category := categorizeFile(path, info, opts.By)

		// Skip files that do not belong to the requested category
		if opts.Category != "" && !strings.EqualFold(category, opts.Category) {
			return nil
		}

		targetDir := filepath.Join(destDir, category)
		newPath := filepath.Join(targetDir, d.Name())

		// The file is already where it belongs
		if isSamePath(filepath.Dir(path), targetDir) {
			return nil
		}

		op := output.OrganizeOperation{
			OldPath:  relPath,
			NewPath:  filepath.Join(category, d.Name()),
			Category: category,
			Size:     info.Size(),
		}

		// Check if the target already exists
		if _, err := os.Stat(newPath); err == nil {
			if !opts.Overwrite {
				op.Error = "target file already exists"
			} else if !opts.DryRun {
				if err := os.Remove(newPath); err != nil {
					op.Error = fmt.Sprintf("cannot replace target: %v", err)
				}
			}
		} else if !os.IsNotExist(err) {
			op.Error = fmt.Sprintf("cannot check target: %v", err)
		}

		// Move the file if this is not a dry run and nothing went wrong
		if op.Error == "" && !opts.DryRun {
			if err := os.MkdirAll(targetDir, 0755); err != nil {
				op.Error = fmt.Sprintf("cannot create target directory: %v", err)
			} else if err := os.Rename(path, newPath); err != nil {
				op.Error = fmt.Sprintf("move failed: %v", err)
			}
		}

		operations = append(operations, op)

		stats, ok := categoryStats[category]
		if !ok {
			stats = &output.OrganizeCategory{Category: category}
			categoryStats[category] = stats
		}
		stats.FileCount++
		stats.TotalSize += info.Size()

		progress.Increment()
		return nil
	})

	if err != nil {
		return nil, nil, nil, err
	}

	// Build the category summary, sorted by name
	categories := make([]output.OrganizeCategory, 0, len(categoryStats))
	for _, stats := range categoryStats {
		categories = append(categories, *stats)
	}
	sortOrganizeCategories(categories)

	return operations, categories, exclusionsList, nil
}

// sortOrganizeCategories sorts categories by name
func sortOrganizeCategories(categories []output.OrganizeCategory) {
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Category < categories[j].Category
	})
}

// runOrganize executes the organize command
func runOrganize(cmd *cobra.Command, args []string) {
	rootDir := "."
	if len(args) > 0 {
		rootDir = args[0]
	}

	// Validate the grouping criteria
	switch organizeBy {
	case "type", "extension", "date", "size":
		// Valid
	default:
		fmt.Fprintf(os.Stderr, "Error: unsupported grouping '%s'. Supported: type, extension, date, size\n", organizeBy)
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

	destDir := organizeDest
	if destDir == "" {
		destDir = rootDir
	}

	isDryRun := !organizeForce

	// Parse exclusion patterns
	fileMatchers := exclusions.ParseExclusions(excludeFilePatterns, true)
	dirMatchers := exclusions.ParseExclusions(excludeDirPatterns, false)

	// Create progress indicator
	progress := output.NewProgressIndicator(cmd)
	defer progress.Finish()

	opts := organizeOptions{
		By:            organizeBy,
		DestDir:       destDir,
		DryRun:        isDryRun,
		Overwrite:     organizeOverwrite,
		IncludeHidden: organizeIncludeHidden,
		Category:      organizeCategory,
	}

	operations, categories, exclusionsList, err := organizeFiles(rootDir, opts, fileMatchers, dirMatchers, progress)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error organizing files: %v\n", err)
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
		{Name: "by", Value: organizeBy},
		{Name: "dest", Value: destDir},
		{Name: "dry-run", Value: fmt.Sprintf("%t", isDryRun)},
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
		SubCommand:  "organize",
		Flags:       flags,
		Version:     version,
		GeneratedAt: time.Now().Format(time.RFC3339),
	}

	result := &output.OrganizeResult{
		Operations: operations,
		Categories: categories,
		DryRun:     isDryRun,
		Exclusions: exclusionsList,
		Metadata:   metadata,
	}

	// Output the results
	if err := formatter.FormatOrganize(result, writer); err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
		os.Exit(1)
	}
}
