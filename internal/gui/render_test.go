//go:build gui

package gui

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"

	"gioui.org/gpu/headless"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"amurru/filetools/internal/report"
)

// render lays out the given content offscreen and returns the rendered
// image.
//
// The test is skipped when the machine has no offscreen renderer, which
// happens when no EGL implementation is available.
func render(t *testing.T, width, height int, build func(gtx layout.Context, th *theme)) *image.RGBA {
	t.Helper()

	win, err := headless.NewWindow(width, height)
	if err != nil {
		t.Skipf("no offscreen renderer available: %v", err)
	}
	defer win.Release()

	var ops op.Ops
	gtx := layout.Context{
		Ops:         &ops,
		Constraints: layout.Exact(image.Pt(width, height)),
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
	}

	build(gtx, newTheme(lightPalette))

	if err := win.Frame(&ops); err != nil {
		t.Fatalf("rendering failed: %v", err)
	}

	img := image.NewRGBA(image.Rectangle{Max: image.Pt(width, height)})
	if err := win.Screenshot(img); err != nil {
		t.Fatalf("screenshot failed: %v", err)
	}

	return img
}

// isBlank reports whether every pixel of the image has the same color.
func isBlank(img *image.RGBA) bool {
	if img.Bounds().Empty() {
		return true
	}

	first := img.RGBAAt(img.Bounds().Min.X, img.Bounds().Min.Y)
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			if img.RGBAAt(x, y) != first {
				return false
			}
		}
	}
	return true
}

func TestRenderApp(t *testing.T) {
	img := render(t, 1100, 700, func(gtx layout.Context, th *theme) {
		a := newApp(nil)
		a.layout(gtx)
	})

	if isBlank(img) {
		t.Error("Expected the interface to draw something")
	}
}

func TestRenderPages(t *testing.T) {
	pages := []struct {
		name string
		page page
	}{
		{"dupfind", newDupfindPage(nil)},
		{"dirstat", newDirStatPage(nil)},
		{"size", newSizePage(nil)},
		{"rename", newRenamePage(nil)},
		{"organize", newOrganizePage(nil)},
	}

	for _, p := range pages {
		t.Run(p.name, func(t *testing.T) {
			img := render(t, 1100, 700, func(gtx layout.Context, th *theme) {
				p.page.layout(gtx, th)
			})

			if isBlank(img) {
				t.Error("Expected the page to draw something")
			}
		})
	}
}

func TestRenderWidgets(t *testing.T) {
	t.Run("table", func(t *testing.T) {
		tbl := table{
			columns: []tableColumn{
				{title: "NAME", width: unit.Dp(200)},
				{title: "SIZE", width: unit.Dp(80)},
			},
			rows: [][]string{
				{"a.txt", "12 B"},
				{"b.txt", "34 B"},
			},
		}

		img := render(t, 400, 200, func(gtx layout.Context, th *theme) {
			tbl.layout(gtx, th)
		})
		if isBlank(img) {
			t.Error("Expected the table to draw something")
		}
	})

	t.Run("bar chart", func(t *testing.T) {
		entries := []barChartEntry{
			{label: ".txt", value: "60%", percent: 60},
			{label: ".go", value: "40%", percent: 40},
		}

		img := render(t, 400, 200, func(gtx layout.Context, th *theme) {
			barChart(gtx, th, entries)
		})
		if isBlank(img) {
			t.Error("Expected the bar chart to draw something")
		}
	})

	t.Run("tiles", func(t *testing.T) {
		tiles := []statTile{
			{label: "FILES", value: "42"},
			{label: "SIZE", value: "1.2 MB"},
		}

		img := render(t, 400, 120, func(gtx layout.Context, th *theme) {
			statTiles(gtx, th, tiles)
		})
		if isBlank(img) {
			t.Error("Expected the tiles to draw something")
		}
	})
}

