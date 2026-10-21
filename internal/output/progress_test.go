package output

import (
	"io"
	"os"
	"strings"
	"testing"
)

// captureStderr redirects os.Stderr for the duration of fn and returns what was written.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}

	original := os.Stderr
	os.Stderr = w

	fn()

	os.Stderr = original
	w.Close()

	output, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("Failed to read captured stderr: %v", err)
	}
	r.Close()

	return string(output)
}

func TestNoOpProgressIndicator(t *testing.T) {
	p := &NoOpProgressIndicator{}

	if p.IsEnabled() {
		t.Error("Expected no-op indicator to be disabled")
	}

	output := captureStderr(t, func() {
		p.Start(10, "test")
		p.Increment()
		p.Update(5)
		p.Finish()
	})

	if output != "" {
		t.Errorf("Expected no output from no-op indicator, got '%s'", output)
	}
}

func TestSimpleProgressIndicator_Disabled(t *testing.T) {
	p := &SimpleProgressIndicator{enabled: false}

	if p.IsEnabled() {
		t.Error("Expected indicator to be disabled")
	}

	output := captureStderr(t, func() {
		p.Start(10, "test")
		p.Increment()
		p.Finish()
	})

	if output != "" {
		t.Errorf("Expected no output from disabled indicator, got '%s'", output)
	}
}

func TestSimpleProgressIndicator_KnownTotal(t *testing.T) {
	p := &SimpleProgressIndicator{enabled: true}

	output := captureStderr(t, func() {
		p.Start(10, "test")
		for i := 0; i < 5; i++ {
			p.Increment()
		}
		p.Finish()
	})

	if !strings.Contains(output, "test: 0/10 (0.0%)") {
		t.Errorf("Expected initial progress line, got '%s'", output)
	}
	if !strings.Contains(output, "test: 5/10 (50.0%)") {
		t.Errorf("Expected 50%% progress line, got '%s'", output)
	}
	if strings.Contains(output, "10/10 (100.0%)") {
		t.Errorf("Expected progress to reflect the real count, got '%s'", output)
	}
}

func TestSimpleProgressIndicator_FinishUsesCurrentCount(t *testing.T) {
	p := &SimpleProgressIndicator{enabled: true}

	output := captureStderr(t, func() {
		p.Start(10, "test")
		for i := 0; i < 3; i++ {
			p.Increment()
		}
		p.Finish()
	})

	// The final line must report the real progress, not a hardcoded 100%
	if !strings.Contains(output, "test: 3/10 (30.0%)") {
		t.Errorf("Expected final line to report 3/10 (30.0%%), got '%s'", output)
	}
	if strings.Contains(output, "10/10 (100.0%)") {
		t.Errorf("Expected no hardcoded 100%% line, got '%s'", output)
	}
}

func TestSimpleProgressIndicator_UnknownTotal(t *testing.T) {
	p := &SimpleProgressIndicator{enabled: true}

	output := captureStderr(t, func() {
		p.Start(-1, "test")
		for i := 0; i < 3; i++ {
			p.Increment()
		}
		p.Finish()
	})

	if !strings.Contains(output, "test: ...") {
		t.Errorf("Expected unknown-total start line, got '%s'", output)
	}
	if !strings.Contains(output, "test: 3 items processed") {
		t.Errorf("Expected final count line, got '%s'", output)
	}
}

func TestSimpleProgressIndicator_Update(t *testing.T) {
	p := &SimpleProgressIndicator{enabled: true}

	output := captureStderr(t, func() {
		p.Start(10, "test")
		p.Update(4)
		p.Finish()
	})

	if !strings.Contains(output, "test: 4/10 (40.0%)") {
		t.Errorf("Expected updated progress line, got '%s'", output)
	}
}

func TestIsTerminal_NonTerminal(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "notatty")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer f.Close()

	if isTerminal(f) {
		t.Error("Expected regular file not to be detected as a terminal")
	}
}

func TestIsTerminal_NilFile(t *testing.T) {
	if isTerminal(nil) {
		t.Error("Expected nil file not to be detected as a terminal")
	}
}
