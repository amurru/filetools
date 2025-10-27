package cmd

import (
	"os"
	"path/filepath"
	"testing"
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

	hashMap, err := findDuplicates(tmpDir, "md5")
	if err != nil {
		t.Fatalf("findDuplicates failed: %v", err)
	}

	// Check that we have one duplicate group
	duplicateGroups := 0
	for _, files := range hashMap {
		if len(files) > 1 {
			duplicateGroups++
			if len(files) != 2 {
				t.Errorf("Expected 2 duplicate files, got %d", len(files))
			}
		}
	}

	if duplicateGroups != 1 {
		t.Errorf("Expected 1 duplicate group, got %d", duplicateGroups)
	}
}
