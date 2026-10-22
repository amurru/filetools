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

// sizePage shows how much space the files of a directory tree take up.
type sizePage struct {
	job *Job

	dir      pathField
	excludes exclusionsField
	top      topField

	// filesTable lists the largest files. It is kept across frames so that
	// its scroll position survives.
	filesTable table

	scanBtn widget.Clickable
	picker  dirPicker
}

// topField is the numeric input for the number of largest files.
type topField struct {
	editor widget.Editor
}

// layout draws the input.
func (t *topField) layout(gtx layout.Context, th *theme) layout.Dimensions {
	editor := material.Editor(th.Theme, &t.editor, "10")
	return editor.Layout(gtx)
}

// value returns the requested number of files.
func (t *topField) value() int {
	value := 10
	if _, err := fmt.Sscanf(t.editor.Text(), "%d", &value); err != nil {
		return 10
	}
	if value < 0 {
		return 0
	}
	return value
}

// newSizePage builds the size analysis page.
func newSizePage(win *app.Window) *sizePage {
	p := &sizePage{job: newJob(win), filesTable: newTable()}
	p.top.editor.SetText("10")
	p.picker.onChoose = func(path string) {
		p.dir.setPath(path)
	}
	return p
}

// title implements page.
func (p *sizePage) title() string { return "Size analysis" }

// layout implements page.
func (p *sizePage) layout(gtx layout.Context, th *theme) layout.Dimensions {
	return layout.Stack{}.Layout(gtx,
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return p.layoutContent(gtx, th)
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return p.picker.layout(gtx, th)
		}),
	)
}

// layoutContent draws the form and the results of the page.
func (p *sizePage) layoutContent(gtx layout.Context, th *theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return pageHeader(gtx, th, "Size analysis")
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
			return layoutProgress(gtx, th, p.job)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return p.layoutResults(gtx, th)
		}),
	)
}

// layoutForm draws the inputs of the page.
func (p *sizePage) layoutForm(gtx layout.Context, th *theme) layout.Dimensions {
	return panel(gtx, th, "Directory", func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return formRow(gtx, th, "Path", func(gtx layout.Context) layout.Dimensions {
					return p.dir.layout(gtx, th, "/path/to/directory", func() {
						p.picker.open(p.dir.path())
					})
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Spacer{Height: unit.Dp(10)}.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return formRow(gtx, th, "Top files", func(gtx layout.Context) layout.Dimensions {
					return p.top.layout(gtx, th)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Spacer{Height: unit.Dp(10)}.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return p.excludes.layout(gtx, th)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Spacer{Height: unit.Dp(14)}.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if p.scanBtn.Clicked(gtx) {
					p.run()
				}
				button := material.Button(th.Theme, &p.scanBtn, "Analyze")
				return button.Layout(gtx)
			}),
		)
	})
}

// layoutResults draws the outcome of the last analysis.
func (p *sizePage) layoutResults(gtx layout.Context, th *theme) layout.Dimensions {
	result, _ := p.job.output().(*report.SizeReportResult)
	if result == nil {
		return emptyHint(gtx, th, "Choose a directory and run the analysis.")
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return statTiles(gtx, th, []statTile{
					{label: "FILES", value: itoa(result.TotalFiles)},
					{label: "TOTAL SIZE", value: formatSize(result.TotalSize)},
					{label: "DIRECTORIES", value: itoa(result.TotalDirs)},
				})
			})
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Flexed(2, func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Right: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return p.layoutBuckets(gtx, th, result)
					})
				}),
				layout.Flexed(3, func(gtx layout.Context) layout.Dimensions {
					return p.layoutLargest(gtx, th, result)
				}),
			)
		}),
	)
}

// layoutBuckets draws the distribution of the files over the size ranges.
func (p *sizePage) layoutBuckets(gtx layout.Context, th *theme, result *report.SizeReportResult) layout.Dimensions {
	entries := make([]barChartEntry, 0, len(result.Buckets))
	for _, bucket := range result.Buckets {
		entries = append(entries, barChartEntry{
			label:   bucket.Label,
			value:   fmt.Sprintf("%d files  %s", bucket.FileCount, formatSize(bucket.TotalSize)),
			percent: bucket.Percentage,
		})
	}

	return fillPanel(gtx, th, "Size distribution", func(gtx layout.Context) layout.Dimensions {
		return barChart(gtx, th, entries)
	})
}

// layoutLargest lists the largest files of the tree.
func (p *sizePage) layoutLargest(gtx layout.Context, th *theme, result *report.SizeReportResult) layout.Dimensions {
	p.filesTable.columns = []tableColumn{
		{title: "FILE", width: unit.Dp(300)},
		{title: "SIZE", width: unit.Dp(96)},
	}
	p.filesTable.empty = "No files to report."

	rows := make([][]string, 0, len(result.LargestFiles))
	for _, file := range result.LargestFiles {
		rows = append(rows, []string{file.Path, formatSize(file.Size)})
	}
	p.filesTable.setRows(rows)

	return fillPanel(gtx, th, "Largest files", func(gtx layout.Context) layout.Dimensions {
		return p.filesTable.layout(gtx, th)
	})
}

// run starts the analysis in the background.
func (p *sizePage) run() {
	dir := p.dir.path()
	if dir == "" {
		return
	}

	opts := scan.SizeOptions{
		Dir:         dir,
		Top:         p.top.value(),
		ExcludeFile: p.excludes.filePattern(),
		ExcludeDir:  p.excludes.dirPattern(),
	}

	p.job.Run(func(pr progress.Indicator) (any, error) {
		return scan.AnalyzeSizes(opts, pr)
	})
}
