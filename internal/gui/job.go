//go:build gui

package gui

import (
	"sync"
	"sync/atomic"
	"time"

	"gioui.org/app"

	"amurru/filetools/internal/progress"
)

// paintInterval is the minimum delay between two redraw requests caused by
// progress reporting. Without it a scan of a large tree would flood the
// event loop with repaint requests.
const paintInterval = 50 * time.Millisecond

// Job runs an analysis engine in the background and reports its progress.
//
// The engine runs in its own goroutine. The job stores the result and the
// progress under a lock and asks the window for a new frame, so the
// interface stays responsive while a scan is running. The Job implements
// progress.Indicator and is the bridge between the engines, which report
// their progress from worker goroutines, and the interface.
type Job struct {
	mu sync.Mutex

	win *app.Window

	desc    string
	total   int64
	running bool
	done    bool
	err     error

	// result holds the value returned by the engine.
	result any

	// current is updated by the engine and read by the interface, so it is
	// kept out of the mutex to keep the hot path cheap.
	current atomic.Int64

	// lastPaint is the time of the last repaint request.
	lastPaint time.Time
}

// newJob returns a job that reports to the given window.
func newJob(win *app.Window) *Job {
	return &Job{win: win}
}

// Start implements progress.Indicator.
func (j *Job) Start(total int64, description string) {
	j.mu.Lock()
	j.desc = description
	j.total = total
	j.mu.Unlock()

	j.current.Store(0)
	j.invalidate()
}

// Increment implements progress.Indicator.
func (j *Job) Increment() {
	j.current.Add(1)
	j.invalidateThrottled()
}

// Update implements progress.Indicator.
func (j *Job) Update(current int64) {
	j.current.Store(current)
	j.invalidateThrottled()
}

// Finish implements progress.Indicator.
func (j *Job) Finish() {
	j.invalidate()
}

// IsEnabled implements progress.Indicator.
func (j *Job) IsEnabled() bool { return true }

// Run starts fn in a background goroutine.
//
// The job stores the value returned by fn once it completes. A run that is
// already in progress is ignored, so a second click on the action button
// cannot start a second scan.
func (j *Job) Run(fn func(p progress.Indicator) (any, error)) {
	j.mu.Lock()
	if j.running {
		j.mu.Unlock()
		return
	}
	j.running = true
	j.done = false
	j.err = nil
	j.result = nil
	j.desc = ""
	j.total = 0
	j.mu.Unlock()

	j.current.Store(0)
	j.invalidate()

	go func() {
		result, err := fn(j)

		j.mu.Lock()
		j.result = result
		j.err = err
		j.running = false
		j.done = true
		j.mu.Unlock()

		j.invalidate()
	}()
}

// isRunning reports whether a scan is in progress.
func (j *Job) isRunning() bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.running
}

// state returns a snapshot of the progress of the job.
func (j *Job) state() (description string, current, total int64, running, done bool, err error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.desc, j.current.Load(), j.total, j.running, j.done, j.err
}

// output returns the value produced by the last completed run.
func (j *Job) output() any {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.result
}

// invalidate asks the window for a new frame.
func (j *Job) invalidate() {
	if j.win == nil {
		return
	}
	j.win.Invalidate()
}

// invalidateThrottled asks the window for a new frame, but no more often
// than once per paint interval.
func (j *Job) invalidateThrottled() {
	if j.win == nil {
		return
	}

	now := time.Now()

	j.mu.Lock()
	if now.Sub(j.lastPaint) < paintInterval {
		j.mu.Unlock()
		return
	}
	j.lastPaint = now
	j.mu.Unlock()

	j.win.Invalidate()
}
