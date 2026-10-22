//go:build gui

package gui

import (
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"amurru/filetools/internal/version"
)

// sidebarWidth is the width of the navigation sidebar.
const sidebarWidth = unit.Dp(236)

// cornerRadius is the radius used for the rounded panels of the interface.
const cornerRadius = 8

// page is one tool of the graphical interface.
type page interface {
	// title returns the name of the page shown in the navigation.
	title() string
	// layout draws the page.
	layout(gtx layout.Context, th *theme) layout.Dimensions
}

// navItem is one entry of the navigation sidebar.
type navItem struct {
	title string
	click widget.Clickable
	page  page
}

// App holds the state of the graphical interface.
type App struct {
	win   *app.Window
	theme *theme

	palette Palette
	items   []navItem
	active  int

	// themeClick toggles between the light and the dark theme.
	themeClick widget.Clickable
}

// Run opens the graphical interface and blocks until the window is closed.
func Run() error {
	go func() {
		win := new(app.Window)
		win.Option(
			app.Title("filetools"),
			app.Size(unit.Dp(1180), unit.Dp(780)),
			app.MinSize(unit.Dp(920), unit.Dp(620)),
		)

		if err := loop(win); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()

	app.Main()
	return nil
}

// loop runs the event loop of the window.
func loop(win *app.Window) error {
	a := newApp(win)

	var ops op.Ops
	for {
		switch e := win.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			ops.Reset()
			gtx := app.NewContext(&ops, e)
			a.layout(gtx)
			e.Frame(gtx.Ops)
		}
	}
}

// newApp builds the interface with all its pages.
func newApp(win *app.Window) *App {
	a := &App{
		win:     win,
		palette: lightPalette,
	}
	a.theme = newTheme(a.palette)

	pages := []page{
		newDupfindPage(win),
		newDirStatPage(win),
		newSizePage(win),
		newRenamePage(win),
		newOrganizePage(win),
	}
	for _, p := range pages {
		a.items = append(a.items, navItem{title: p.title(), page: p})
	}

	return a
}

// layout draws the whole interface.
func (a *App) layout(gtx layout.Context) layout.Dimensions {
	return layout.Stack{}.Layout(gtx,
		// The background of the window.
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			paintBackground(gtx, a.palette.Surface, 0)
			return layout.Dimensions{Size: gtx.Constraints.Min}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					// The sidebar has a fixed width.
					gtx.Constraints.Min.X = gtx.Metric.Dp(sidebarWidth)
					gtx.Constraints.Max.X = gtx.Constraints.Min.X
					return a.layoutSidebar(gtx)
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return a.layoutMain(gtx)
				}),
			)
		}),
	)
}

// layoutSidebar draws the navigation.
func (a *App) layoutSidebar(gtx layout.Context) layout.Dimensions {
	return layout.Stack{}.Layout(gtx,
		// The background of the sidebar.
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			paintBackground(gtx, a.palette.Sidebar, 0)
			return layout.Dimensions{Size: gtx.Constraints.Min}
		}), layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.layoutBrand(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Spacer{Height: unit.Dp(16)}.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.layoutNav(gtx)
					}),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Dimensions{Size: gtx.Constraints.Min}
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.layoutFooter(gtx)
					}),
				)
			})
		}),
	)
}

// layoutBrand draws the application name at the top of the sidebar.
func (a *App) layoutBrand(gtx layout.Context) layout.Dimensions {
	label := material.H4(a.theme.Theme, "filetools")
	label.Color = a.palette.Accent

	return layout.Inset{Top: 8, Bottom: 4}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return label.Layout(gtx)
	})
}

// layoutNav draws the navigation entries.
func (a *App) layoutNav(gtx layout.Context) layout.Dimensions {
	var children []layout.FlexChild
	for i := range a.items {
		item := &a.items[i]
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.layoutNavItem(gtx, item, i == a.active)
		}))
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

// layoutNavItem draws one navigation entry.
func (a *App) layoutNavItem(gtx layout.Context, item *navItem, active bool) layout.Dimensions {
	return material.Clickable(gtx, &item.click, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: 7, Bottom: 7, Left: 10, Right: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			if active {
				paintBackground(gtx, a.palette.SurfaceAlt, cornerRadius)
			}

			label := material.Body2(a.theme.Theme, item.title)
			if active {
				label.Color = a.palette.Accent
			}

			return label.Layout(gtx)
		})
	})
}

// layoutFooter draws the theme switch and the version at the bottom of the
// sidebar.
func (a *App) layoutFooter(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if a.themeClick.Clicked(gtx) {
				if a.palette.Name == lightPalette.Name {
					a.palette = darkPalette
				} else {
					a.palette = lightPalette
				}
				a.theme = newTheme(a.palette)
			}

			button := material.Button(a.theme.Theme, &a.themeClick, a.palette.Name+" theme")
			button.TextSize = a.theme.TextSize

			return button.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Spacer{Height: unit.Dp(8)}.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			label := material.Caption(a.theme.Theme, "v"+version.String())
			label.Color = a.palette.Muted
			return label.Layout(gtx)
		}),
	)
}

// layoutMain draws the active page.
func (a *App) layoutMain(gtx layout.Context) layout.Dimensions {
	for i := range a.items {
		item := &a.items[i]
		if item.click.Clicked(gtx) {
			a.active = i
		}
	}

	return layout.UniformInset(unit.Dp(20)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return a.items[a.active].page.layout(gtx, a.theme)
	})
}
