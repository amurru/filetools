//go:build gui

package gui

import (
	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/amurru/filetools/internal/progress"
	"github.com/amurru/filetools/internal/report"
	"github.com/amurru/filetools/internal/scan"
)

// hashAlgorithms are the hash algorithms offered by the duplicate finder.
var hashAlgorithms = []struct {
	key   string
	label string
}{
	{"md5", "MD5"},
	{"sha1", "SHA-1"},
	{"sha256", "SHA-256"},
}

// dupfindPage reports the files that have identical content.
type dupfindPage struct {
	job *Job

	dir      pathField
	excludes exclusionsField
	minSize  sizeField
	maxSize  sizeField

	algorithm widget.Enum
	scanBtn   widget.Clickable
	picker    dirPicker

	// groupsTable lists the duplicate groups. It is kept across frames so
	// that its scroll position survives.
	groupsTable table
	// groupsFor is the result the rows of the table were built from.
	groupsFor *report.DuplicateResult
}

// sizeField is a numeric input for a size limit.
type sizeField struct {
	editor widget.Editor
}

// layout draws the input.
func (s *sizeField) layout(gtx layout.Context, th *theme, hint string) layout.Dimensions {
	s.editor.SingleLine = true
	editor := material.Editor(th.Theme, &s.editor, hint)
	return editor.Layout(gtx)
}

// value returns the parsed size in bytes. An unparsable or empty value
// means no limit.
func (s *sizeField) value() int64 {
	size, err := scan.ParseSize(s.editor.Text())
	if err != nil {
		return 0
	}
	return size
}

// newDupfindPage builds the duplicate finder page.
func newDupfindPage(win *app.Window) *dupfindPage {
	p := &dupfindPage{job: newJob(win), groupsTable: newTable()}
	p.algorithm.Value = "md5"
	p.picker.onChoose = func(path string) {
		p.dir.setPath(path)
	}
	return p
}

// title implements page.
func (p *dupfindPage) title() string { return "Duplicate files" }

// layout implements page.
func (p *dupfindPage) layout(gtx layout.Context, th *theme) layout.Dimensions {
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
func (p *dupfindPage) layoutContent(gtx layout.Context, th *theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return pageHeader(gtx, th, "Duplicate files")
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
func (p *dupfindPage) layoutForm(gtx layout.Context, th *theme) layout.Dimensions {
	return panel(gtx, th, "Search", func(gtx layout.Context) layout.Dimensions {
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
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return formRow(gtx, th, "Min size", func(gtx layout.Context) layout.Dimensions {
							return p.minSize.layout(gtx, th, "1MB")
						})
					}),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return formRow(gtx, th, "Max size", func(gtx layout.Context) layout.Dimensions {
							return p.maxSize.layout(gtx, th, "100MB")
						})
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Spacer{Height: unit.Dp(10)}.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return p.excludes.layout(gtx, th)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Spacer{Height: unit.Dp(10)}.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return p.layoutAlgorithm(gtx, th)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Spacer{Height: unit.Dp(12)}.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if p.scanBtn.Clicked(gtx) {
					p.run()
				}
				button := material.Button(th.Theme, &p.scanBtn, "Find duplicates")
				return button.Layout(gtx)
			}),
		)
	})
}

// layoutAlgorithm draws the radio buttons of the hash algorithms.
func (p *dupfindPage) layoutAlgorithm(gtx layout.Context, th *theme) layout.Dimensions {
	children := make([]layout.FlexChild, 0, len(hashAlgorithms))
	for _, algorithm := range hashAlgorithms {
		algorithm := algorithm
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			button := material.RadioButton(th.Theme, &p.algorithm, algorithm.key, algorithm.label)
			return layout.Inset{Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return button.Layout(gtx)
			})
		}))
	}

	return formRow(gtx, th, "Hash", func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, children...)
	})
}

// layoutResults draws the outcome of the last search.
func (p *dupfindPage) layoutResults(gtx layout.Context, th *theme) layout.Dimensions {
	result, _ := p.job.output().(*report.DuplicateResult)
	if result == nil {
		return emptyHint(gtx, th, "Choose a directory and start the search.")
	}

	if !result.Found {
		label := material.Body2(th.Theme, "No duplicate files found.")
		return layout.Inset{Top: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return label.Layout(gtx)
		})
	}

	files := 0
	reclaimable := int64(0)
	for _, group := range result.Groups {
		files += len(group.Files)
		if group.Size > 0 {
			reclaimable += group.Size * int64(len(group.Files)-1)
		}
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return statTiles(gtx, th, []statTile{
					{label: "GROUPS", value: itoa(len(result.Groups))},
					{label: "FILES", value: itoa(files)},
					{label: "RECLAIMABLE", value: formatSize(reclaimable)},
				})
			})
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return p.layoutGroups(gtx, th, result)
		}),
	)
}

// layoutGroups lists the groups of identical files.
func (p *dupfindPage) layoutGroups(gtx layout.Context, th *theme, result *report.DuplicateResult) layout.Dimensions {
	p.groupsTable.columns = []tableColumn{
		{title: "FILE", width: unit.Dp(330)},
		{title: "SIZE", width: unit.Dp(96)},
		{title: "HASH", width: unit.Dp(100)},
	}
	p.groupsTable.empty = "No duplicate files found."

	// The rows only change when the result changes, so they are built once
	// per result instead of once per frame.
	if p.groupsFor != result {
		p.groupsFor = result
		var rows [][]string
		for _, group := range result.Groups {
			hash := group.Hash
			if len(hash) > 8 {
				hash = hash[:8]
			}
			for _, file := range group.Files {
				rows = append(rows, []string{file, formatSize(group.Size), hash})
			}
		}
		p.groupsTable.setRows(rows)
	}

	return fillPanel(gtx, th, "Groups", func(gtx layout.Context) layout.Dimensions {
		return p.groupsTable.layout(gtx, th)
	})
}

// run starts the search in the background.
func (p *dupfindPage) run() {
	dir := p.dir.path()
	if dir == "" {
		return
	}

	opts := scan.DuplicateOptions{
		Dir:         dir,
		Algorithm:   p.algorithm.Value,
		MinSize:     p.minSize.value(),
		MaxSize:     p.maxSize.value(),
		ExcludeFile: p.excludes.filePattern(),
		ExcludeDir:  p.excludes.dirPattern(),
	}

	p.job.Run(func(pr progress.Indicator) (any, error) {
		return scan.FindDuplicates(opts, pr)
	})
}
