//go:build gui

package gui

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"amurru/filetools/internal/progress"
)

// labelWidth is the width of the labels of the form rows.
const labelWidth = unit.Dp(116)

// panel draws a titled panel around the given content.
//
// The panel hugs its content: the background is painted with exactly the
// size of the content, which is measured first.
func panel(gtx layout.Context, th *theme, title string, content layout.Widget) layout.Dimensions {
	return backdrop(gtx, panelBackground(th), cornerRadius, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if title == "" {
						return layout.Dimensions{}
					}
					label := material.Subtitle2(th.Theme, title)
					return layout.Inset{Bottom: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return label.Layout(gtx)
					})
				}),
				layout.Rigid(content),
			)
		})
	})
}

// fillPanel draws a titled panel that fills the space it is given. It is
// used for the result areas, whose content needs a bounded height to
// scroll.
func fillPanel(gtx layout.Context, th *theme, title string, content layout.Widget) layout.Dimensions {
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			paintBackground(gtx, panelBackground(th), cornerRadius)
			return layout.Dimensions{Size: gtx.Constraints.Min}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if title == "" {
							return layout.Dimensions{}
						}
						label := material.Subtitle2(th.Theme, title)
						return layout.Inset{Bottom: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return label.Layout(gtx)
						})
					}),
					layout.Flexed(1, content),
				)
			})
		}),
	)
}

// backdrop measures the content, paints a rounded background behind it and
// replays the content on top.
//
// The content is measured first because the background must have exactly the
// size of the content.
func backdrop(gtx layout.Context, c color.NRGBA, radius unit.Dp, content layout.Widget) layout.Dimensions {
	record := op.Record(gtx.Ops)
	dims := content(gtx)
	call := record.Stop()

	paintBackgroundRect(gtx, c, radius, dims.Size)
	call.Add(gtx.Ops)

	return dims
}

// paintBackground fills the whole area available to the context with the
// given color.
func paintBackground(gtx layout.Context, c color.NRGBA, radius unit.Dp) {
	paintBackgroundRect(gtx, c, radius, gtx.Constraints.Min)
}

// paintBackgroundRect fills a rectangle of the given size at the current
// origin with the given color.
func paintBackgroundRect(gtx layout.Context, c color.NRGBA, radius unit.Dp, size image.Point) {
	r := int(gtx.Metric.Dp(radius))

	defer clip.RRect{
		Rect: image.Rectangle{Max: size},
		SE:   r, SW: r, NW: r, NE: r,
	}.Push(gtx.Ops).Pop()

	paint.ColorOp{Color: c}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
}

// panelBackground returns the background color of a panel.
func panelBackground(th *theme) color.NRGBA {
	return th.palette.Panel
}

// fixedWidth lays out the widget with the given width and reports that
// width, so that the columns of a table or a form line up.
//
// Gio reports the size of a widget by the size of its content, so the width
// has to be enforced explicitly.
func fixedWidth(gtx layout.Context, width unit.Dp, w layout.Widget) layout.Dimensions {
	pixels := gtx.Metric.Dp(width)
	gtx.Constraints.Min.X = pixels
	gtx.Constraints.Max.X = pixels

	dims := w(gtx)
	if dims.Size.X > pixels {
		dims.Size.X = pixels
	}

	return dims
}

// formRow lays out a label and a control next to each other. The label has
// a fixed width, so that the controls of a form line up.
func formRow(gtx layout.Context, th *theme, label string, control layout.Widget) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return fixedWidth(gtx, labelWidth, func(gtx layout.Context) layout.Dimensions {
				l := material.Body2(th.Theme, label)
				return l.Layout(gtx)
			})
		}),
		layout.Flexed(1, control),
	)
}

// textField is a single line text input.
type textField struct {
	editor widget.Editor
}

// layout draws the input.
func (t *textField) layout(gtx layout.Context, th *theme, hint string) layout.Dimensions {
	t.editor.SingleLine = true
	editor := material.Editor(th.Theme, &t.editor, hint)
	return editor.Layout(gtx)
}

