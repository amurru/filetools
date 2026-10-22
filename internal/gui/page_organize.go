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

// groupCriteria are the criteria offered by the organizer.
var groupCriteria = []struct {
	key   string
	label string
}{
	{"type", "Type"},
	{"extension", "Extension"},
	{"date", "Date"},
	{"size", "Size"},
}

// organizePage moves the files of a directory tree into category folders.
type organizePage struct {
	previewJob *Job
	applyJob   *Job

	source pathField
	dest   pathField

	criteria  widget.Enum
	hidden    widget.Bool
	overwrite widget.Bool

	// previewKey holds the inputs the current preview was computed from.
	previewKey string

	applied bool

	applyBtn     widget.Clickable
	picker       dirPicker
	sourcePicker dirPicker
	confirm      confirmDialog

	// movesTable lists the planned moves. It is kept across frames so that
	// its scroll position survives.
	movesTable table
}

// newOrganizePage builds the organize page.
func newOrganizePage(win *app.Window) *organizePage {
	p := &organizePage{previewJob: newJob(win), applyJob: newJob(win), movesTable: newTable()}
	p.source.setPath(".")
	p.criteria.Value = "type"
	p.picker.onChoose = func(path string) {
		p.dest.setPath(path)
	}
	p.sourcePicker.onChoose = func(path string) {
		p.source.setPath(path)
	}
	return p
}

// title implements page.
func (p *organizePage) title() string { return "Organize" }

// layout implements page.
func (p *organizePage) layout(gtx layout.Context, th *theme) layout.Dimensions {
	return layout.Stack{}.Layout(gtx,
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return p.layoutContent(gtx, th)
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return p.sourcePicker.layout(gtx, th)
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
func (p *organizePage) layoutContent(gtx layout.Context, th *theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return pageHeader(gtx, th, "Organize files")
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
func (p *organizePage) layoutStatus(gtx layout.Context, th *theme) layout.Dimensions {
	if p.applyJob.isRunning() {
		return layoutProgress(gtx, th, p.applyJob)
	}
	return layoutProgress(gtx, th, p.previewJob)
}

// layoutForm draws the inputs of the page.
func (p *organizePage) layoutForm(gtx layout.Context, th *theme) layout.Dimensions {
	return panel(gtx, th, "Grouping", func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return formRow(gtx, th, "Source", func(gtx layout.Context) layout.Dimensions {
					return p.source.layout(gtx, th, "/path/to/directory", func() {
						p.sourcePicker.open(p.source.path())
					})
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Spacer{Height: unit.Dp(8)}.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return formRow(gtx, th, "Destination", func(gtx layout.Context) layout.Dimensions {
					return p.dest.layout(gtx, th, "same as source", func() {
						p.picker.open(p.dest.path())
					})
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Spacer{Height: unit.Dp(8)}.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return p.layoutCriteria(gtx, th)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Spacer{Height: unit.Dp(8)}.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return p.layoutToggles(gtx, th)
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

// layoutCriteria draws the radio buttons of the grouping criteria.
func (p *organizePage) layoutCriteria(gtx layout.Context, th *theme) layout.Dimensions {
	children := make([]layout.FlexChild, 0, len(groupCriteria))
	for _, criterion := range groupCriteria {
		criterion := criterion
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			button := material.RadioButton(th.Theme, &p.criteria, criterion.key, criterion.label)
			return layout.Inset{Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return button.Layout(gtx)
			})
		}))
	}

	return formRow(gtx, th, "Group by", func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, children...)
	})
}

// layoutToggles draws the checkboxes of the page.
func (p *organizePage) layoutToggles(gtx layout.Context, th *theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			button := material.CheckBox(th.Theme, &p.hidden, "Include hidden files")
			return layout.Inset{Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return button.Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			button := material.CheckBox(th.Theme, &p.overwrite, "Replace existing files")
			return button.Layout(gtx)
		}),
	)
}

// layoutActions draws the buttons of the page.
func (p *organizePage) layoutActions(gtx layout.Context, th *theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if p.applyBtn.Clicked(gtx) {
				p.confirm.ask(p.confirmMessage(), p.apply)
			}
			button := material.Button(th.Theme, &p.applyBtn, "Apply moves")
			return button.Layout(gtx)
		}),
	)
}

// layoutResults draws the preview of the planned moves.
func (p *organizePage) layoutResults(gtx layout.Context, th *theme) layout.Dimensions {
	p.refreshPreview()

	result, _ := p.previewJob.output().(*report.OrganizeResult)
	if result == nil {
		return layout.Dimensions{}
	}

	if len(result.Operations) == 0 {
		label := material.Body2(th.Theme, "No files need organizing.")
		return layout.Inset{Top: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return label.Layout(gtx)
		})
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return statTiles(gtx, th, []statTile{
					{label: "FILES", value: itoa(len(result.Operations))},
					{label: "CATEGORIES", value: itoa(len(result.Categories))},
				})
			})
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return p.layoutMoves(gtx, th, result)
		}),
	)
}

// layoutMoves lists the planned moves.
func (p *organizePage) layoutMoves(gtx layout.Context, th *theme, result *report.OrganizeResult) layout.Dimensions {
	p.movesTable.columns = []tableColumn{
		{title: "FROM", width: unit.Dp(250)},
		{title: "TO", width: unit.Dp(250)},
		{title: "CATEGORY", width: unit.Dp(110)},
		{title: "STATUS", width: unit.Dp(120)},
	}
	p.movesTable.empty = "No files need organizing."

	rows := make([][]string, 0, len(result.Operations))
	for _, op := range result.Operations {
		status := "ok"
		if op.Error != "" {
			status = op.Error
		}
		rows = append(rows, []string{op.OldPath, op.NewPath, op.Category, status})
	}
	p.movesTable.setRows(rows)

	return fillPanel(gtx, th, "Preview", func(gtx layout.Context) layout.Dimensions {
		return p.movesTable.layout(gtx, th)
	})
}

// refreshPreview recomputes the preview when the inputs changed or when the
// last apply changed the files on disk.
func (p *organizePage) refreshPreview() {
	if _, _, _, _, done, _ := p.applyJob.state(); done && !p.applied {
		p.applied = true
		p.previewKey = ""
	}

	key := p.source.path() + "\x00" + p.dest.path() + "\x00" + p.criteria.Value
	if key == p.previewKey || p.previewJob.isRunning() {
		return
	}
	p.previewKey = key

	opts := p.options(true)
	p.previewJob.Run(func(pr progress.Indicator) (any, error) {
		return scan.Organize(opts, pr)
	})
}

// options builds the options of a run.
func (p *organizePage) options(dryRun bool) scan.OrganizeOptions {
	source := p.source.path()
	if source == "" {
		source = "."
	}

	return scan.OrganizeOptions{
		Source:        source,
		Dest:          p.dest.path(),
		By:            p.criteria.Value,
		DryRun:        dryRun,
		Overwrite:     p.overwrite.Value,
		IncludeHidden: p.hidden.Value,
	}
}

// confirmMessage describes the planned moves.
func (p *organizePage) confirmMessage() string {
	result, _ := p.previewJob.output().(*report.OrganizeResult)
	if result == nil {
		return "No files will be moved."
	}
	return fmt.Sprintf("%d files will be moved into their category folders.", len(result.Operations))
}

// apply performs the moves for real.
func (p *organizePage) apply() {
	opts := p.options(false)
	p.applied = false
	p.applyJob.Run(func(pr progress.Indicator) (any, error) {
		return scan.Organize(opts, pr)
	})
}
