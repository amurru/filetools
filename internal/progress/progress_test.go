package progress

import (
	"bytes"
	"strings"
	"testing"
)

func TestNop(t *testing.T) {
	p := Nop{}

	if p.IsEnabled() {
		t.Error("Expected no-op indicator to be disabled")
	}

	var buf bytes.Buffer
	p.Start(10, "test")
	p.Increment()
	p.Update(5)
	p.Finish()

	if buf.Len() != 0 {
		t.Errorf("Expected no output from no-op indicator, got '%s'", buf.String())
	}
}

func TestTextKnownTotal(t *testing.T) {
	var buf bytes.Buffer
	p := NewText(&buf)

	p.Start(10, "test")
	for i := 0; i < 5; i++ {
		p.Increment()
	}
	p.Finish()

	out := buf.String()
	if !strings.Contains(out, "test: 0/10 (0.0%)") {
		t.Errorf("Expected initial progress line, got '%s'", out)
	}
	if !strings.Contains(out, "test: 5/10 (50.0%)") {
		t.Errorf("Expected 50%% progress line, got '%s'", out)
	}
	if strings.Contains(out, "10/10 (100.0%)") {
		t.Errorf("Expected progress to reflect the real count, got '%s'", out)
	}
}

func TestTextFinishUsesCurrentCount(t *testing.T) {
	var buf bytes.Buffer
	p := NewText(&buf)

	p.Start(10, "test")
	for i := 0; i < 3; i++ {
		p.Increment()
	}
	p.Finish()

	// The final line must report the real progress, not a hardcoded 100%
	out := buf.String()
	if !strings.Contains(out, "test: 3/10 (30.0%)") {
		t.Errorf("Expected final line to report 3/10 (30.0%%), got '%s'", out)
	}
	if strings.Contains(out, "10/10 (100.0%)") {
		t.Errorf("Expected no hardcoded 100%% line, got '%s'", out)
	}
}

func TestTextUnknownTotal(t *testing.T) {
	var buf bytes.Buffer
	p := NewText(&buf)

	p.Start(-1, "test")
	for i := 0; i < 3; i++ {
		p.Increment()
	}
	p.Finish()

	out := buf.String()
	if !strings.Contains(out, "test: ...") {
		t.Errorf("Expected unknown-total start line, got '%s'", out)
	}
	if !strings.Contains(out, "test: 3 items processed") {
		t.Errorf("Expected final count line, got '%s'", out)
	}
}

func TestTextUpdate(t *testing.T) {
	var buf bytes.Buffer
	p := NewText(&buf)

	p.Start(10, "test")
	p.Update(4)
	p.Finish()

	out := buf.String()
	if !strings.Contains(out, "test: 4/10 (40.0%)") {
		t.Errorf("Expected updated progress line, got '%s'", out)
	}
}

func TestTextIsEnabled(t *testing.T) {
	p := NewText(&bytes.Buffer{})
	if !p.IsEnabled() {
		t.Error("Expected the text indicator to be enabled")
	}
}
