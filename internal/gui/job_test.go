//go:build gui

package gui

import (
	"errors"
	"sync"
	"testing"
	"time"

	"amurru/filetools/internal/progress"
	"amurru/filetools/internal/report"
	"amurru/filetools/internal/scan"
)

// waitFor polls the job until the predicate holds or the timeout expires.
func waitFor(t *testing.T, job *Job, predicate func() bool) bool {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if predicate() {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}

	return false
}

func TestJobRunsInTheBackground(t *testing.T) {
	job := newJob(nil)

	job.Run(func(p progress.Indicator) (any, error) {
		p.Start(3, "working")
		for i := 0; i < 3; i++ {
			p.Increment()
		}
		p.Finish()
		return 42, nil
	})

	if !waitFor(t, job, func() bool { return job.output() != nil }) {
		t.Fatal("Expected the job to finish")
	}

	if got := job.output(); got != 42 {
		t.Errorf("Expected the result 42, got %v", got)
	}

	_, _, _, running, done, err := job.state()
	if running {
		t.Error("Expected the job to be finished")
	}
	if !done {
		t.Error("Expected the job to be marked as done")
	}
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestJobReportsTheError(t *testing.T) {
	job := newJob(nil)

	failure := errors.New("the scan failed")

	job.Run(func(p progress.Indicator) (any, error) {
		return nil, failure
	})

	if !waitFor(t, job, func() bool {
		_, _, _, _, _, err := job.state()
		return err != nil
	}) {
		t.Fatal("Expected the job to report the error")
	}

	_, _, _, _, _, err := job.state()
	if !errors.Is(err, failure) {
		t.Errorf("Expected the error to be reported, got %v", err)
	}
}

func TestJobIgnoresASecondRun(t *testing.T) {
	job := newJob(nil)

	var wg sync.WaitGroup
	wg.Add(1)

	job.Run(func(p progress.Indicator) (any, error) {
		defer wg.Done()
		p.Start(1, "working")
		time.Sleep(50 * time.Millisecond)
		return "first", nil
	})

	// The second run must be ignored while the first one is in progress.
	job.Run(func(p progress.Indicator) (any, error) {
		return "second", nil
	})

	wg.Wait()

	if !waitFor(t, job, func() bool { return job.output() != nil }) {
		t.Fatal("Expected the job to finish")
	}

	if got := job.output(); got != "first" {
		t.Errorf("Expected the first run to win, got %v", got)
	}
}

func TestJobRunsTheAnalysisEngine(t *testing.T) {
	dir := t.TempDir()

	job := newJob(nil)
	job.Run(func(p progress.Indicator) (any, error) {
		return scan.AnalyzeDirStat(scan.DirStatOptions{Dir: dir}, p)
	})

	if !waitFor(t, job, func() bool { return job.output() != nil }) {
		t.Fatal("Expected the analysis to finish")
	}

	result, ok := job.output().(*report.DirStatResult)
	if !ok {
		t.Fatalf("Expected a directory statistics result, got %T", job.output())
	}
	if result.TotalFiles != 0 {
		t.Errorf("Expected an empty directory, got %d files", result.TotalFiles)
	}
}