// value returns the current text.
func (t *textField) value() string {
	return strings.TrimSpace(t.editor.Text())
}

// setValue replaces the text.
func (t *textField) setValue(value string) {
	t.editor.SetText(value)
}

// pathField is a text field for a directory path with a browse button.
type pathField struct {
	editor widget.Editor
	browse widget.Clickable
	// hideBrowse removes the browse button, for fields that are part of a
	// picker already.
	hideBrowse bool
}

// layout draws the path field.
//
// When the browse button is pressed, onBrowse is called so that the page can
// open a directory picker.
func (p *pathField) layout(gtx layout.Context, th *theme, hint string, onBrowse func()) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			p.editor.SingleLine = true
			editor := material.Editor(th.Theme, &p.editor, hint)
			return editor.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if p.hideBrowse {
				return layout.Dimensions{}
			}
			clicked := p.browse.Clicked(gtx)
			button := material.Button(th.Theme, &p.browse, "Browse")
			dims := button.Layout(gtx)
			if clicked && onBrowse != nil {
				onBrowse()
			}
			return dims
		}),
	)
}

// path returns the current value of the field.
func (p *pathField) path() string {
	return strings.TrimSpace(p.editor.Text())
}

// setPath replaces the value of the field.
func (p *pathField) setPath(path string) {
	p.editor.SetText(path)
}

// exclusionsField holds the two exclusion inputs of a page.
type exclusionsField struct {
	files textField
	dirs  textField
}

// layout draws both exclusion inputs.
func (e *exclusionsField) layout(gtx layout.Context, th *theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return formRow(gtx, th, "Exclude files", func(gtx layout.Context) layout.Dimensions {
				return e.files.layout(gtx, th, "*.log,*.tmp")
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Spacer{Height: unit.Dp(6)}.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return formRow(gtx, th, "Exclude dirs", func(gtx layout.Context) layout.Dimensions {
				return e.dirs.layout(gtx, th, "node_modules,*.git")
			})
		}),
	)
}

// filePattern returns the file exclusion patterns of the field.
func (e *exclusionsField) filePattern() string {
	return e.files.value()
}

// dirPattern returns the directory exclusion patterns of the field.
func (e *exclusionsField) dirPattern() string {
	return e.dirs.value()
}

// statTile is a single number with a caption.
type statTile struct {
	label   string
	value   string
	caption string
}

// statTiles lays out a row of summary numbers.
func statTiles(gtx layout.Context, th *theme, tiles []statTile) layout.Dimensions {
	children := make([]layout.FlexChild, 0, len(tiles))
	for _, tile := range tiles {
		tile := tile
		children = append(children, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return backdrop(gtx, panelBackground(th), cornerRadius, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(14)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								label := material.Caption(th.Theme, tile.label)
								return label.Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								value := material.H4(th.Theme, tile.value)
								return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return value.Layout(gtx)
								})
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								if tile.caption == "" {
									return layout.Dimensions{}
								}
								caption := material.Caption(th.Theme, tile.caption)
								return caption.Layout(gtx)
							}),
						)
					})
				})
			})
		}))
	}

	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, children...)
}

// tableColumn describes one column of a table.
type tableColumn struct {
	title string
	width unit.Dp
}

// table renders rows of text in a scrollable list with a fixed header.
//
// The table must be kept across frames, because the list records its scroll
// position internally.
type table struct {
	list    widget.List
	columns []tableColumn
	rows    [][]string
	empty   string
}

// newTable returns a table with a vertical list.
func newTable() table {
	list := widget.List{}
	list.Axis = layout.Vertical
	return table{list: list}
}

// setRows replaces the rows of the table.
func (t *table) setRows(rows [][]string) {
	t.rows = rows
}

