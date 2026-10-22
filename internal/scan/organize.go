package scan

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"amurru/filetools/internal/exclusions"
	"amurru/filetools/internal/progress"
	"amurru/filetools/internal/report"
)

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

// OrganizeOptions configures an organize run.
type OrganizeOptions struct {
	// Source is the directory tree to scan.
	Source string
	// Dest is the directory the files are moved into. An empty value means
	// the source directory.
	Dest string
	// By selects the grouping criteria: type, extension, date or size.
	By string
	// DryRun reports the planned moves without touching the filesystem.
	DryRun bool
	// Overwrite allows replacing a file that already exists at the target.
	Overwrite bool
	// IncludeHidden also organizes files whose name starts with a dot.
	IncludeHidden bool
	// Category limits the run to the files of a single category.
	Category string
	// ExcludeFile and ExcludeDir are comma separated exclusion patterns.
	ExcludeFile string
	ExcludeDir  string
}

// Organize moves the files of a directory tree into per category
// subdirectories.
//
// The run is a dry run unless DryRun is false, which mirrors the safety
// default of the command line interface.
func Organize(opts OrganizeOptions, p progress.Indicator) (*report.OrganizeResult, error) {
	switch opts.By {
	case "type", "extension", "date", "size":
		// Supported
	default:
		return nil, fmt.Errorf("unsupported grouping: %s", opts.By)
	}

	if err := validateDir(opts.Source); err != nil {
		return nil, err
	}

	destDir := opts.Dest
	if destDir == "" {
		destDir = opts.Source
	}

	m := newMatchers(opts.ExcludeFile, opts.ExcludeDir)

	operations, categories, exclusionsList, err := organizeFiles(opts, destDir, m, p)
	if err != nil {
		return nil, err
	}

	return &report.OrganizeResult{
		Operations: operations,
		Categories: categories,
		DryRun:     opts.DryRun,
		Exclusions: exclusionsList,
	}, nil
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
		return SizeBucketLabel(info.Size())

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
func organizeFiles(opts OrganizeOptions, destDir string, m matchers, p progress.Indicator) ([]report.OrganizeOperation, []report.OrganizeCategory, []report.Exclusion, error) {
	var operations []report.OrganizeOperation
	var exclusionsList []report.Exclusion

	categoryStats := make(map[string]*report.OrganizeCategory)

	// plannedTargets holds the targets that another file of this run already
	// moves to. Two files of different directories can share a name, and
	// they would end up at the same target, where the second one would
	// destroy the first.
	plannedTargets := make(map[string]string)

	p.Start(-1, "Organizing files")

	err := filepath.WalkDir(opts.Source, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not access %s: %v\n", path, err)
			return nil
		}

		// Skip the root directory itself
		if path == opts.Source {
			return nil
		}

		isDir := d.IsDir()

		// Never descend into the destination directory, otherwise organized
		// files would be picked up again on the next run
		if isDir && isSamePath(path, destDir) {
			return filepath.SkipDir
		}

		// Get relative path for exclusion checking
		relPath, err := filepath.Rel(opts.Source, path)
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

		op := report.OrganizeOperation{
			OldPath:  relPath,
			NewPath:  filepath.Join(category, d.Name()),
			Category: category,
			Size:     info.Size(),
		}

		// Check if the target is already taken by another file of this run
		// or if it already exists on disk.
		targetKey := filepath.Clean(newPath)
		if previous, taken := plannedTargets[targetKey]; taken {
			op.Error = fmt.Sprintf("target already used by %s", previous)
		} else if _, err := os.Stat(newPath); err == nil {
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

		// The target is taken, whether the file was moved or only planned.
		if op.Error == "" {
			plannedTargets[targetKey] = relPath
		}

		operations = append(operations, op)

		stats, ok := categoryStats[category]
		if !ok {
			stats = &report.OrganizeCategory{Category: category}
			categoryStats[category] = stats
		}
		stats.FileCount++
		stats.TotalSize += info.Size()

		p.Increment()
		return nil
	})

	if err != nil {
		return nil, nil, nil, err
	}

	// Build the category summary, sorted by name
	categories := make([]report.OrganizeCategory, 0, len(categoryStats))
	for _, stats := range categoryStats {
		categories = append(categories, *stats)
	}
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Category < categories[j].Category
	})

	return operations, categories, exclusionsList, nil
}
