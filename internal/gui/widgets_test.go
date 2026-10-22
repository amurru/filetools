//go:build gui

package gui

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDirPickerNavigation verifies that clicking an entry of the directory
// list navigates into it.
//
// The list keeps drawing the entries of the old listing while the click is
// handled, so the navigation must not change the list before it is drawn.
func TestDirPickerNavigation(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{
		filepath.Join(root, "a"),
		filepath.Join(root, "a", "empty"),
		filepath.Join(root, "b"),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create %s: %v", dir, err)
		}
	}

	d := &dirPicker{}
	d.open(root)

	gtx := testContext(520, 600)

	// The first layout registers the click areas.
	d.layout(gtx, newTheme(lightPalette))

	// Click the first entry, which leads to a directory with fewer entries
	// than the current one.
	d.dirClicks[0].Click()
	gtx.Ops.Reset()
	d.layout(gtx, newTheme(lightPalette))

	want := filepath.Join(root, "a")
	if got := d.path.path(); got != want {
		t.Errorf("path = %q, want %q", got, want)
	}
}

// TestDirPickerOpensAbsolutePath verifies that the picker works with an
// absolute path, so that the user can walk up from the working directory.
func TestDirPickerOpensAbsolutePath(t *testing.T) {
	d := &dirPicker{}
	d.open("")

	if !filepath.IsAbs(d.path.path()) {
		t.Errorf("path = %q, want an absolute path", d.path.path())
	}
}

// TestDirPickerWalksUp verifies that the picker can leave the directory it
// started in.
func TestDirPickerWalksUp(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatalf("Failed to create %s: %v", sub, err)
	}

	d := &dirPicker{}
	d.open(sub)

	if got := parentDirectory(d.path.path()); got != root {
		t.Errorf("parent = %q, want %q", got, root)
	}
}
