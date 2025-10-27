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
func findDuplicates(rootDir, algorithm string) (map[string][]string, error) {
	hashMap := make(map[string][]string)

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
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

	return hashMap, err
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

	hashMap, err := findDuplicates(rootDir, hashAlgorithm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error traversing directory: %v\n", err)
		os.Exit(1)
	}

	// Find and display duplicates
	foundDuplicates := false
	for hash, files := range hashMap {
		if len(files) > 1 {
			if !foundDuplicates {
				fmt.Println("Duplicate files found:")
				foundDuplicates = true
			}

			// Sort files alphabetically
			sort.Strings(files)

			// Get file info for size
			info, err := os.Stat(files[0])
			sizeStr := "unknown size"
			if err == nil {
				sizeStr = fmt.Sprintf("%d bytes", info.Size())
			}

			// Display the first file as the "original"
			fmt.Printf("- %s (size: %s, hash: %s)\n", filepath.Base(files[0]), sizeStr, hash[:8]+"...")
			for _, file := range files {
				fmt.Printf("  - %s\n", file)
			}
			fmt.Println()
		}
	}

	if !foundDuplicates {
		fmt.Println("No duplicate files found.")
	}
}