// layout draws the table.
func (t *table) layout(gtx layout.Context, th *theme) layout.Dimensions {
	if len(t.rows) == 0 {
		label := material.Body2(th.Theme, t.empty)
		return layout.Inset{Top: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return label.Layout(gtx)
		})
	}

	list := material.List(th.Theme, &t.list)

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return t.layoutHeader(gtx, th)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Spacer{Height: unit.Dp(6)}.Layout(gtx)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return list.Layout(gtx, len(t.rows), func(gtx layout.Context, i int) layout.Dimensions {
				return t.layoutRow(gtx, th, i)
			})
		}),
	)
}

// layoutHeader draws the column titles.
func (t *table) layoutHeader(gtx layout.Context, th *theme) layout.Dimensions {
	return t.layoutCells(gtx, func(gtx layout.Context, i int) layout.Dimensions {
		label := material.Caption(th.Theme, t.columns[i].title)
		label.MaxLines = 1
		return label.Layout(gtx)
	})
}

// layoutRow draws one row of the table.
func (t *table) layoutRow(gtx layout.Context, th *theme, index int) layout.Dimensions {
	return t.layoutCells(gtx, func(gtx layout.Context, i int) layout.Dimensions {
		text := ""
		if index < len(t.rows) && i < len(t.rows[index]) {
			text = t.rows[index][i]
		}
		label := material.Body2(th.Theme, text)
		label.MaxLines = 1
		return label.Layout(gtx)
	})
}

// layoutCells lays out one cell per column.
func (t *table) layoutCells(gtx layout.Context, cell func(gtx layout.Context, i int) layout.Dimensions) layout.Dimensions {
	children := make([]layout.FlexChild, 0, len(t.columns)+1)
	for i := range t.columns {
		i := i
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Right: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return fixedWidth(gtx, t.columns[i].width, func(gtx layout.Context) layout.Dimensions {
					return cell(gtx, i)
				})
			})
		}))
	}
	children = append(children, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: gtx.Constraints.Min}
	}))

	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, children...)
}

// barChartEntry is one bar of a bar chart.
type barChartEntry struct {
	label   string
	value   string
	percent float64
}

// barHeight is the height of a bar of the bar chart.
const barHeight = unit.Dp(8)

// barChart draws horizontal bars showing the relative size of each entry.
func barChart(gtx layout.Context, th *theme, entries []barChartEntry) layout.Dimensions {
	if len(entries) == 0 {
		return layout.Dimensions{}
	}

	var children []layout.FlexChild
	for _, entry := range entries {
		entry := entry
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								label := material.Caption(th.Theme, entry.label)
								label.MaxLines = 1
								return fixedWidth(gtx, unit.Dp(110), func(gtx layout.Context) layout.Dimensions {
									return label.Layout(gtx)
								})
							}),
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								value := material.Caption(th.Theme, entry.value)
								value.MaxLines = 1
								value.Alignment = text.End
								return value.Layout(gtx)
							}),
						)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Spacer{Height: unit.Dp(4)}.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layoutBar(gtx, th, entry)
					}),
				)
			})
		}))
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

// layoutBar draws the bar of one entry.
//
// The bar is painted directly instead of using a layout, because its width
// is a fraction of the available space and its height is fixed.
func layoutBar(gtx layout.Context, th *theme, entry barChartEntry) layout.Dimensions {
	height := gtx.Metric.Dp(barHeight)
	width := gtx.Constraints.Max.X

	trackRadius := barHeight / 2
	paintBackgroundRect(gtx, th.palette.SurfaceAlt, trackRadius, image.Pt(width, height))

	fill := int(float32(width) * float32(entry.percent) / 100)
	if fill > width {
		fill = width
	}
	if fill > 0 {
		fillRadius := trackRadius
		if fill < height {
			fillRadius = unit.Dp(float32(fill) / 2 / gtx.Metric.PxPerDp)
		}
		paintBackgroundRect(gtx, th.Palette.ContrastBg, fillRadius, image.Pt(fill, height))
	}

	return layout.Dimensions{Size: image.Pt(width, height)}
}

