//go:build gui

package gui

import (
	"testing"

	"github.com/amurru/filetools/internal/report"
)

// TestOrganizeCanApplyRequiresCurrentPreview verifies that Apply is only
// allowed when the preview matches the current inputs.
func TestOrganizeCanApplyRequiresCurrentPreview(t *testing.T) {
	p := newOrganizePage(nil)
	p.source.setPath("/data")
	p.criteria.Value = "type"

	// No preview yet.
	if p.canApply() {
		t.Fatal("canApply = true without a preview")
	}

	// A preview for the current inputs with one operation.
	p.previewKey = p.previewInputs()
	p.previewJob.result = &report.OrganizeResult{
		Operations: []report.OrganizeOperation{{OldPath: "a.txt", NewPath: "documents/a.txt"}},
	}
	if !p.canApply() {
		t.Fatal("canApply = false for a matching preview")
	}

	// Changing the source invalidates the preview.
	p.source.setPath("/other")
	if p.canApply() {
		t.Error("canApply = true after the source changed")
	}
	p.source.setPath("/data")

	// Changing Overwrite invalidates the preview.
	p.overwrite.Value = true
	if p.canApply() {
		t.Error("canApply = true after Overwrite changed")
	}
}

// TestRenameCanApplyRequiresCurrentPreview verifies that Apply is only
// allowed when the preview matches the current inputs.
func TestRenameCanApplyRequiresCurrentPreview(t *testing.T) {
	p := newRenamePage(nil)
	p.dir.setPath("/data")
	p.match.setValue("*.jpg")
	p.sed.setValue("s/old/new/")

	if p.canApply() {
		t.Fatal("canApply = true without a preview")
	}

	p.previewKey = p.previewInputs()
	p.previewJob.result = &report.RenameResult{
		Operations: []report.RenameOperation{{OldPath: "a.jpg", NewPath: "b.jpg"}},
	}
	if !p.canApply() {
		t.Fatal("canApply = false for a matching preview")
	}

	p.sed.setValue("s/old/other/")
	if p.canApply() {
		t.Error("canApply = true after the sed expression changed")
	}
}
