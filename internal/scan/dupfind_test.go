package scan

import (
	"os"
	"path/filepath"
	"testing"

	"amurru/filetools/internal/progress"
)

func TestCalculateHash(t *testing.T) {
	// Create a temporary file with known content
	tmpFile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	content := "Hello, World!"
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	tests := []struct {
		algorithm string
		expected  string
	}{
		{"md5", "65a8e27d8879283831b664bd8b7f0ad4"},
		{"sha1", "0a0a9f2a6772942557ab5355d76af442f8f65e01"},
		{"sha256", "dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f"},
	}

	for _, test := range tests {
		hash, err := calculateHash(tmpFile.Name(), test.algorithm)
		if err != nil {
			t.Errorf("calculateHash(%s) failed: %v", test.algorithm, err)
		}
		if hash != test.expected {
			t.Errorf("calculateHash(%s) = %s, want %s", test.algorithm, hash, test.expected)
		}
	}
}

func TestCalculateHashUnsupportedAlgorithm(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	_, err = calculateHash(tmpFile.Name(), "unsupported")
	if err == nil {
		t.Error("Expected error for unsupported algorithm")
	}
}

func TestFindDuplicatesWithExclusions(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "duptest_excl")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create duplicate files
	content := "duplicate content"
	for _, name := range []string{"file1.txt", "file2.txt"} {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", name, err)
		}
	}

	result, err := FindDuplicates(DuplicateOptions{
		Dir:         tmpDir,
		Algorithm:   "md5",
		ExcludeFile: "file2.txt",
	}, progress.Nop{})
	if err != nil {
		t.Fatalf("FindDuplicates failed: %v", err)
	}

	if len(result.Exclusions) != 1 {
		t.Errorf("Expected 1 exclusion, got %d", len(result.Exclusions))
	}

	if result.Found {
		t.Errorf("Expected no duplicate groups after exclusion, got %v", result.Groups)
	}
}

// TestFindDuplicatesDoesNotExcludeTheRoot verifies that a directory pattern
// that matches the root of the scan does not prune the whole tree.
func TestFindDuplicatesDoesNotExcludeTheRoot(t *testing.T) {
	tmpDir := t.TempDir()

	content := "duplicate content"
	for _, name := range []string{"file1.txt", "file2.txt"} {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", name, err)
		}
	}

	// The pattern matches the root directory, which must still be scanned.
	result, err := FindDuplicates(DuplicateOptions{
		Dir:        tmpDir,
		Algorithm:  "md5",
		ExcludeDir: ".*",
	}, progress.Nop{})
	if err != nil {
		t.Fatalf("FindDuplicates failed: %v", err)
	}

	if !result.Found {
		t.Error("Expected the duplicates to be found")
	}
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		wantErr  bool
	}{
		{"", 0, false},
		{"1024", 1024, false},
		{"1K", 1024, false},
		{"1KB", 1024, false},
		{"1kib", 1024, false},
		{"5M", 5 * 1024 * 1024, false},
		{"10MB", 10 * 1024 * 1024, false},
		{"1G", 1024 * 1024 * 1024, false},
		{"2.5M", int64(2.5 * 1024 * 1024), false},
		{"abc", 0, true},
		{"-1", 0, true},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			size, err := ParseSize(test.input)
			if (err != nil) != test.wantErr {
				t.Fatalf("ParseSize(%q) error = %v, wantErr %v", test.input, err, test.wantErr)
			}
			if err == nil && size != test.expected {
				t.Errorf("ParseSize(%q) = %d, want %d", test.input, size, test.expected)
			}
		})
	}
}

