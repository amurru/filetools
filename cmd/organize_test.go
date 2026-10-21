package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"amurru/filetools/internal/exclusions"
	"amurru/filetools/internal/output"
)

// createOrganizeTree creates a directory with one file per extension and returns its path
func createOrganizeTree(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()

	subDir := filepath.Join(tmpDir, "sub")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	files := map[string]int{
		"a.txt":     10,
		"b.png":     20,
		"c.go":      30,
		"sub/d.pdf": 40,
	}

	for path, size := range files {
		content := make([]byte, size)
		if err := os.WriteFile(filepath.Join(tmpDir, path), content, 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", path, err)
		}
	}

	return tmpDir
}

func TestCategorizeFile(t *testing.T) {
	tmpDir := t.TempDir()

	filePath := filepath.Join(tmpDir, "photo.PNG")
	if err := os.WriteFile(filePath, []byte("12345"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	tests := []struct {
		by       string
		expected string
	}{
		{"type", "images"},
		{"extension", "png"},
		{"size", "< 1 KB"},
		{"unknown", "images"}, // falls back to type based categorization
	}

	for _, test := range tests {
		t.Run(test.by, func(t *testing.T) {
			category := categorizeFile(filePath, info, test.by)
			if category != test.expected {
				t.Errorf("categorizeFile(by=%s) = %s, expected %s", test.by, category, test.expected)
			}
		})
	}
}

func TestCategorizeFileNoExtension(t *testing.T) {
	tmpDir := t.TempDir()

	filePath := filepath.Join(tmpDir, "README")
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	if category := categorizeFile(filePath, info, "extension"); category != "no-extension" {
		t.Errorf("Expected 'no-extension', got %s", category)
	}
	if category := categorizeFile(filePath, info, "type"); category != "other" {
		t.Errorf("Expected 'other', got %s", category)
	}
}

func TestCategorizeFileByDate(t *testing.T) {
	tmpDir := t.TempDir()

	filePath := filepath.Join(tmpDir, "dated.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Pin the modification time so the expected month is deterministic
	stamp := time.Date(2024, time.March, 15, 10, 0, 0, 0, time.UTC)
	if err := os.Chtimes(filePath, stamp, stamp); err != nil {
		t.Fatalf("Failed to set file times: %v", err)
	}

	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	if category := categorizeFile(filePath, info, "date"); category != "2024-03" {
		t.Errorf("Expected '2024-03', got %s", category)
	}
}

func TestOrganizeFilesDryRun(t *testing.T) {
	tmpDir := createOrganizeTree(t)

	opts := organizeOptions{By: "type", DestDir: tmpDir, DryRun: true}
	ops, categories, excl, err := organizeFiles(tmpDir, opts, nil, nil, &output.NoOpProgressIndicator{})
	if err != nil {
		t.Fatalf("organizeFiles failed: %v", err)
	}

	if len(ops) != 4 {
		t.Errorf("Expected 4 operations, got %d", len(ops))
	}
	if len(categories) != 3 {
		t.Errorf("Expected 3 categories, got %d", len(categories))
	}
	if len(excl) != 0 {
		t.Errorf("Expected no exclusions, got %d", len(excl))
	}

	// The files must not have moved
	for _, name := range []string{"a.txt", "b.png", "c.go"} {
		if _, err := os.Stat(filepath.Join(tmpDir, name)); err != nil {
			t.Errorf("File %s was moved during a dry run: %v", name, err)
		}
	}
}

func TestOrganizeFilesMove(t *testing.T) {
	tmpDir := createOrganizeTree(t)

	opts := organizeOptions{By: "type", DestDir: tmpDir, DryRun: false}
	ops, categories, _, err := organizeFiles(tmpDir, opts, nil, nil, &output.NoOpProgressIndicator{})
	if err != nil {
		t.Fatalf("organizeFiles failed: %v", err)
	}

	if len(ops) != 4 {
		t.Fatalf("Expected 4 operations, got %d", len(ops))
	}

	for _, op := range ops {
		if op.Error != "" {
			t.Errorf("Unexpected error for %s: %s", op.OldPath, op.Error)
		}
	}

	// Files must now live in their category directory
	expected := map[string]string{
		"documents/a.txt": "",
		"images/b.png":    "",
		"code/c.go":       "",
		"documents/d.pdf": "",
	}

	for path := range expected {
		if _, err := os.Stat(filepath.Join(tmpDir, path)); err != nil {
			t.Errorf("Expected file at %s: %v", path, err)
		}
	}

	// The category summary must match the moved files
	if len(categories) != 3 {
		t.Errorf("Expected 3 categories, got %d", len(categories))
	}
	var total int
	for _, cat := range categories {
		total += cat.FileCount
	}
	if total != 4 {
		t.Errorf("Expected 4 files across categories, got %d", total)
	}
}

func TestOrganizeFilesIsIdempotent(t *testing.T) {
	tmpDir := createOrganizeTree(t)

	opts := organizeOptions{By: "type", DestDir: tmpDir, DryRun: false}
	if _, _, _, err := organizeFiles(tmpDir, opts, nil, nil, &output.NoOpProgressIndicator{}); err != nil {
		t.Fatalf("organizeFiles failed: %v", err)
	}

	// A second run must not find anything left to do
	ops, _, _, err := organizeFiles(tmpDir, opts, nil, nil, &output.NoOpProgressIndicator{})
	if err != nil {
		t.Fatalf("organizeFiles failed: %v", err)
	}
	if len(ops) != 0 {
		t.Errorf("Expected no operations on the second run, got %d", len(ops))
	}
}

func TestOrganizeFilesCollision(t *testing.T) {
	tmpDir := createOrganizeTree(t)

	// Create a file that would collide with the target
	docsDir := filepath.Join(tmpDir, "documents")
	if err := os.Mkdir(docsDir, 0755); err != nil {
		t.Fatalf("Failed to create documents dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "a.txt"), []byte("existing"), 0644); err != nil {
		t.Fatalf("Failed to create colliding file: %v", err)
	}

	opts := organizeOptions{By: "type", DestDir: tmpDir, DryRun: false}
	ops, _, _, err := organizeFiles(tmpDir, opts, nil, nil, &output.NoOpProgressIndicator{})
	if err != nil {
		t.Fatalf("organizeFiles failed: %v", err)
	}

	var collision *output.OrganizeOperation
	for i := range ops {
		if ops[i].OldPath == "a.txt" {
			collision = &ops[i]
		}
	}

	if collision == nil {
		t.Fatal("Expected an operation for a.txt")
	}
	if collision.Error == "" {
		t.Error("Expected an error for the colliding file")
	}

	// The existing file must be left untouched
	content, err := os.ReadFile(filepath.Join(docsDir, "a.txt"))
	if err != nil {
		t.Fatalf("Failed to read colliding file: %v", err)
	}
	if string(content) != "existing" {
		t.Errorf("Expected the existing file to be kept, got %q", string(content))
	}
}

func TestOrganizeFilesOverwrite(t *testing.T) {
	tmpDir := createOrganizeTree(t)

	docsDir := filepath.Join(tmpDir, "documents")
	if err := os.Mkdir(docsDir, 0755); err != nil {
		t.Fatalf("Failed to create documents dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "a.txt"), []byte("existing"), 0644); err != nil {
		t.Fatalf("Failed to create colliding file: %v", err)
	}

	opts := organizeOptions{By: "type", DestDir: tmpDir, DryRun: false, Overwrite: true}
	ops, _, _, err := organizeFiles(tmpDir, opts, nil, nil, &output.NoOpProgressIndicator{})
	if err != nil {
		t.Fatalf("organizeFiles failed: %v", err)
	}

	for _, op := range ops {
		if op.OldPath == "a.txt" && op.Error != "" {
			t.Errorf("Expected no error with overwrite enabled, got %s", op.Error)
		}
	}

	content, err := os.ReadFile(filepath.Join(docsDir, "a.txt"))
	if err != nil {
		t.Fatalf("Failed to read overwritten file: %v", err)
	}
	if string(content) == "existing" {
		t.Error("Expected the existing file to be replaced")
	}
}

func TestOrganizeFilesHiddenFiles(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, ".hidden.txt"), []byte("secret"), 0644); err != nil {
		t.Fatalf("Failed to create hidden file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "visible.txt"), []byte("data"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Hidden files are skipped by default
	opts := organizeOptions{By: "type", DestDir: tmpDir, DryRun: true}
	ops, _, _, err := organizeFiles(tmpDir, opts, nil, nil, &output.NoOpProgressIndicator{})
	if err != nil {
		t.Fatalf("organizeFiles failed: %v", err)
	}
	if len(ops) != 1 {
		t.Errorf("Expected 1 operation, got %d", len(ops))
	}

	// Unless they are explicitly requested
	opts.IncludeHidden = true
	ops, _, _, err = organizeFiles(tmpDir, opts, nil, nil, &output.NoOpProgressIndicator{})
	if err != nil {
		t.Fatalf("organizeFiles failed: %v", err)
	}
	if len(ops) != 2 {
		t.Errorf("Expected 2 operations, got %d", len(ops))
	}
}

func TestOrganizeFilesCategoryFilter(t *testing.T) {
	tmpDir := createOrganizeTree(t)

	opts := organizeOptions{By: "type", DestDir: tmpDir, DryRun: true, Category: "images"}
	ops, categories, _, err := organizeFiles(tmpDir, opts, nil, nil, &output.NoOpProgressIndicator{})
	if err != nil {
		t.Fatalf("organizeFiles failed: %v", err)
	}

	if len(ops) != 1 {
		t.Fatalf("Expected 1 operation, got %d", len(ops))
	}
	if ops[0].Category != "images" {
		t.Errorf("Expected category 'images', got %s", ops[0].Category)
	}
	if len(categories) != 1 {
		t.Errorf("Expected 1 category, got %d", len(categories))
	}
}

func TestOrganizeFilesWithExclusions(t *testing.T) {
	tmpDir := createOrganizeTree(t)

	fileMatchers := exclusions.ParseExclusions("*.go", true)

	opts := organizeOptions{By: "type", DestDir: tmpDir, DryRun: true}
	ops, _, excl, err := organizeFiles(tmpDir, opts, fileMatchers, nil, &output.NoOpProgressIndicator{})
	if err != nil {
		t.Fatalf("organizeFiles failed: %v", err)
	}

	if len(excl) != 1 {
		t.Errorf("Expected 1 exclusion, got %d", len(excl))
	}
	for _, op := range ops {
		if op.OldPath == "c.go" {
			t.Error("Expected the excluded file to be skipped")
		}
	}
}

func TestOrganizeFilesSeparateDestination(t *testing.T) {
	tmpDir := createOrganizeTree(t)
	destDir := filepath.Join(t.TempDir(), "organized")

	opts := organizeOptions{By: "extension", DestDir: destDir, DryRun: false}
	ops, _, _, err := organizeFiles(tmpDir, opts, nil, nil, &output.NoOpProgressIndicator{})
	if err != nil {
		t.Fatalf("organizeFiles failed: %v", err)
	}

	if len(ops) != 4 {
		t.Fatalf("Expected 4 operations, got %d", len(ops))
	}

	for _, op := range ops {
		if op.Error != "" {
			t.Errorf("Unexpected error: %s", op.Error)
		}
	}

	// Files must be moved to the destination directory
	if _, err := os.Stat(filepath.Join(destDir, "txt", "a.txt")); err != nil {
		t.Errorf("Expected file at destination: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "a.txt")); !os.IsNotExist(err) {
		t.Error("Expected the source file to be gone")
	}
}
