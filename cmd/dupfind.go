package cmd

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
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

// partialHashSize is the number of bytes read from the beginning of a file
// when computing its quick fingerprint
const partialHashSize = 64 * 1024

// newHasher creates a hash for the given algorithm
func newHasher(algorithm string) (hash.Hash, error) {
	switch algorithm {
	case "md5":
		return md5.New(), nil
	case "sha1":
		return sha1.New(), nil
	case "sha256":
		return sha256.New(), nil
	default:
		return nil, fmt.Errorf("unsupported hash algorithm: %s", algorithm)
	}
}

// calculateHash computes the hash of a file using the specified algorithm
func calculateHash(filePath, algorithm string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher, err := newHasher(algorithm)
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// calculatePartialHash computes the hash of the first bytes of a file.
// Files smaller than partialHashSize are hashed completely, so their
// partial hash is already their full hash.
func calculatePartialHash(filePath, algorithm string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher, err := newHasher(algorithm)
	if err != nil {
		return "", err
	}

	// io.CopyN reports io.EOF when the file is shorter than the limit
	if _, err := io.CopyN(hasher, file, partialHashSize); err != nil && err != io.EOF {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// hashFunc computes a hash for a file
type hashFunc func(filePath, algorithm string) (string, error)

// hashFilesConcurrently hashes multiple files concurrently using a worker pool
func hashFilesConcurrently(files []string, algorithm string, progress output.ProgressIndicator, hash hashFunc) []hashResult {
	if len(files) == 0 {
		return nil
	}

	numWorkers := runtime.NumCPU()
	if len(files) < numWorkers {
		numWorkers = len(files) // Don't create more workers than files
	}

	jobs := make(chan hashJob, len(files))
	results := make(chan hashResult, len(files))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				sum, err := hash(job.path, algorithm)
				results <- hashResult{job.path, sum, err}
			}
		}()
	}

	// Send jobs
	for _, file := range files {
		jobs <- hashJob{path: file}
	}
	close(jobs)

	// Wait for completion in background
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var output []hashResult
	for result := range results {
		output = append(output, result)
		progress.Increment()
	}

	return output
}

// hashJob represents a file hashing task
type hashJob struct {
	path string
}

// hashResult represents the result of a hashing operation
type hashResult struct {
	path  string
	hash  string
	error error
}

// candidate is a file that shares its size with at least one other file
type candidate struct {
	path string
	size int64
}

// sizeKey identifies a group of files sharing both a size and a partial hash
type sizeKey struct {
	size int64
	hash string
}

// findDuplicates traverses the directory and finds duplicate files
func findDuplicates(rootDir, algorithm string, minSize, maxSize int64, fileMatchers, dirMatchers []exclusions.ExclusionMatcher, progress output.ProgressIndicator) (map[string][]string, []output.Exclusion, error) {
	hashMap := make(map[string][]string)
	var exclusionsList []output.Exclusion

	// Single walk: collect files grouped by size
	sizeGroups := make(map[int64][]string)

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Get relative path for exclusion checking
		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}

		isDir := d.IsDir()

		// Check for exclusions
		if exclusion := exclusions.CheckExclusions(relPath, isDir, fileMatchers, dirMatchers); exclusion != nil {
			exclusionsList = append(exclusionsList, *exclusion)
			if isDir {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip directories
		if isDir {
			return nil
		}

		// Group files by size
		info, err := d.Info()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not stat %s: %v\n", path, err)
			return nil
		}
		size := info.Size()

		// Apply the size range filter
		if minSize > 0 && size < minSize {
			return nil
		}
		if maxSize > 0 && size > maxSize {
			return nil
		}

		sizeGroups[size] = append(sizeGroups[size], path)
		return nil
	})

	if err != nil {
		return nil, exclusionsList, err
	}

	// Only files sharing a size can be duplicates, so only those are hashed
	var candidates []candidate
	for size, files := range sizeGroups {
		if len(files) <= 1 {
			// No duplicates possible for this size
			continue
		}
		for _, file := range files {
			candidates = append(candidates, candidate{path: file, size: size})
		}
	}

	progress.Start(int64(len(candidates)), "Fingerprinting candidate files")

	// Fingerprint the candidates on their first bytes. Files that differ
	// here can never be identical, so they are never read completely.
	candidatePaths := make([]string, len(candidates))
	for i, c := range candidates {
		candidatePaths[i] = c.path
	}

	partialResults := hashFilesConcurrently(candidatePaths, algorithm, progress, calculatePartialHash)

	// Group the candidates by size and partial hash
	partialGroups := make(map[sizeKey][]string)
	sizes := make(map[string]int64)
	for _, c := range candidates {
		sizes[c.path] = c.size
	}

	for _, result := range partialResults {
		if result.error != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not hash file %s: %v\n", result.path, result.error)
			continue
		}

		key := sizeKey{size: sizes[result.path], hash: result.hash}
		partialGroups[key] = append(partialGroups[key], result.path)

		// A file smaller than the fingerprint limit was hashed completely,
		// so its fingerprint is already its final hash
		if sizes[result.path] <= partialHashSize {
			hashMap[result.hash] = append(hashMap[result.hash], result.path)
		}
	}

	// Only the larger files that share a fingerprint have to be read in full
	var toVerify []string
	for key, files := range partialGroups {
		if len(files) <= 1 || key.size <= partialHashSize {
			continue
		}
		toVerify = append(toVerify, files...)
	}

	if len(toVerify) > 0 {
		progress.Start(int64(len(toVerify)), "Verifying candidates")

		fullResults := hashFilesConcurrently(toVerify, algorithm, progress, calculateHash)
		for _, result := range fullResults {
			if result.error != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not hash file %s: %v\n", result.path, result.error)
				continue
			}
			hashMap[result.hash] = append(hashMap[result.hash], result.path)
		}
	}

	return hashMap, exclusionsList, nil
}

// parseSize parses a human readable size such as "1024", "512K" or "10MB"
// into a number of bytes
func parseSize(value string) (int64, error) {
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

	// Parse the size range filters
	minSizeBytes, err := parseSize(minSize)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: --min-size: %v\n", err)
		os.Exit(1)
	}
	maxSizeBytes, err := parseSize(maxSize)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: --max-size: %v\n", err)
		os.Exit(1)
	}
	if minSizeBytes > 0 && maxSizeBytes > 0 && minSizeBytes > maxSizeBytes {
		fmt.Fprintf(os.Stderr, "Error: --min-size must not be larger than --max-size\n")
		os.Exit(1)
	}

	// Parse exclusion patterns
	fileMatchers := exclusions.ParseExclusions(excludeFilePatterns, true)
	dirMatchers := exclusions.ParseExclusions(excludeDirPatterns, false)

	// Create progress indicator
	progress := output.NewProgressIndicator(cmd)
	defer progress.Finish()

	hashMap, exclusionsList, err := findDuplicates(rootDir, hashAlgorithm, minSizeBytes, maxSizeBytes, fileMatchers, dirMatchers, progress)
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

	// Add size range flags if specified
	if minSize != "" {
		flags = append(flags, output.Flag{Name: "min-size", Value: minSize})
	}
	if maxSize != "" {
		flags = append(flags, output.Flag{Name: "max-size", Value: maxSize})
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
