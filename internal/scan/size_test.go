package scan

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/amurru/filetools/internal/progress"
)

func TestSizeBucketLabel(t *testing.T) {
	tests := []struct {
		size     int64
		expected string
	}{
		{0, "< 1 KB"},
		{512, "< 1 KB"},
		{1024, "1 KB - 10 KB"},
		{10 * 1024, "10 KB - 100 KB"},
		{100 * 1024, "100 KB - 1 MB"},
		{1024 * 1024, "1 MB - 10 MB"},
		{10 * 1024 * 1024, "10 MB - 100 MB"},
		{100 * 1024 * 1024, "> 100 MB"},
	}

	for _, test := range tests {
		label := SizeBucketLabel(test.size)
		if label != test.expected {
			t.Errorf("SizeBucketLabel(%d) = %s, expected %s", test.size, label, test.expected)
		}
	}
}

func TestAnalyzeSizes(t *testing.T) {
	tmpDir := t.TempDir()

	subDir := filepath.Join(tmpDir, "sub")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	files := map[string]int{
		"small.txt":   100,
		"medium.txt":  2048,
		"large.txt":   4096,
		"sub/big.bin": 20 * 1024,
	}

	for path, size := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, path), make([]byte, size), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", path, err)
		}
	}

	result, err := AnalyzeSizes(SizeOptions{Dir: tmpDir, Top: 10}, progress.Nop{})
	if err != nil {
		t.Fatalf("AnalyzeSizes failed: %v", err)
	}

	if result.TotalFiles != 4 {
		t.Errorf("Expected 4 files, got %d", result.TotalFiles)
	}

	expectedSize := int64(100 + 2048 + 4096 + 20*1024)
	if result.TotalSize != expectedSize {
		t.Errorf("Expected total size %d, got %d", expectedSize, result.TotalSize)
	}

	if result.TotalDirs != 1 {
		t.Errorf("Expected 1 directory, got %d", result.TotalDirs)
	}

	// The largest files are reported first
	if len(result.LargestFiles) != 4 {
		t.Fatalf("Expected 4 largest files, got %d", len(result.LargestFiles))
	}
	if result.LargestFiles[0].Size != 20*1024 {
		t.Errorf("Expected the largest file to be 20480 bytes, got %d", result.LargestFiles[0].Size)
	}
	if result.LargestFiles[0].Path != "sub/big.bin" {
		t.Errorf("Expected path 'sub/big.bin', got '%s'", result.LargestFiles[0].Path)
	}

	// Every file must land in a bucket
	if len(result.Buckets) != 3 {
		t.Errorf("Expected 3 buckets, got %d", len(result.Buckets))
	}

	var bucketTotal int
	var bucketSize int64
	for _, bucket := range result.Buckets {
		bucketTotal += bucket.FileCount
		bucketSize += bucket.TotalSize
	}
	if bucketTotal != 4 {
		t.Errorf("Expected 4 files across buckets, got %d", bucketTotal)
	}
	if bucketSize != expectedSize {
		t.Errorf("Expected bucket sizes to sum to %d, got %d", expectedSize, bucketSize)
	}
}

func TestAnalyzeSizesTopLimit(t *testing.T) {
	tmpDir := t.TempDir()

	for _, name := range []string{"a.bin", "b.bin", "c.bin"} {
		if err := os.WriteFile(filepath.Join(tmpDir, name), make([]byte, 10), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", name, err)
		}
	}

	result, err := AnalyzeSizes(SizeOptions{Dir: tmpDir, Top: 2}, progress.Nop{})
	if err != nil {
		t.Fatalf("AnalyzeSizes failed: %v", err)
	}

	if len(result.LargestFiles) != 2 {
		t.Errorf("Expected 2 largest files, got %d", len(result.LargestFiles))
	}
	if result.TotalFiles != 3 {
		t.Errorf("Expected 3 files in total, got %d", result.TotalFiles)
	}
}

func TestAnalyzeSizesEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	result, err := AnalyzeSizes(SizeOptions{Dir: tmpDir, Top: 10}, progress.Nop{})
	if err != nil {
		t.Fatalf("AnalyzeSizes failed: %v", err)
	}

	if result.TotalFiles != 0 {
		t.Errorf("Expected 0 files, got %d", result.TotalFiles)
	}
	if result.TotalSize != 0 {
		t.Errorf("Expected total size 0, got %d", result.TotalSize)
	}
	if len(result.Buckets) != 0 {
		t.Errorf("Expected no buckets, got %d", len(result.Buckets))
	}
}

func TestAnalyzeSizesWithExclusions(t *testing.T) {
	tmpDir := t.TempDir()

	for _, name := range []string{"keep.txt", "skip.log"} {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte("data"), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", name, err)
		}
	}

	result, err := AnalyzeSizes(SizeOptions{Dir: tmpDir, Top: 10, ExcludeFile: "*.log"}, progress.Nop{})
	if err != nil {
		t.Fatalf("AnalyzeSizes failed: %v", err)
	}

	if result.TotalFiles != 1 {
		t.Errorf("Expected 1 file after exclusion, got %d", result.TotalFiles)
	}
	if len(result.Exclusions) != 1 {
		t.Errorf("Expected 1 exclusion, got %d", len(result.Exclusions))
	}
}

func TestAnalyzeSizesMissingDirectory(t *testing.T) {
	_, err := AnalyzeSizes(SizeOptions{Dir: filepath.Join(t.TempDir(), "missing"), Top: 10}, progress.Nop{})
	if err == nil {
		t.Error("Expected an error for a missing directory")
	}
}
