package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"amurru/filetools/internal/exclusions"
	"amurru/filetools/internal/output"
)

func TestParseSedExpression(t *testing.T) {
	tests := []struct {
		input    string
		wantRe   string
		wantRepl string
		wantGlob bool
		wantErr  bool
	}{
		{"s/old/new/", "old", "new", false, false},
		{"s/old/new/g", "old", "new", true, false},
		{"s/(.+)/prefix_$1/", "(.+)", "prefix_$1", false, false},
		{"invalid", "", "", false, true},
		{"s/old/", "", "", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			re, repl, glob, err := parseSedExpression(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSedExpression() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if re.String() != tt.wantRe {
				t.Errorf("parseSedExpression() re = %v, want %v", re.String(), tt.wantRe)
			}
			if repl != tt.wantRepl {
				t.Errorf("parseSedExpression() repl = %v, want %v", repl, tt.wantRepl)
			}
			if glob != tt.wantGlob {
				t.Errorf("parseSedExpression() glob = %v, want %v", glob, tt.wantGlob)
			}
		})
	}
}

func TestPerformRenames(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "rename_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	files := []string{"test1.jpg", "test2.jpg", "other.txt"}
	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Test rename
	re, _ := regexp.Compile("test")
	ops, excl, err := performRenames(tmpDir, "*.jpg", re, "renamed", false, true, false, nil, nil, &output.NoOpProgressIndicator{})
	if err != nil {
		t.Fatal(err)
	}

	if len(excl) != 0 {
		t.Errorf("expected no exclusions, got %d", len(excl))
	}

	if len(ops) != 2 {
		t.Errorf("expected 2 operations, got %d", len(ops))
	}

	// Check operations
	expected := map[string]string{
		"test1.jpg": "renamed1.jpg",
		"test2.jpg": "renamed2.jpg",
	}

	for _, op := range ops {
		if op.Error != "" {
			t.Errorf("unexpected error: %s", op.Error)
		}
		if expectedNew, ok := expected[filepath.Base(op.OldPath)]; !ok || op.NewPath != expectedNew {
			t.Errorf("unexpected rename: %s -> %s", op.OldPath, op.NewPath)
		}
	}
}

func TestPerformRenamesDryRun(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "rename_test_dry")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file
	oldPath := filepath.Join(tmpDir, "old.jpg")
	if err := os.WriteFile(oldPath, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	re, _ := regexp.Compile("old")
	ops, _, err := performRenames(tmpDir, "*.jpg", re, "new", false, true, false, nil, nil, &output.NoOpProgressIndicator{})
	if err != nil {
		t.Fatal(err)
	}

	if len(ops) != 1 {
		t.Errorf("expected 1 operation, got %d", len(ops))
	}

	// File should still exist with old name
	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		t.Error("file was renamed in dry run")
	}
}

func TestPerformRenamesWithDirExclusion(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "rename_test_excl")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Files inside an excluded directory must not be touched
	subDir := filepath.Join(tmpDir, "skipme")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	files := []string{"test1.jpg", "skipme/test2.jpg"}
	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	dirMatchers := exclusions.ParseExclusions("skipme", false)

	re, _ := regexp.Compile("test")
	ops, excl, err := performRenames(tmpDir, "*.jpg", re, "renamed", false, true, false, nil, dirMatchers, &output.NoOpProgressIndicator{})
	if err != nil {
		t.Fatal(err)
	}

	if len(excl) != 1 {
		t.Errorf("expected 1 exclusion, got %d", len(excl))
	}
	if len(ops) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(ops))
	}
	if ops[0].OldPath != "test1.jpg" {
		t.Errorf("expected only the root file to be renamed, got %s", ops[0].OldPath)
	}

	// The excluded file must keep its name
	if _, err := os.Stat(filepath.Join(subDir, "test2.jpg")); err != nil {
		t.Errorf("excluded file was renamed: %v", err)
	}
}

func TestPerformRenamesMultipleDirs(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "rename_test_multi")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test files in root and subdir
	files := []string{"test1.jpg", "subdir/test2.jpg", "subdir/test3.jpg"}
	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Test rename
	re, _ := regexp.Compile("test")
	ops, excl, err := performRenames(tmpDir, "*.jpg", re, "renamed", false, true, false, nil, nil, &output.NoOpProgressIndicator{})
	if err != nil {
		t.Fatal(err)
	}

	if len(excl) != 0 {
		t.Errorf("expected no exclusions, got %d", len(excl))
	}

	if len(ops) != 3 {
		t.Errorf("expected 3 operations, got %d", len(ops))
	}

	// Check operations
	expected := map[string]string{
		"test1.jpg":        "renamed1.jpg",
		"subdir/test2.jpg": "subdir/renamed2.jpg",
		"subdir/test3.jpg": "subdir/renamed3.jpg",
	}

	for _, op := range ops {
		if op.Error != "" {
			t.Errorf("unexpected error: %s", op.Error)
		}
		if expectedNew, ok := expected[op.OldPath]; !ok || op.NewPath != expectedNew {
			t.Errorf("unexpected rename: %s -> %s", op.OldPath, op.NewPath)
		}
	}
}