// formatSize renders a size in bytes in a human readable form.
func formatSize(size int64) string {
	switch {
	case size < 1024:
		return fmt.Sprintf("%d B", size)
	case size < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	case size < 1024*1024*1024:
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	case size < 1024*1024*1024*1024:
		return fmt.Sprintf("%.1f GB", float64(size)/(1024*1024*1024))
	default:
		return fmt.Sprintf("%.1f TB", float64(size)/(1024*1024*1024*1024))
	}
}

// listDirectories returns the subdirectories of dir, sorted by name.
func listDirectories(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			dirs = append(dirs, entry.Name())
		}
	}
	sort.Strings(dirs)

	return dirs, nil
}

// parentDirectory returns the parent of dir.
func parentDirectory(dir string) string {
	parent := filepath.Dir(dir)
	if parent == dir {
		return dir
	}
	return parent
}

// dirPicker lets the user choose a directory.
//
// Gio has no native file dialog, so the picker lists the subdirectories of
// the current directory and lets the user navigate with it.
type dirPicker struct {
	visible bool

	// onChoose is called with the chosen directory when the user confirms
	// the selection.
	onChoose func(path string)

	path   pathField
	list   widget.List
	dirs   []string
	err    error
	upBtn  widget.Clickable
	okBtn  widget.Clickable
	cancel widget.Clickable
	// dirClicks holds the click state of the listed directories. It is
	// resized whenever the listing changes.
	dirClicks []widget.Clickable
}

// open shows the picker for the given directory.
//
// The picker always works with an absolute path, so that the user can walk
// up from the directory they started in.
func (d *dirPicker) open(dir string) {
	if dir == "" {
		if wd, err := os.Getwd(); err == nil {
			dir = wd
		}
	}
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}

	d.visible = true
	d.path.hideBrowse = true
	d.path.setPath(dir)
	d.list = widget.List{}
	d.list.Axis = layout.Vertical
	d.refresh()
}

// refresh reloads the listing of the current directory.
func (d *dirPicker) refresh() {
	dir := d.path.path()
	if dir == "" {
		dir = "."
	}

	d.dirs, d.err = listDirectories(dir)
}

// layout draws the picker as an overlay.
func (d *dirPicker) layout(gtx layout.Context, th *theme) layout.Dimensions {
	if !d.visible {
		return layout.Dimensions{}
	}

	return layout.Stack{}.Layout(gtx,
		// The dimmed background.
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			paintBackground(gtx, color.NRGBA{A: 0x99, R: 0, G: 0, B: 0}, 0)
			return layout.Dimensions{Size: gtx.Constraints.Min}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				maxWidth := gtx.Metric.Dp(520)
				if gtx.Constraints.Max.X < maxWidth {
					maxWidth = gtx.Constraints.Max.X
				}
				minWidth := gtx.Metric.Dp(420)
				if gtx.Constraints.Min.X > minWidth {
					minWidth = gtx.Constraints.Min.X
				}
				gtx.Constraints.Max.X = maxWidth
				gtx.Constraints.Min.X = minWidth

				return backdrop(gtx, panelBackground(th), cornerRadius, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								title := material.Subtitle2(th.Theme, "Choose a directory")
								return layout.Inset{Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return title.Layout(gtx)
								})
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return d.path.layout(gtx, th, "/path/to/directory", nil)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Spacer{Height: unit.Dp(10)}.Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return d.layoutError(gtx, th)
							}),
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return d.layoutDirs(gtx, th)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Spacer{Height: unit.Dp(12)}.Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return d.layoutButtons(gtx, th)
							}),
						)
					})
				})
			})
		}),
	)
}

// layoutError shows the error of the last listing.
func (d *dirPicker) layoutError(gtx layout.Context, th *theme) layout.Dimensions {
	if d.err == nil {
		return layout.Dimensions{}
	}

	label := material.Body2(th.Theme, d.err.Error())
	label.Color = dangerColor(th)

	return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return label.Layout(gtx)
	})
}

