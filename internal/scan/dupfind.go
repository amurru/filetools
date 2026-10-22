package scan

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
	"sync"
	"sync/atomic"

	"github.com/amurru/filetools/internal/exclusions"
	"github.com/amurru/filetools/internal/progress"
	"github.com/amurru/filetools/internal/report"
)

// partialHashSize is the number of bytes read from the beginning of a file
// when computing its quick fingerprint
const partialHashSize = 64 * 1024

// DuplicateOptions configures a duplicate file search.
type DuplicateOptions struct {
	// Dir is the directory tree to scan.
	Dir string
	// Algorithm is the hash algorithm to use: md5, sha1 or sha256.
	Algorithm string
	// MinSize and MaxSize bound the size of the considered files in bytes.
	// A value of zero means no bound.
	MinSize int64
	MaxSize int64
	// ExcludeFile and ExcludeDir are comma separated exclusion patterns.
	ExcludeFile string
	ExcludeDir  string
}

// FindDuplicates searches Dir for files with identical content.
//
// Only files that share their size can be duplicates, so the engine first
// groups the tree by size, then fingerprints the candidates on their first
// bytes and finally reads the few files that can still be identical in
// full.
func FindDuplicates(opts DuplicateOptions, p progress.Indicator) (*report.DuplicateResult, error) {
	switch opts.Algorithm {
	case "md5", "sha1", "sha256":
		// Supported
	default:
		return nil, fmt.Errorf("unsupported hash algorithm: %s", opts.Algorithm)
	}

	if opts.MinSize > 0 && opts.MaxSize > 0 && opts.MinSize > opts.MaxSize {
		return nil, fmt.Errorf("min size must not be larger than max size")
	}

	if err := validateDir(opts.Dir); err != nil {
		return nil, err
	}

	hashMap, exclusionsList, err := findDuplicates(opts, p)
	if err != nil {
		return nil, err
	}

	result := &report.DuplicateResult{
		Groups:     []report.DuplicateGroup{},
		Exclusions: exclusionsList,
	}

	for hash, files := range hashMap {
		if len(files) < 2 {
			continue
		}
		result.Found = true

		// Sort files alphabetically
		sort.Strings(files)

		// Get file info for size
		size := int64(-1)
		if info, err := os.Stat(files[0]); err == nil {
			size = info.Size()
		}

		result.Groups = append(result.Groups, report.DuplicateGroup{
			Hash:     hash,
			HashType: opts.Algorithm,
			Size:     size,
			Files:    files,
		})
	}

	// Report the largest groups first and break ties by hash so that the
	// output is deterministic.
	sort.Slice(result.Groups, func(i, j int) bool {
		if result.Groups[i].Size != result.Groups[j].Size {
			return result.Groups[i].Size > result.Groups[j].Size
		}
		return result.Groups[i].Hash < result.Groups[j].Hash
	})

	return result, nil
}

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
func hashFilesConcurrently(files []string, algorithm string, p progress.Indicator, hash hashFunc) []hashResult {
	if len(files) == 0 {
		return nil
	}

	numWorkers := runtime.NumCPU()
	if len(files) < numWorkers {
		numWorkers = len(files) // Don't create more workers than files
	}

	jobs := make(chan hashJob, len(files))
	results := make(chan hashResult, len(files))

	// stopped is local to this call. Workers of an abandoned call stop
	// even after a new Run resets the job's shared cancellation flag.
	var stopped atomic.Bool

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				progress.WaitIfPaused(p)
				if stopped.Load() || progress.IsCancelled(p) {
					continue
				}
				sum, err := hash(job.path, algorithm)
				results <- hashResult{job.path, sum, err}
			}
		}()
	}

	// Send jobs
	for _, file := range files {
		progress.WaitIfPaused(p)
		if progress.IsCancelled(p) {
			stopped.Store(true)
			break
		}
		jobs <- hashJob{path: file}
	}
	close(jobs)

	// Wait for completion in background
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results. The channels are buffered for all files, so
	// returning early on cancellation never blocks the workers.
	var output []hashResult
	for result := range results {
		progress.WaitIfPaused(p)
		output = append(output, result)
		p.Increment()
		if progress.IsCancelled(p) {
			stopped.Store(true)
			break
		}
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
func findDuplicates(opts DuplicateOptions, p progress.Indicator) (map[string][]string, []report.Exclusion, error) {
	hashMap := make(map[string][]string)
	var exclusionsList []report.Exclusion

	m := newMatchers(opts.ExcludeFile, opts.ExcludeDir)

	// Single walk: collect files grouped by size
	sizeGroups := make(map[int64][]string)

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

		// Get relative path for exclusion checking
		relPath, err := filepath.Rel(opts.Dir, path)
		if err != nil {
			return err
		}

		isDir := d.IsDir()

		// Check for exclusions
		if exclusion := exclusions.CheckExclusions(relPath, isDir, m.files, m.dirs); exclusion != nil {
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
		if opts.MinSize > 0 && size < opts.MinSize {
			return nil
		}
		if opts.MaxSize > 0 && size > opts.MaxSize {
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

	p.Start(int64(len(candidates)), "Fingerprinting candidate files")

	// Fingerprint the candidates on their first bytes. Files that differ
	// here can never be identical, so they are never read completely.
	candidatePaths := make([]string, len(candidates))
	for i, c := range candidates {
		candidatePaths[i] = c.path
	}

	partialResults := hashFilesConcurrently(candidatePaths, opts.Algorithm, p, calculatePartialHash)
	if progress.IsCancelled(p) {
		return nil, exclusionsList, progress.ErrCancelled
	}

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
		progress.WaitIfPaused(p)
		if progress.IsCancelled(p) {
			return nil, exclusionsList, progress.ErrCancelled
		}
		p.Start(int64(len(toVerify)), "Verifying candidates")

		fullResults := hashFilesConcurrently(toVerify, opts.Algorithm, p, calculateHash)
		if progress.IsCancelled(p) {
			return nil, exclusionsList, progress.ErrCancelled
		}
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
