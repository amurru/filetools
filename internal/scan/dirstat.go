package scan

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/amurru/filetools/internal/exclusions"
	"github.com/amurru/filetools/internal/progress"
	"github.com/amurru/filetools/internal/report"
)

// DirStatOptions configures a directory statistics analysis.
type DirStatOptions struct {
	// Dir is the directory tree to analyze.
	Dir string
	// ExcludeFile and ExcludeDir are comma separated exclusion patterns.
	ExcludeFile string
	ExcludeDir  string
}

// AnalyzeDirStat collects file counts, sizes and per type statistics for a
// directory tree.
func AnalyzeDirStat(opts DirStatOptions, p progress.Indicator) (*report.DirStatResult, error) {
	if err := validateDir(opts.Dir); err != nil {
		return nil, err
	}

	totalFiles := 0
	totalSize := int64(0)
	var largestFile *report.FileInfo

	fileTypes := make(map[string]*report.FileType)
	directories := make(map[string]*report.DirectoryInfo)
	var exclusionsList []report.Exclusion

	m := newMatchers(opts.ExcludeFile, opts.ExcludeDir)

	// Single-pass walk: collect stats dynamically
	// Start with estimated count for progress (will increment as we go)
	p.Start(-1, "Analyzing files")

	err := filepath.WalkDir(opts.Dir, func(path string, d fs.DirEntry, err error) error {
		progress.WaitIfPaused(p)
		if progress.IsCancelled(p) {
			return progress.ErrCancelled
		}
		if err != nil {
			// Skip files/directories we can't access
			fmt.Fprintf(os.Stderr, "Warning: could not access %s: %v\n", path, err)
			return nil
		}

		// Skip the root directory itself
		if path == opts.Dir {
			return nil
		}

		isDir := d.IsDir()

		// Get relative path for directory tracking and exclusion checking
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
			// Initialize directory stats
			directories[relPath] = &report.DirectoryInfo{
				Path:      relPath,
				FileCount: 0,
				TotalSize: 0,
			}
		} else {
			// File statistics
			info, err := d.Info()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not stat %s: %v\n", path, err)
				return nil
			}
			size := info.Size()

			totalFiles++
			totalSize += size
			p.Increment()

			// Track largest file
			if largestFile == nil || size > largestFile.Size {
				largestFile = &report.FileInfo{
					Name: d.Name(),
					Size: size,
					Path: relPath,
				}
			}

			// File type statistics
			ext := strings.ToLower(filepath.Ext(path))
			if ext == "" {
				ext = "(no extension)"
			}

			if _, exists := fileTypes[ext]; !exists {
				fileTypes[ext] = &report.FileType{
					Extension: ext,
					Count:     0,
					TotalSize: 0,
				}
			}
			fileTypes[ext].Count++
			fileTypes[ext].TotalSize += size

			// Add to directory statistics
			dir := filepath.Dir(relPath)
			if dirStats, exists := directories[dir]; exists {
				dirStats.FileCount++
				dirStats.TotalSize += size
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Convert maps to slices and calculate percentages
	fileTypesSlice := make([]report.FileType, 0, len(fileTypes))
	for _, ft := range fileTypes {
		if totalSize > 0 {
			ft.Percentage = float64(ft.TotalSize) / float64(totalSize) * 100
		}
		fileTypesSlice = append(fileTypesSlice, *ft)
	}

	directoriesSlice := make([]report.DirectoryInfo, 0, len(directories))
	for _, dir := range directories {
		if dir.FileCount > 0 { // Only include directories with files
			if totalSize > 0 {
				dir.Percentage = float64(dir.TotalSize) / float64(totalSize) * 100
			}
			directoriesSlice = append(directoriesSlice, *dir)
		}
	}

	// Sort file types by total size (descending)
	sort.Slice(fileTypesSlice, func(i, j int) bool {
		if fileTypesSlice[i].TotalSize != fileTypesSlice[j].TotalSize {
			return fileTypesSlice[i].TotalSize > fileTypesSlice[j].TotalSize
		}
		return fileTypesSlice[i].Extension < fileTypesSlice[j].Extension
	})

	// Sort directories by total size (descending)
	sort.Slice(directoriesSlice, func(i, j int) bool {
		if directoriesSlice[i].TotalSize != directoriesSlice[j].TotalSize {
			return directoriesSlice[i].TotalSize > directoriesSlice[j].TotalSize
		}
		return directoriesSlice[i].Path < directoriesSlice[j].Path
	})

	result := &report.DirStatResult{
		TotalFiles:  totalFiles,
		TotalSize:   totalSize,
		LargestFile: largestFile,
		FileTypes:   fileTypesSlice,
		Directories: directoriesSlice,
		Exclusions:  exclusionsList,
	}

	return result, nil
}
