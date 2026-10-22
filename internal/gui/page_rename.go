//go:build gui

package gui

import (
	"fmt"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"amurru/filetools/internal/progress"
	"amurru/filetools/internal/report"
	"amurru/filetools/internal/scan"
)

// renamePage renames the files matching a pattern.
//
// The page keeps two jobs: one computes the preview of the planned renames
// and one performs them. Keeping them apart makes sure the preview is
// always a dry run and never changes the filesystem by accident.
type renamePage struct {
	previewJob *Job
	applyJob   *Job

	dir   pathField
	match textField
	sed   textField

	// debouncer delays the recomputation of the preview while the inputs
	// are changing.
	debouncer previewDebouncer

	// applied reports whether the outcome of the last apply was already
	// taken into account.
	applied bool

	applyBtn widget.Clickable
	picker   dirPicker
	confirm  confirmDialog

	// previewTable lists the planned renames. It is kept across frames so
	// that its scroll position survives.
	previewTable table
}

// newRenamePage builds the rename page.
func newRenamePage(win *app.Window) *renamePage {
	p := &renamePage{previewJob: newJob(win), applyJob: newJob(win), previewTable: newTable()}
	p.match.setValue("*")
	p.picker.onChoose = func(path string) {
		p.dir.setPath(path)
	}
	return p
}

// title implements page.
func (p *renamePage) title() string { return "Rename" }

// layout implements page.
func (p *renamePage) layout(gtx layout.Context, th *theme) layout.Dimensions {
	return layout.Stack{}.Layout(gtx,
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return p.layoutContent(gtx, th)
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return p.picker.layout(gtx, th)
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return p.confirm.layout(gtx, th)
		}),
	)
}

// layoutContent draws the form and the results of the page.
func (p *renamePage) layoutContent(gtx layout.Context, th *theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return pageHeader(gtx, th, "Rename files")
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Spacer{Height: unit.Dp(14)}.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return p.layoutForm(gtx, th)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Spacer{Height: unit.Dp(14)}.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return p.layoutStatus(gtx, th)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return p.layoutResults(gtx, th)
		}),
	)
}

// layoutStatus draws the progress of the running job.
func (p *renamePage) layoutStatus(gtx layout.Context, th *theme) layout.Dimensions {
	if p.applyJob.isRunning() {
		return layoutProgress(gtx, th, p.applyJob)
	}
	return layoutProgress(gtx, th, p.previewJob)
}

// layoutForm draws the inputs of the page.
func (p *renamePage) layoutForm(gtx layout.Context, th *theme) layout.Dimensions {
	return panel(gtx, th, "Pattern", func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return formRow(gtx, th, "Path", func(gtx layout.Context) layout.Dimensions {
					return p.dir.layout(gtx, th, "/path/to/directory", func() {
						p.picker.open(p.dir.path())
					})
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Spacer{Height: unit.Dp(8)}.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return formRow(gtx, th, "Match", func(gtx layout.Context) layout.Dimensions {
							return p.match.layout(gtx, th, "*.jpg")
						})
					}),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return formRow(gtx, th, "Replace", func(gtx layout.Context) layout.Dimensions {
							return p.sed.layout(gtx, th, "s/old/new/g")
						})
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Spacer{Height: unit.Dp(12)}.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return p.layoutActions(gtx, th)
			}),
		)
	})
}

// layoutActions draws the buttons of the page.
func (p *renamePage) layoutActions(gtx layout.Context, th *theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if p.applyBtn.Clicked(gtx) {
				p.confirm.ask(p.confirmMessage(), p.apply)
			}
			button := material.Button(th.Theme, &p.applyBtn, "Apply renames")
			return button.Layout(gtx)
		}),
	)
}

// layoutResults draws the preview of the planned renames.
func (p *renamePage) layoutResults(gtx layout.Context, th *theme) layout.Dimensions {
	p.refreshPreview(gtx)

	result, _ := p.previewJob.output().(*report.RenameResult)
	if result == nil {
		return emptyHint(gtx, th, "Choose a directory to see the files that would be renamed.")
	}

	if len(result.Operations) == 0 {
		label := material.Body2(th.Theme, "No files match the pattern.")
		return layout.Inset{Top: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return label.Layout(gtx)
		})
	}

	p.previewTable.columns = []tableColumn{
		{title: "FROM", width: unit.Dp(280)},
		{title: "TO", width: unit.Dp(280)},
		{title: "STATUS", width: unit.Dp(140)},
	}
	p.previewTable.empty = "No files match the pattern."

	rows := make([][]string, 0, len(result.Operations))
	for _, op := range result.Operations {
		status := "ok"
		if op.Error != "" {
			status = op.Error
		}
		rows = append(rows, []string{op.OldPath, op.NewPath, status})
	}
	p.previewTable.setRows(rows)

	return fillPanel(gtx, th, "Preview", func(gtx layout.Context) layout.Dimensions {
		return p.previewTable.layout(gtx, th)
	})
}

// refreshPreview recomputes the preview when the inputs changed or when the
// last apply changed the files on disk.
func (p *renamePage) refreshPreview(gtx layout.Context) {
	// An apply that just finished invalidates the preview.
	if _, _, _, _, done, _ := p.applyJob.state(); done && !p.applied {
		p.applied = true
		p.debouncer.invalidate()
	}

	// Without a directory there is nothing to preview.
	if p.dir.path() == "" {
		return
	}

	key := p.dir.path() + "\x00" + p.match.value() + "\x00" + p.sed.value()
	if !p.debouncer.refresh(gtx, key) || p.previewJob.isRunning() {
		return
	}

	opts := p.options(true)
	p.previewJob.Run(func(pr progress.Indicator) (any, error) {
		return scan.Rename(opts, pr)
	})
}

// options builds the options of a run.
func (p *renamePage) options(dryRun bool) scan.RenameOptions {
	dir := p.dir.path()
	if dir == "" {
		dir = "."
	}

	return scan.RenameOptions{
		Dir:    dir,
		Match:  p.match.value(),
		Sed:    p.sed.value(),
		DryRun: dryRun,
	}
}

// confirmMessage describes the planned renames.
func (p *renamePage) confirmMessage() string {
	result, _ := p.previewJob.output().(*report.RenameResult)
	if result == nil {
		return "No files will be renamed."
	}
	return fmt.Sprintf("%d files will be renamed. This cannot be undone.", len(result.Operations))
}

// apply performs the renames for real.
func (p *renamePage) apply() {
	opts := p.options(false)
	p.applied = false
	p.applyJob.Run(func(pr progress.Indicator) (any, error) {
		return scan.Rename(opts, pr)
	})
}
