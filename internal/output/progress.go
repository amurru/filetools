package output

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// ProgressIndicator defines the interface for progress indicators
type ProgressIndicator interface {
	Start(total int64, description string)
	Increment()
	Update(current int64)
	Finish()
	IsEnabled() bool
}

// NoOpProgressIndicator provides a no-operation progress indicator
type NoOpProgressIndicator struct{}

func (n *NoOpProgressIndicator) Start(total int64, description string) {}
func (n *NoOpProgressIndicator) Increment()                            {}
func (n *NoOpProgressIndicator) Update(current int64)                  {}
func (n *NoOpProgressIndicator) Finish()                               {}
func (n *NoOpProgressIndicator) IsEnabled() bool                       { return false }

// SimpleProgressIndicator provides a basic text-based progress indicator
type SimpleProgressIndicator struct {
	description string
	current     int64
	total       int64
	enabled     bool
	lastShown   int64
}

func (s *SimpleProgressIndicator) Start(total int64, description string) {
	s.total = total
	s.description = description
	s.current = 0
	s.lastShown = 0
	if s.enabled && s.total > 0 {
		fmt.Fprintf(os.Stderr, "%s: 0/%d (0.0%%)\n", description, total)
	} else if s.enabled {
		// Unknown total - just show description
		fmt.Fprintf(os.Stderr, "%s: ...\n", description)
	}
}

func (s *SimpleProgressIndicator) Increment() {
	if !s.enabled {
		return
	}
	s.current++
	s.updateProgress()
}

func (s *SimpleProgressIndicator) Update(current int64) {
	if !s.enabled {
		return
	}
	s.current = current
	s.updateProgress()
}

func (s *SimpleProgressIndicator) Finish() {
	if s.enabled && s.total > 0 {
		percentage := float64(s.total) / float64(s.total) * 100
		fmt.Fprintf(os.Stderr, "%s: %d/%d (%.1f%%)\n", s.description, s.total, s.total, percentage)
	} else if s.enabled {
		// Unknown total - show final count
		fmt.Fprintf(os.Stderr, "%s: %d items processed\n", s.description, s.current)
	}
}

func (s *SimpleProgressIndicator) IsEnabled() bool {
	return s.enabled
}

func (s *SimpleProgressIndicator) updateProgress() {
	// Handle unknown total
	if s.total <= 0 {
		updateInterval := int64(100)
		if s.current-s.lastShown >= updateInterval {
			fmt.Fprintf(os.Stderr, "%s: %d processed...\n", s.description, s.current)
			s.lastShown = s.current
		}
		return
	}

	// Update progress display based on total size
	var updateInterval int64
	if s.total <= 10 {
		updateInterval = 1 // Show every item for very small totals
	} else if s.total <= 100 {
		updateInterval = 10 // Show every 10 items for small totals
	} else if s.total <= 1000 {
		updateInterval = 50 // Show every 50 items for medium totals
	} else if s.total <= 10000 {
		updateInterval = 500 // Show every 500 items for large totals
	} else {
		updateInterval = 1000 // Show every 1000 items for very large totals
	}

	if s.current-s.lastShown >= updateInterval || s.current == s.total {
		percentage := float64(s.current) / float64(s.total) * 100
		fmt.Fprintf(os.Stderr, "%s: %d/%d (%.1f%%)\n", s.description, s.current, s.total, percentage)
		s.lastShown = s.current
	}
}

// NewProgressIndicator creates a new progress indicator based on command context
// For now, this is a simple implementation. We'll integrate the actual progressbar library later.
func NewProgressIndicator(cmd *cobra.Command) ProgressIndicator {
	// Check if progress bars should be disabled
	noProgressFlag, _ := cmd.Flags().GetBool("no-progress")

	// Check if output is going to a file (not stdout)
	outputFile, _ := cmd.Flags().GetString("file")
	outputToFile := outputFile != ""

	// Check if output format is structured
	outputFormat := getOutputFormatFromCmd(cmd)
	structuredOutput := outputFormat != FormatText

	// Only disable progress for structured file output (to avoid breaking parsers)
	// Allow progress for text file output since it's human-readable
	fileOutputStructured := outputToFile && structuredOutput

	// Disable progress bars if stderr is not a terminal
	enabled := !noProgressFlag && !fileOutputStructured && isTerminal(os.Stderr)

	if !enabled {
		return &NoOpProgressIndicator{}
	}

	return &SimpleProgressIndicator{
		enabled: true,
	}
}

// Helper function to check if file descriptor is a terminal
func isTerminal(_ *os.File) bool {
	// Simple check - in a real implementation we'd use unix.Isatty
	// For now, assume it's a terminal unless we know otherwise
	return true
}

// Helper function to get output format from command (similar to existing getOutputFormat)
func getOutputFormatFromCmd(cmd *cobra.Command) OutputFormat {
	// Check shortcut flags first (higher priority)
	if jsonFlag, _ := cmd.Flags().GetBool("json"); jsonFlag {
		return FormatJSON
	}
	if xmlFlag, _ := cmd.Flags().GetBool("xml"); xmlFlag {
		return FormatXML
	}
	if htmlFlag, _ := cmd.Flags().GetBool("html"); htmlFlag {
		return FormatHTML
	}

	// Fall back to the output flag
	outputFormat, _ := cmd.Flags().GetString("output")
	switch outputFormat {
	case "json":
		return FormatJSON
	case "xml":
		return FormatXML
	case "html":
		return FormatHTML
	case "text":
		fallthrough
	default:
		return FormatText
	}
}