func TestCalculatePartialHashMatchesFullHashForSmallFiles(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	content := make([]byte, 1024)
	for i := range content {
		content[i] = byte(i % 256)
	}
	if _, err := tmpFile.Write(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	full, err := calculateHash(tmpFile.Name(), "md5")
	if err != nil {
		t.Fatalf("calculateHash failed: %v", err)
	}

	partial, err := calculatePartialHash(tmpFile.Name(), "md5")
	if err != nil {
		t.Fatalf("calculatePartialHash failed: %v", err)
	}

	if partial != full {
		t.Errorf("Expected partial hash %s to equal full hash %s for a small file", partial, full)
	}
}

func TestCalculatePartialHashDiffersForLargeFiles(t *testing.T) {
	dir := t.TempDir()

	// Two files of the same size sharing their first bytes but differing later
	content := make([]byte, 2*partialHashSize)
	for i := range content {
		content[i] = byte(i % 256)
	}

	first := filepath.Join(dir, "first.bin")
	second := filepath.Join(dir, "second.bin")

	if err := os.WriteFile(first, content, 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	altered := make([]byte, len(content))
	copy(altered, content)
	altered[len(altered)-1] ^= 0xFF
	if err := os.WriteFile(second, altered, 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	firstPartial, err := calculatePartialHash(first, "md5")
	if err != nil {
		t.Fatalf("calculatePartialHash failed: %v", err)
	}
	secondPartial, err := calculatePartialHash(second, "md5")
	if err != nil {
		t.Fatalf("calculatePartialHash failed: %v", err)
	}

	if firstPartial != secondPartial {
		t.Error("Expected the fingerprints of the shared prefix to match")
	}

	firstFull, err := calculateHash(first, "md5")
	if err != nil {
		t.Fatalf("calculateHash failed: %v", err)
	}
	secondFull, err := calculateHash(second, "md5")
	if err != nil {
		t.Fatalf("calculateHash failed: %v", err)
	}

	if firstFull == secondFull {
		t.Error("Expected the full hashes to differ")
	}
}

func TestFindDuplicatesLargeFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "duptest_large")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Two identical files larger than the fingerprint limit
	content := make([]byte, partialHashSize+1024)
	for i := range content {
		content[i] = byte(i % 256)
	}

	for _, name := range []string{"large1.bin", "large2.bin"} {
		if err := os.WriteFile(filepath.Join(tmpDir, name), content, 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", name, err)
		}
	}

	result, err := FindDuplicates(DuplicateOptions{Dir: tmpDir, Algorithm: "md5"}, progress.Nop{})
	if err != nil {
		t.Fatalf("FindDuplicates failed: %v", err)
	}

	if len(result.Groups) != 1 {
		t.Errorf("Expected 1 duplicate group, got %d", len(result.Groups))
	}
	if !result.Found {
		t.Error("Expected the result to report that duplicates were found")
	}
}

func TestFindDuplicatesSizeFilter(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "duptest_size")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Small duplicates and large duplicates
	small := []byte("small duplicate")
	large := make([]byte, 4096)

	for _, name := range []string{"small1.bin", "small2.bin"} {
		if err := os.WriteFile(filepath.Join(tmpDir, name), small, 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", name, err)
		}
	}
	for _, name := range []string{"large1.bin", "large2.bin"} {
		if err := os.WriteFile(filepath.Join(tmpDir, name), large, 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", name, err)
		}
	}

	// Only files of at least 1KB are considered
	result, err := FindDuplicates(DuplicateOptions{
		Dir:       tmpDir,
		Algorithm: "md5",
		MinSize:   1024,
	}, progress.Nop{})
	if err != nil {
		t.Fatalf("FindDuplicates failed: %v", err)
	}

	if len(result.Groups) != 1 {
		t.Fatalf("Expected 1 duplicate group, got %d", len(result.Groups))
	}
	for _, file := range result.Groups[0].Files {
		if filepath.Base(file) == "small1.bin" {
			t.Error("Expected the small files to be filtered out")
		}
	}
}

func TestFindDuplicates(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "duptest")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create some files with duplicate content
	content1 := "duplicate content"
	content2 := "unique content"

	// Create duplicate files
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	if err := os.WriteFile(file1, []byte(content1), 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	if err := os.WriteFile(file2, []byte(content1), 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	// Create unique file
	file3 := filepath.Join(tmpDir, "file3.txt")
	if err := os.WriteFile(file3, []byte(content2), 0644); err != nil {
		t.Fatalf("Failed to create file3: %v", err)
	}

	result, err := FindDuplicates(DuplicateOptions{Dir: tmpDir, Algorithm: "md5"}, progress.Nop{})
	if err != nil {
		t.Fatalf("FindDuplicates failed: %v", err)
	}

	if len(result.Groups) != 1 {
		t.Fatalf("Expected 1 duplicate group, got %d", len(result.Groups))
	}
	if len(result.Groups[0].Files) != 2 {
		t.Errorf("Expected 2 duplicate files, got %d", len(result.Groups[0].Files))
	}
}

func TestFindDuplicatesUnsupportedAlgorithm(t *testing.T) {
	_, err := FindDuplicates(DuplicateOptions{Dir: ".", Algorithm: "crc32"}, progress.Nop{})
	if err == nil {
		t.Error("Expected an error for an unsupported hash algorithm")
	}
}

func TestFindDuplicatesMissingDirectory(t *testing.T) {
	_, err := FindDuplicates(DuplicateOptions{Dir: filepath.Join(t.TempDir(), "missing"), Algorithm: "md5"}, progress.Nop{})
	if err == nil {
		t.Error("Expected an error for a missing directory")
	}
}
