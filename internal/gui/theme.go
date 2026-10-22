//go:build gui

package gui

import (
	"image/color"

	"gioui.org/widget/material"
)

// Palette is the color scheme of the interface.
type Palette struct {
	// Name identifies the palette in the interface.
	Name string
	// Surface is the background of the window and of the navigation.
	Surface color.NRGBA
	// Sidebar is the background of the navigation.
	Sidebar color.NRGBA
	// Panel is the background of the cards and of the input fields.
	Panel color.NRGBA
	// SurfaceAlt is the background of the raised elements, such as the
	// selected navigation entry.
	SurfaceAlt color.NRGBA
	// Border separates the panels from the background.
	Border color.NRGBA
	// Accent is the color of the selected navigation entry and of the
	// primary action.
	Accent color.NRGBA
	// AccentText is the text color on top of Accent.
	AccentText color.NRGBA
	// Text is the color of the body text.
	Text color.NRGBA
	// Muted is the color of secondary text.
	Muted color.NRGBA
	// Danger marks destructive actions and errors.
	Danger color.NRGBA
}

// lightPalette is the palette used in the light theme.
var lightPalette = Palette{
	Name:       "Light",
	Surface:    rgb(0xf4f5f7),
	Sidebar:    rgb(0xe7eaef),
	Panel:      rgb(0xffffff),
	SurfaceAlt: rgb(0xd8dde5),
	Border:     rgb(0xd5d8de),
	Accent:     rgb(0x3f51b5),
	AccentText: rgb(0xffffff),
	Text:       rgb(0x1b1d21),
	Muted:      rgb(0x6b7280),
	Danger:     rgb(0xc62828),
}

// darkPalette is the palette used in the dark theme.
var darkPalette = Palette{
	Name:       "Dark",
	Surface:    rgb(0x1c1c1e),
	Sidebar:    rgb(0x141416),
	Panel:      rgb(0x2a2a2e),
	SurfaceAlt: rgb(0x3a3a42),
	Border:     rgb(0x3a3a40),
	Accent:     rgb(0x7f9bf0),
	AccentText: rgb(0x10131a),
	Text:       rgb(0xe8e8ea),
	Muted:      rgb(0x9aa0aa),
	Danger:     rgb(0xef5350),
}

// theme carries the Gio theme and the palette of the interface.
type theme struct {
	*material.Theme

	palette Palette
}

// newTheme builds the theme of the interface for the given palette.
func newTheme(p Palette) *theme {
	th := &theme{
		Theme:   material.NewTheme(),
		palette: p,
	}

	th.Palette = material.Palette{
		Bg:         p.Surface,
		Fg:         p.Text,
		ContrastBg: p.Accent,
		ContrastFg: p.AccentText,
	}

	return th
}

// rgb builds an opaque color from a 24 bit RGB value.
func rgb(c uint32) color.NRGBA {
	return color.NRGBA{A: 0xff, R: uint8(c >> 16), G: uint8(c >> 8), B: uint8(c)}
}
