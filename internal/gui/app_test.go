//go:build gui

package gui

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/io/semantic"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// testContext returns a context for a window of the given size.
func testContext(width, height int) layout.Context {
	return layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Exact(image.Pt(width, height)),
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
	}
}

// navEntryPosition returns the position of the navigation entry with the
// given title.
//
// The entries are looked up in the semantic tree of the frame, so that the
// test does not depend on the way the navigation is laid out.
func navEntryPosition(t *testing.T, r *input.Router, title string) f32.Point {
	t.Helper()

	for _, node := range r.AppendSemantics(nil) {
		if node.Desc.Class != semantic.Button {
			continue
		}
		// Only the entries of the navigation are on the left.
		if node.Desc.Bounds.Min.X >= int(sidebarWidth) {
			continue
		}

		for _, child := range node.Children {
			if child.Desc.Label != title {
				continue
			}

			bounds := node.Desc.Bounds
			return f32.Pt(
				float32(bounds.Min.X+bounds.Max.X)/2,
				float32(bounds.Min.Y+bounds.Max.Y)/2,
			)
		}
	}

	t.Fatalf("no navigation entry titled %q", title)
	return f32.Point{}
}

// TestNavClickSwitchesPage verifies that clicking a navigation entry
// activates its page.
func TestNavClickSwitchesPage(t *testing.T) {
	var r input.Router
	gtx := testContext(1180, 780)
	gtx.Source = r.Source()

	a := newApp(nil)

	// The first layout registers the clickable areas of the entries.
	a.layout(gtx)
	r.Frame(gtx.Ops)

	for n := range a.items {
		// The entries are clicked in turn, starting with an entry that is
		// not active, so that every click changes the page.
		i := (n + 1) % len(a.items)
		pos := navEntryPosition(t, &r, a.items[i].title)
		r.Queue(
			pointer.Event{
				Kind:      pointer.Press,
				Source:    pointer.Mouse,
				Buttons:   pointer.ButtonPrimary,
				Position:  pos,
				PointerID: 1,
			},
			pointer.Event{
				Kind:      pointer.Release,
				Source:    pointer.Mouse,
				Buttons:   pointer.ButtonPrimary,
				Position:  pos,
				PointerID: 1,
			},
		)

		gtx.Ops.Reset()
		a.layout(gtx)
		r.Frame(gtx.Ops)

		if a.active != i {
			t.Errorf("clicking %q activated the page %d, want %d", a.items[i].title, a.active, i)
		}
	}
}

// TestNavClickActivates verifies that a click on a navigation entry is
// reported, even when it is requested programmatically.
func TestNavClickActivates(t *testing.T) {
	gtx := testContext(1180, 780)

	a := newApp(nil)
	a.layout(gtx)

	for i := range a.items {
		a.items[i].click.Click()

		gtx.Ops.Reset()
		a.layout(gtx)

		if a.active != i {
			t.Errorf("clicking %q activated the page %d, want %d", a.items[i].title, a.active, i)
		}
	}
}
