//go:build gui

package gui

import (
	"time"

	"gioui.org/layout"
	"gioui.org/op"
)

// previewDelay is how long the inputs of a page must stay unchanged before
// its preview is recomputed.
const previewDelay = 400 * time.Millisecond

// previewDebouncer debounces the automatic refresh of a preview.
//
// The preview of a page is recomputed when its inputs change. Typing must
// not start a scan per keystroke, so the refresh waits until the inputs
// have been quiet for a moment.
type previewDebouncer struct {
	// key holds the inputs the current preview was computed from.
	key string
	// pending holds the inputs waiting to be previewed.
	pending string
	// since is the time the pending inputs were last changed.
	since time.Time
}

// refresh reports whether the preview of the given inputs should run now.
//
// When it should not, a new frame is scheduled for the end of the debounce
// delay, so the preview starts once the inputs have been quiet long enough.
func (d *previewDebouncer) refresh(gtx layout.Context, key string) bool {
	if key == d.key {
		return false
	}

	if key != d.pending {
		d.pending = key
		d.since = time.Now()
		gtx.Execute(op.InvalidateCmd{At: d.since.Add(previewDelay)})
		return false
	}

	if time.Since(d.since) < previewDelay {
		return false
	}

	d.key = key
	d.pending = ""

	return true
}

// invalidate marks the preview as stale, so that it is recomputed.
func (d *previewDebouncer) invalidate() {
	d.key = ""
	d.pending = ""
}
