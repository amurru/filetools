package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"amurru/filetools/internal/output"
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

// analyzeDirectory traverses the directory and collects statistics
func analyzeDirectory(rootDir string) (*output.DirStatResult, error) {
	totalFiles := 0
	totalSize := int64(0)
	var largestFile *output.FileInfo

	fileTypes := make(map[string]*output.FileType)
	directories := make(map[string]*output.DirectoryInfo)

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip files/directories we can't access
			fmt.Fprintf(os.Stderr, "Warning: could not access %s: %v\n", path, err)
			return nil
		}

		// Skip the root directory itself
		if path == rootDir {
			return nil
		}

		// Get relative path for directory tracking
		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}

		if info.IsDir() {
			// Initialize directory stats
			directories[relPath] = &output.DirectoryInfo{
				Path:      relPath,
				FileCount: 0,
				TotalSize: 0,
			}
		} else {
			// File statistics
			totalFiles++
			totalSize += info.Size()

			// Track largest file
			if largestFile == nil || info.Size() > largestFile.Size {
				largestFile = &output.FileInfo{
					Name: filepath.Base(path),
					Size: info.Size(),
					Path: relPath,
				}
			}

			// File type statistics
			ext := strings.ToLower(filepath.Ext(path))
			if ext == "" {
				ext = "(no extension)"
			}

			if _, exists := fileTypes[ext]; !exists {
				fileTypes[ext] = &output.FileType{
					Extension: ext,
					Count:     0,
					TotalSize: 0,
				}
			}
			fileTypes[ext].Count++
			fileTypes[ext].TotalSize += info.Size()

			// Add to directory statistics
			dir := filepath.Dir(relPath)
			if dirStats, exists := directories[dir]; exists {
				dirStats.FileCount++
				dirStats.TotalSize += info.Size()
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Convert maps to slices and calculate percentages
	var fileTypesSlice []output.FileType
	for _, ft := range fileTypes {
		ft.Percentage = float64(ft.TotalSize) / float64(totalSize) * 100
		fileTypesSlice = append(fileTypesSlice, *ft)
	}

	var directoriesSlice []output.DirectoryInfo
	for _, dir := range directories {
		if dir.FileCount > 0 { // Only include directories with files
			dir.Percentage = float64(dir.TotalSize) / float64(totalSize) * 100
			directoriesSlice = append(directoriesSlice, *dir)
		}
	}

	// Sort file types by total size (descending)
	sort.Slice(fileTypesSlice, func(i, j int) bool {
		return fileTypesSlice[i].TotalSize > fileTypesSlice[j].TotalSize
	})

	// Sort directories by total size (descending)
	sort.Slice(directoriesSlice, func(i, j int) bool {
		return directoriesSlice[i].TotalSize > directoriesSlice[j].TotalSize
	})

	result := &output.DirStatResult{
		TotalFiles:  totalFiles,
		TotalSize:   totalSize,
		LargestFile: largestFile,
		FileTypes:   fileTypesSlice,
		Directories: directoriesSlice,
	}

	return result, nil
}

// runDirstat executes the dirstat command
func runDirstat(cmd *cobra.Command, args []string) {
	rootDir := "."
	if len(args) > 0 {
		rootDir = args[0]
	}

	// Validate directory exists
	if info, err := os.Stat(rootDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	} else if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: %s is not a directory\n", rootDir)
		os.Exit(1)
	}

	// Analyze directory
	result, err := analyzeDirectory(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error analyzing directory: %v\n", err)
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
		{Name: "output", Value: string(format)},
	}

	// Add file flag if specified
	if outputFile != "" {
		flags = append(flags, output.Flag{Name: "file", Value: outputFile})
	}

	metadata := &output.Metadata{
		ToolName:    "filetools",
		SubCommand:  "dirstat",
		Flags:       flags,
		Version:     version,
		GeneratedAt: time.Now().Format(time.RFC3339),
	}

	// Set metadata in result
	result.Metadata = metadata

	// Output the results
	if err := formatter.FormatDirStat(result, writer); err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
		os.Exit(1)
	}
}
