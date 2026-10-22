package scan

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"amurru/filetools/internal/progress"
)

// cancelledIndicator reports cancellation immediately. The engines must
// abort with ErrCancelled instead of walking the tree.
type cancelledIndicator struct{ progress.Nop }

func (cancelledIndicator) Cancelled() bool { return true }
func (cancelledIndicator) WaitIfPaused()   {}

func writeSizedFile(t *testing.T, path string, size int) {
	t.Helper()
	content := make([]byte, size)
	for i := range content {
		content[i] = byte(i % 256)
	}
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("Failed to create %s: %v", path, err)
	}
}

func TestEnginesAbortWhenCancelled(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "canceltest")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writeSizedFile(t, filepath.Join(tmpDir, "a.txt"), 100)
	writeSizedFile(t, filepath.Join(tmpDir, "b.txt"), 100)

	p := cancelledIndicator{}

	if _, err := AnalyzeDirStat(DirStatOptions{Dir: tmpDir}, p); !errors.Is(err, progress.ErrCancelled) {
		t.Errorf("AnalyzeDirStat: expected ErrCancelled, got %v", err)
	}
	if _, err := AnalyzeSizes(SizeOptions{Dir: tmpDir}, p); !errors.Is(err, progress.ErrCancelled) {
		t.Errorf("AnalyzeSizes: expected ErrCancelled, got %v", err)
	}
	if _, err := Rename(RenameOptions{Dir: tmpDir, Match: "*", Sed: "s/a/b/"}, p); !errors.Is(err, progress.ErrCancelled) {
		t.Errorf("Rename: expected ErrCancelled, got %v", err)
	}
	if _, err := Organize(OrganizeOptions{Source: tmpDir, By: "type", DryRun: true}, p); !errors.Is(err, progress.ErrCancelled) {
		t.Errorf("Organize: expected ErrCancelled, got %v", err)
	}
	if _, err := FindDuplicates(DuplicateOptions{Dir: tmpDir, Algorithm: "md5"}, p); !errors.Is(err, progress.ErrCancelled) {
		t.Errorf("FindDuplicates: expected ErrCancelled, got %v", err)
	}
}
