// Package progress reports the progress of long running operations.
//
// The package defines the Indicator interface implemented by the text
// progress reporter used by the command line interface and by the job
// runner used by the graphical user interface. It deliberately knows nothing
// about either front end: the decision of whether progress is shown at all
// belongs to the caller.
package progress

import (
	"errors"
	"fmt"
	"io"
)

// ErrCancelled is returned by the analysis engines when the operation was
// stopped through the indicator.
var ErrCancelled = errors.New("operation cancelled")

// Cancellable is an optional extension of Indicator implemented by the
// graphical job runner. The analysis engines check it cooperatively, so
// that long running operations can be paused and stopped. The command line
// reporter does not implement it, where the helpers below are no-ops.
type Cancellable interface {
	// Cancelled reports whether the operation was stopped.
	Cancelled() bool
	// WaitIfPaused blocks while the operation is paused.
	WaitIfPaused()
}

// IsCancelled reports whether p was stopped.
func IsCancelled(p Indicator) bool {
	if c, ok := p.(Cancellable); ok {
		return c.Cancelled()
	}
	return false
}

// WaitIfPaused blocks while p is paused. It returns immediately for
// indicators that do not support pausing.
func WaitIfPaused(p Indicator) {
	if c, ok := p.(Cancellable); ok {
		c.WaitIfPaused()
	}
}

// Indicator reports the progress of a long running operation.
//
// Implementations must be safe for concurrent use because the analysis
// engines report progress from worker goroutines.
type Indicator interface {
	// Start begins reporting progress for an operation with the given total
	// number of steps. A negative total means the number of steps is
	// unknown.
	Start(total int64, description string)
	// Increment advances the progress by one step.
	Increment()
	// Update sets the progress to an absolute value.
	Update(current int64)
	// Finish completes the reporting.
	Finish()
	// IsEnabled reports whether the indicator produces any output.
	IsEnabled() bool
}

// Nop is an Indicator that reports nothing. It is used when progress output
// would interfere with the requested output format.
type Nop struct{}

// Start implements Indicator.
func (Nop) Start(total int64, description string) {}

// Increment implements Indicator.
func (Nop) Increment() {}

// Update implements Indicator.
func (Nop) Update(current int64) {}

// Finish implements Indicator.
func (Nop) Finish() {}

// IsEnabled implements Indicator.
func (Nop) IsEnabled() bool { return false }

// Text writes human readable progress lines to a writer.
type Text struct {
	writer      io.Writer
	description string
	current     int64
	total       int64
	lastShown   int64
}

// NewText returns a Text indicator writing to w.
func NewText(w io.Writer) *Text {
	return &Text{writer: w}
}

// Start implements Indicator.
func (t *Text) Start(total int64, description string) {
	t.total = total
	t.description = description
	t.current = 0
	t.lastShown = 0
	if t.total > 0 {
		fmt.Fprintf(t.writer, "%s: 0/%d (0.0%%)\n", description, total)
		return
	}
	// Unknown total, just show the description
	fmt.Fprintf(t.writer, "%s: ...\n", description)
}

// Increment implements Indicator.
func (t *Text) Increment() {
	t.current++
	t.updateProgress()
}

// Update implements Indicator.
func (t *Text) Update(current int64) {
	t.current = current
	t.updateProgress()
}

// Finish implements Indicator.
func (t *Text) Finish() {
	if t.total > 0 {
		// Skip the final line if the last update already showed this state
		if t.current == t.lastShown {
			return
		}
		percentage := float64(t.current) / float64(t.total) * 100
		fmt.Fprintf(t.writer, "%s: %d/%d (%.1f%%)\n", t.description, t.current, t.total, percentage)
		return
	}

	// Unknown total, show final count
	fmt.Fprintf(t.writer, "%s: %d items processed\n", t.description, t.current)
}

// IsEnabled implements Indicator.
func (t *Text) IsEnabled() bool { return true }

// updateProgress writes the current state when enough progress was made to
// be worth a line of output.
func (t *Text) updateProgress() {
	// Handle unknown total
	if t.total <= 0 {
		const updateInterval = 100
		if t.current-t.lastShown >= updateInterval {
			fmt.Fprintf(t.writer, "%s: %d processed...\n", t.description, t.current)
			t.lastShown = t.current
		}
		return
	}

	// Update progress display based on total size
	var updateInterval int64
	switch {
	case t.total <= 10:
		updateInterval = 1 // Show every item for very small totals
	case t.total <= 100:
		updateInterval = 10 // Show every 10 items for small totals
	case t.total <= 1000:
		updateInterval = 50 // Show every 50 items for medium totals
	case t.total <= 10000:
		updateInterval = 500 // Show every 500 items for large totals
	default:
		updateInterval = 1000 // Show every 1000 items for very large totals
	}

	if t.current-t.lastShown >= updateInterval || t.current == t.total {
		percentage := float64(t.current) / float64(t.total) * 100
		fmt.Fprintf(t.writer, "%s: %d/%d (%.1f%%)\n", t.description, t.current, t.total, percentage)
		t.lastShown = t.current
	}
}
