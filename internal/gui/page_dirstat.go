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

// dirStatPage shows the statistics of a directory tree.
type dirStatPage struct {
	job *Job

	dir      pathField
	excludes exclusionsField

	// dirsTable lists the subdirectories. It is kept across frames so that
	// its scroll position survives.
	dirsTable table

	scanBtn widget.Clickable
	picker  dirPicker
}

// newDirStatPage builds the directory statistics page.
func newDirStatPage(win *app.Window) *dirStatPage {
	p := &dirStatPage{job: newJob(win), dirsTable: newTable()}
	p.dir.setPath(".")
	p.picker.onChoose = func(path string) {
		p.dir.setPath(path)
	}
	return p
}

// title implements page.
func (p *dirStatPage) title() string { return "Directory stats" }

// layout implements page.
func (p *dirStatPage) layout(gtx layout.Context, th *theme) layout.Dimensions {
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
func (p *dirStatPage) layoutContent(gtx layout.Context, th *theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return pageHeader(gtx, th, "Directory statistics")
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
func (p *dirStatPage) layoutForm(gtx layout.Context, th *theme) layout.Dimensions {
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
func (p *dirStatPage) layoutResults(gtx layout.Context, th *theme) layout.Dimensions {
	result, _ := p.job.output().(*report.DirStatResult)
	if result == nil {
		return layout.Dimensions{}
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return statTiles(gtx, th, []statTile{
					{label: "FILES", value: itoa(result.TotalFiles)},
					{label: "TOTAL SIZE", value: formatSize(result.TotalSize)},
					{label: "DIRECTORIES", value: itoa(len(result.Directories))},
				})
			})
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Flexed(2, func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Right: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return p.layoutTypes(gtx, th, result)
					})
				}),
				layout.Flexed(3, func(gtx layout.Context) layout.Dimensions {
					return p.layoutDirectories(gtx, th, result)
				}),
			)
		}),
	)
}

// layoutTypes draws the breakdown by file type.
func (p *dirStatPage) layoutTypes(gtx layout.Context, th *theme, result *report.DirStatResult) layout.Dimensions {
	entries := make([]barChartEntry, 0, len(result.FileTypes))
	for _, ft := range result.FileTypes {
		entries = append(entries, barChartEntry{
			label:   ft.Extension,
			value:   fmt.Sprintf("%s  %.1f%%", formatSize(ft.TotalSize), ft.Percentage),
			percent: ft.Percentage,
		})
	}

	return fillPanel(gtx, th, "File types", func(gtx layout.Context) layout.Dimensions {
		return barChart(gtx, th, entries)
	})
}

// layoutDirectories lists the subdirectories by size.
func (p *dirStatPage) layoutDirectories(gtx layout.Context, th *theme, result *report.DirStatResult) layout.Dimensions {
	p.dirsTable.columns = []tableColumn{
		{title: "DIRECTORY", width: unit.Dp(190)},
		{title: "FILES", width: unit.Dp(48)},
		{title: "SIZE", width: unit.Dp(78)},
		{title: "SHARE", width: unit.Dp(56)},
	}
	p.dirsTable.empty = "No directories to report."

	rows := make([][]string, 0, len(result.Directories))
	for _, dir := range result.Directories {
		rows = append(rows, []string{
			dir.Path,
			itoa(dir.FileCount),
			formatSize(dir.TotalSize),
			fmt.Sprintf("%.1f%%", dir.Percentage),
		})
	}
	p.dirsTable.setRows(rows)

	return fillPanel(gtx, th, "Subdirectories", func(gtx layout.Context) layout.Dimensions {
		return p.dirsTable.layout(gtx, th)
	})
}

// run starts the analysis in the background.
func (p *dirStatPage) run() {
	dir := p.dir.path()
	if dir == "" {
		dir = "."
	}

	opts := scan.DirStatOptions{
		Dir:         dir,
		ExcludeFile: p.excludes.filePattern(),
		ExcludeDir:  p.excludes.dirPattern(),
	}

	p.job.Run(func(pr progress.Indicator) (any, error) {
		return scan.AnalyzeDirStat(opts, pr)
	})
}