// layoutDirs lists the subdirectories of the current directory. Clicking an
// entry navigates into it.
//
// The navigation is applied after the list is laid out, because the list
// keeps using the entries of the old listing while it draws the rest of
// them.
func (d *dirPicker) layoutDirs(gtx layout.Context, th *theme) layout.Dimensions {
	if len(d.dirClicks) != len(d.dirs) {
		d.dirClicks = make([]widget.Clickable, len(d.dirs))
	}

	list := material.List(th.Theme, &d.list)

	var next string
	dims := list.Layout(gtx, len(d.dirs), func(gtx layout.Context, i int) layout.Dimensions {
		name := d.dirs[i]
		click := &d.dirClicks[i]

		if click.Clicked(gtx) {
			next = filepath.Join(d.path.path(), name)
		}

		return layout.Inset{Bottom: unit.Dp(2)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return material.Clickable(gtx, click, func(gtx layout.Context) layout.Dimensions {
				entry := material.Body2(th.Theme, name)
				return entry.Layout(gtx)
			})
		})
	})

	if next != "" {
		d.path.setPath(next)
		d.refresh()
	}

	return dims
}

// layoutButtons draws the buttons of the picker.
func (d *dirPicker) layoutButtons(gtx layout.Context, th *theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if d.upBtn.Clicked(gtx) {
				d.path.setPath(parentDirectory(d.path.path()))
				d.refresh()
			}
			button := material.Button(th.Theme, &d.upBtn, "Up")
			return button.Layout(gtx)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: gtx.Constraints.Min}
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if d.cancel.Clicked(gtx) {
				d.visible = false
			}
			button := material.Button(th.Theme, &d.cancel, "Cancel")
			return layout.Inset{Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return button.Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if d.okBtn.Clicked(gtx) {
				d.visible = false
				if d.onChoose != nil {
					d.onChoose(d.path.path())
				}
			}
			button := material.Button(th.Theme, &d.okBtn, "Use this folder")
			return button.Layout(gtx)
		}),
	)
}

// confirmDialog asks the user to confirm a destructive action.
type confirmDialog struct {
	visible bool
	message string

	confirmBtn widget.Clickable
	cancelBtn  widget.Clickable

	// onConfirm is called when the user confirms the action.
	onConfirm func()
}

// ask opens the dialog with the given message.
func (c *confirmDialog) ask(message string, onConfirm func()) {
	c.visible = true
	c.message = message
	c.onConfirm = onConfirm
}

// layout draws the dialog as an overlay.
func (c *confirmDialog) layout(gtx layout.Context, th *theme) layout.Dimensions {
	if !c.visible {
		return layout.Dimensions{}
	}

	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			paintBackground(gtx, color.NRGBA{A: 0x99, R: 0, G: 0, B: 0}, 0)
			return layout.Dimensions{Size: gtx.Constraints.Min}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				maxWidth := gtx.Metric.Dp(460)
				if gtx.Constraints.Max.X < maxWidth {
					maxWidth = gtx.Constraints.Max.X
				}
				gtx.Constraints.Max.X = maxWidth
				gtx.Constraints.Min.X = 0

				return backdrop(gtx, panelBackground(th), cornerRadius, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(18)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								title := material.Subtitle2(th.Theme, "Are you sure?")
								return layout.Inset{Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return title.Layout(gtx)
								})
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								label := material.Body2(th.Theme, c.message)
								label.MaxLines = 3
								return label.Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Spacer{Height: unit.Dp(18)}.Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return c.layoutButtons(gtx, th)
							}),
						)
					})
				})
			})
		}),
	)
}

