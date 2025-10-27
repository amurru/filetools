package cmd

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"amurru/filetools/internal/exclusions"
	"amurru/filetools/internal/output"
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

If the directory is not specified, the current directory will be used.
`,
	Run: runDupfind,
}

var hashAlgorithm string

func init() {
	rootCmd.AddCommand(dupfindCmd)

	// Add hash algorithm flag
	dupfindCmd.Flags().StringVarP(&hashAlgorithm, "hash", "H", "md5", "Hash algorithm to use (md5, sha1, sha256)")
}

// calculateHash computes the hash of a file using the specified algorithm
func calculateHash(filePath, algorithm string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var hasher io.Writer
	switch algorithm {
	case "md5":
		hasher = md5.New()
	case "sha1":
		hasher = sha1.New()
	case "sha256":
		hasher = sha256.New()
	default:
		return "", fmt.Errorf("unsupported hash algorithm: %s", algorithm)
	}

	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	hash := hasher.(interface{ Sum([]byte) []byte }).Sum(nil)
	return hex.EncodeToString(hash), nil
}

// findDuplicates traverses the directory and finds duplicate files
func findDuplicates(rootDir, algorithm string, fileMatchers, dirMatchers []exclusions.ExclusionMatcher) (map[string][]string, []output.Exclusion, error) {
	hashMap := make(map[string][]string)
	var exclusionsList []output.Exclusion

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path for exclusion checking
		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}

		// Check for exclusions
		if exclusion := exclusions.CheckExclusions(relPath, info.IsDir(), fileMatchers, dirMatchers); exclusion != nil {
			exclusionsList = append(exclusionsList, *exclusion)
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		hash, err := calculateHash(path, algorithm)
		if err != nil {
			// Skip files that can't be hashed (permission issues, etc.)
			fmt.Fprintf(os.Stderr, "Warning: could not hash file %s: %v\n", path, err)
			return nil
		}

		hashMap[hash] = append(hashMap[hash], path)
		return nil
	})

	return hashMap, exclusionsList, err
}

// runDupfind executes the dupfind command
func runDupfind(cmd *cobra.Command, args []string) {
	rootDir := "."
	if len(args) > 0 {
		rootDir = args[0]
	}

	// Validate hash algorithm
	switch hashAlgorithm {
	case "md5", "sha1", "sha256":
		// Valid
	default:
		fmt.Fprintf(os.Stderr, "Error: unsupported hash algorithm '%s'. Supported: md5, sha1, sha256\n", hashAlgorithm)
		os.Exit(1)
	}

	// Parse exclusion patterns
	fileMatchers := exclusions.ParseExclusions(excludeFilePatterns, true)
	dirMatchers := exclusions.ParseExclusions(excludeDirPatterns, false)

	hashMap, exclusionsList, err := findDuplicates(rootDir, hashAlgorithm, fileMatchers, dirMatchers)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error traversing directory: %v\n", err)
		os.Exit(1)
	}

	// Convert hashMap to structured result
	result := &output.DuplicateResult{
		Groups:     []output.DuplicateGroup{},
		Found:      false,
		Exclusions: exclusionsList,
	}

	for hash, files := range hashMap {
		if len(files) > 1 {
			result.Found = true

			// Sort files alphabetically
			sort.Strings(files)

			// Get file info for size
			size := int64(-1)
			if info, err := os.Stat(files[0]); err == nil {
				size = info.Size()
			}

			group := output.DuplicateGroup{
				Hash:     hash,
				HashType: hashAlgorithm,
				Size:     size,
				Files:    files,
			}

			result.Groups = append(result.Groups, group)
		}
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
		{Name: "hash", Value: hashAlgorithm},
		{Name: "output", Value: string(format)},
	}

	// Add file flag if specified
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
		SubCommand:  "dupfind",
		Flags:       flags,
		Version:     version,
		GeneratedAt: time.Now().Format(time.RFC3339),
	}

	// Set metadata in result
	result.Metadata = metadata

	// Output the results
	if err := formatter.FormatDuplicates(result, writer); err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
		os.Exit(1)
	}
}