func TestRenderNavHighlight(t *testing.T) {
	img := render(t, 1180, 780, func(gtx layout.Context, th *theme) {
		a := newApp(nil)
		a.layout(gtx)
	})

	// The active entry is highlighted with the alternative surface color.
	highlight := color.NRGBAModel.Convert(lightPalette.SurfaceAlt).(color.NRGBA)

	found := false
	for y := 0; y < 400 && !found; y++ {
		run := 0
		for x := 0; x < int(sidebarWidth); x++ {
			c := color.NRGBAModel.Convert(img.RGBAAt(x, y)).(color.NRGBA)
			if c != highlight {
				run = 0
				continue
			}
			run++
			// The background of the entry covers the whole width of the
			// navigation.
			if run >= int(sidebarWidth)/2 {
				found = true
				break
			}
		}
	}

	if !found {
		t.Error("Expected the active navigation entry to be highlighted")
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		size     int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
	}

	for _, test := range tests {
		if got := formatSize(test.size); got != test.expected {
			t.Errorf("formatSize(%d) = %s, want %s", test.size, got, test.expected)
		}
	}
}

// TestRenderScreenshot writes a screenshot of the interface to the file
// named by the FILETOOLS_GUI_SCREENSHOT environment variable.
//
// It is a development aid for reviewing the layout and is skipped when the
// variable is not set:
//
//	FILETOOLS_GUI_SCREENSHOT=/tmp/gui.png go test -tags gui -run Screenshot ./internal/gui/
func TestRenderScreenshot(t *testing.T) {
	path := os.Getenv("FILETOOLS_GUI_SCREENSHOT")
	if path == "" {
		t.Skip("set FILETOOLS_GUI_SCREENSHOT to write a screenshot")
	}

	img := render(t, 1180, 780, func(gtx layout.Context, th *theme) {
		a := newApp(nil)
		a.layout(gtx)
	})

	writePNG(t, path, img)
}

// TestRenderResultsScreenshot writes a screenshot of a page showing results,
// so that the tables and the charts can be reviewed.
func TestRenderResultsScreenshot(t *testing.T) {
	path := os.Getenv("FILETOOLS_GUI_RESULTS_SCREENSHOT")
	if path == "" {
		t.Skip("set FILETOOLS_GUI_RESULTS_SCREENSHOT to write a screenshot")
	}

	img := render(t, 1180, 780, func(gtx layout.Context, th *theme) {
		a := newApp(nil)
		a.active = 1
		p := a.items[1].page.(*dirStatPage)
		p.job.result = &report.DirStatResult{
			TotalFiles:  1284,
			TotalSize:   3 * 1024 * 1024 * 1024,
			LargestFile: &report.FileInfo{Name: "movie.mp4", Size: 900 * 1024 * 1024, Path: "videos/movie.mp4"},
			FileTypes: []report.FileType{
				{Extension: ".mp4", Count: 12, TotalSize: 1800 * 1024 * 1024, Percentage: 56},
				{Extension: ".jpg", Count: 840, TotalSize: 700 * 1024 * 1024, Percentage: 22},
				{Extension: ".go", Count: 210, TotalSize: 320 * 1024 * 1024, Percentage: 10},
				{Extension: ".txt", Count: 222, TotalSize: 200 * 1024 * 1024, Percentage: 7},
				{Extension: ".log", Count: 40, TotalSize: 100 * 1024 * 1024, Percentage: 5},
			},
			Directories: []report.DirectoryInfo{
				{Path: "videos", FileCount: 12, TotalSize: 1800 * 1024 * 1024, Percentage: 56},
				{Path: "photos/2024", FileCount: 840, TotalSize: 700 * 1024 * 1024, Percentage: 22},
				{Path: "src", FileCount: 210, TotalSize: 320 * 1024 * 1024, Percentage: 10},
				{Path: "documents", FileCount: 222, TotalSize: 200 * 1024 * 1024, Percentage: 7},
			},
		}
		a.layout(gtx)
	})

	writePNG(t, path, img)
}

// writePNG writes the image to the given path.
func writePNG(t *testing.T, path string, img *image.RGBA) {
	t.Helper()

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("creating the screenshot failed: %v", err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		t.Fatalf("writing the screenshot failed: %v", err)
	}
}