// layoutButtons draws the buttons of the dialog.
func (c *confirmDialog) layoutButtons(gtx layout.Context, th *theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: gtx.Constraints.Min}
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if c.cancelBtn.Clicked(gtx) {
				c.visible = false
			}
			button := material.Button(th.Theme, &c.cancelBtn, "Cancel")
			return layout.Inset{Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return button.Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if c.confirmBtn.Clicked(gtx) {
				c.visible = false
				if c.onConfirm != nil {
					c.onConfirm()
				}
			}
			button := material.Button(th.Theme, &c.confirmBtn, "Apply")
			button.Color = dangerColor(th)
			return button.Layout(gtx)
		}),
	)
}

// pageHeader draws the title of a page.
func pageHeader(gtx layout.Context, th *theme, title string) layout.Dimensions {
	label := material.H3(th.Theme, title)
	return label.Layout(gtx)
}

// emptyHint shows a hint in the results area when there is nothing to show
// yet.
func emptyHint(gtx layout.Context, th *theme, text string) layout.Dimensions {
	label := material.Body2(th.Theme, text)
	label.Color = th.palette.Muted
	return layout.Inset{Top: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return label.Layout(gtx)
	})
}

// layoutProgress draws the progress of a running job.
//
// The layout returns nothing when no job ran yet, so the pages can simply
// place it in their vertical stack. While a job is running, pause and stop
// buttons let the user freeze the engine or abort it. The buttons live on
// the job, so their click state survives across frames.
func layoutProgress(gtx layout.Context, th *theme, job *Job) layout.Dimensions {
	desc, current, total, running, done, err := job.state()
	if !running && !done {
		return layout.Dimensions{}
	}

	if err != nil {
		if errors.Is(err, progress.ErrCancelled) {
			label := material.Body2(th.Theme, "Stopped.")
			label.Color = th.palette.Muted
			return layout.Inset{Bottom: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return label.Layout(gtx)
			})
		}
		label := material.Body2(th.Theme, "Error: "+err.Error())
		label.Color = dangerColor(th)
		return layout.Inset{Bottom: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return label.Layout(gtx)
		})
	}

	progress := float32(0)
	text := desc
	paused := false
	switch {
	case running && total > 0:
		progress = float32(current) / float32(total)
		text = fmt.Sprintf("%s  %d/%d", desc, current, total)
	case running:
		text = fmt.Sprintf("%s  %d items", desc, current)
	case done:
		text = fmt.Sprintf("%s  done", desc)
	}
	if running && job.isPaused() {
		paused = true
		text += "  paused"
	}

	if !running {
		return layout.Inset{Bottom: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					bar := material.ProgressBar(th.Theme, progress)
					return bar.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Left: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						label := material.Caption(th.Theme, text)
						label.MaxLines = 1
						return label.Layout(gtx)
					})
				}),
			)
		})
	}

	// The clicks are read before the buttons are laid out, because laying
	// out a clickable consumes the pending click event.
	if job.stopBtn.Clicked(gtx) {
		job.Cancel()
	}
	if job.pauseBtn.Clicked(gtx) {
		job.TogglePause()
	}
	pauseLabel := "Pause"
	if paused {
		pauseLabel = "Resume"
	}

	return layout.Inset{Bottom: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						bar := material.ProgressBar(th.Theme, progress)
						return bar.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Left: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							label := material.Caption(th.Theme, text)
							label.MaxLines = 1
							return label.Layout(gtx)
						})
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Spacer{Height: unit.Dp(8)}.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						button := material.Button(th.Theme, &job.pauseBtn, pauseLabel)
						button.TextSize = th.TextSize
						return button.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							button := material.Button(th.Theme, &job.stopBtn, "Stop")
							button.TextSize = th.TextSize
							return button.Layout(gtx)
						})
					}),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Dimensions{Size: gtx.Constraints.Min}
					}),
				)
			}),
		)
	})
}

// dangerColor returns the color used for error messages.
func dangerColor(th *theme) color.NRGBA {
	return th.palette.Danger
}

// itoa renders an integer.
func itoa(v int) string {
	return strconv.Itoa(v)
}
