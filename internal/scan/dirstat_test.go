package scan

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/amurru/filetools/internal/progress"
	"github.com/amurru/filetools/internal/report"
)

func TestAnalyzeDirStat(t *testing.T) {
	// Create a temporary directory structure
	tmpDir, err := os.MkdirTemp("", "dirstattest")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Create files with different extensions and sizes
	files := map[string]int{
		"file1.txt":        100,
		"file2.txt":        200,
		"file3.md":         150,
		"subdir/file4.txt": 300,
	}

	for path, size := range files {
		fullPath := filepath.Join(tmpDir, path)
		content := make([]byte, size)
		// Fill with some pattern
		for i := range content {
			content[i] = byte(i % 256)
		}
		if err := os.WriteFile(fullPath, content, 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", path, err)
		}
	}

	result, err := AnalyzeDirStat(DirStatOptions{Dir: tmpDir}, progress.Nop{})
	if err != nil {
		t.Fatalf("AnalyzeDirStat failed: %v", err)
	}

	// Check total file count
	if result.TotalFiles != 4 {
		t.Errorf("Expected 4 files, got %d", result.TotalFiles)
	}

	// Check total size (100 + 200 + 150 + 300 = 750)
	expectedSize := int64(750)
	if result.TotalSize != expectedSize {
		t.Errorf("Expected total size %d, got %d", expectedSize, result.TotalSize)
	}

	// Check largest file
	if result.LargestFile == nil {
		t.Error("Expected largest file to be set")
	} else if result.LargestFile.Size != 300 {
		t.Errorf("Expected largest file size 300, got %d", result.LargestFile.Size)
	}

	// Check file types
	if len(result.FileTypes) != 2 {
		t.Errorf("Expected 2 file types, got %d", len(result.FileTypes))
	}

	// Find .txt file type stats
	var txtType *report.FileType
	for i := range result.FileTypes {
		if result.FileTypes[i].Extension == ".txt" {
			txtType = &result.FileTypes[i]
			break
		}
	}

	if txtType == nil {
		t.Error("Expected .txt file type to be present")
	} else {
		if txtType.Count != 3 {
			t.Errorf("Expected 3 .txt files, got %d", txtType.Count)
		}
		if txtType.TotalSize != 600 { // 100 + 200 + 300
			t.Errorf("Expected .txt total size 600, got %d", txtType.TotalSize)
		}
	}
}

func TestAnalyzeDirStatEmpty(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dirstatempty")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	result, err := AnalyzeDirStat(DirStatOptions{Dir: tmpDir}, progress.Nop{})
	if err != nil {
		t.Fatalf("AnalyzeDirStat failed: %v", err)
	}

	if result.TotalFiles != 0 {
		t.Errorf("Expected 0 files, got %d", result.TotalFiles)
	}

	if result.TotalSize != 0 {
		t.Errorf("Expected total size 0, got %d", result.TotalSize)
	}

	if result.LargestFile != nil {
		t.Error("Expected no largest file for empty directory")
	}
}

func TestAnalyzeDirStatWithExclusions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dirstatexclude")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create files
	files := []string{"file1.txt", "file2.log", "file3.txt"}
	for _, name := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", name, err)
		}
	}

	result, err := AnalyzeDirStat(DirStatOptions{Dir: tmpDir, ExcludeFile: "*.log"}, progress.Nop{})
	if err != nil {
		t.Fatalf("AnalyzeDirStat failed: %v", err)
	}

	// Should only have 2 files (excluded .log)
	if result.TotalFiles != 2 {
		t.Errorf("Expected 2 files after exclusion, got %d", result.TotalFiles)
	}

	// Check exclusions were tracked
	if len(result.Exclusions) != 1 {
		t.Errorf("Expected 1 exclusion, got %d", len(result.Exclusions))
	}
}

func TestAnalyzeDirStatNestedStructure(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dirstatnested")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create nested directory structure
	dirs := []string{"level1", "level1/level2", "level1/level2/level3"}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create files at different levels
	paths := []string{
		"root.txt",
		"level1/level1.txt",
		"level1/level2/level2.txt",
		"level1/level2/level3/level3.txt",
	}

	for _, path := range paths {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.WriteFile(fullPath, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", path, err)
		}
	}

	result, err := AnalyzeDirStat(DirStatOptions{Dir: tmpDir}, progress.Nop{})
	if err != nil {
		t.Fatalf("AnalyzeDirStat failed: %v", err)
	}

	// Check we captured all directories
	if len(result.Directories) < 3 {
		t.Errorf("Expected at least 3 directories, got %d", len(result.Directories))
	}

	// Check total files
	if result.TotalFiles != 4 {
		t.Errorf("Expected 4 files, got %d", result.TotalFiles)
	}
}

func TestAnalyzeDirStatNoExtension(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dirstatnoext")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create files without extensions
	files := []string{"README", "Makefile", "file1.txt"}
	for _, name := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", name, err)
		}
	}

	result, err := AnalyzeDirStat(DirStatOptions{Dir: tmpDir}, progress.Nop{})
	if err != nil {
		t.Fatalf("AnalyzeDirStat failed: %v", err)
	}

	// Check for "(no extension)" file type
	var noExtCount int
	found := false
	for i := range result.FileTypes {
		if result.FileTypes[i].Extension == "(no extension)" {
			noExtCount = result.FileTypes[i].Count
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected '(no extension)' file type to be present")
	} else if noExtCount != 2 {
		t.Errorf("Expected 2 files with no extension, got %d", noExtCount)
	}
}

func TestAnalyzeDirStatEmptyFilesHaveNoPercentage(t *testing.T) {
	tmpDir := t.TempDir()

	// A zero byte file makes the total size zero, which must not produce a
	// division by zero in the percentages.
	if err := os.WriteFile(filepath.Join(tmpDir, "empty.txt"), nil, 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	result, err := AnalyzeDirStat(DirStatOptions{Dir: tmpDir}, progress.Nop{})
	if err != nil {
		t.Fatalf("AnalyzeDirStat failed: %v", err)
	}

	for _, ft := range result.FileTypes {
		if ft.Percentage != 0 {
			t.Errorf("Expected a zero percentage for %s, got %f", ft.Extension, ft.Percentage)
		}
	}
}
