// Package output renders analysis results in the supported output formats.
//
// The formatters in this package translate the data model defined in
// internal/report into text, JSON, XML or HTML. They contain no analysis
// logic and no knowledge of the command line interface.
package output

import (
	"io"

	"amurru/filetools/internal/report"
)

// OutputFormatter defines the interface for different output formats
type OutputFormatter interface {
	FormatDuplicates(result *report.DuplicateResult, writer io.Writer) error
	FormatDirStat(result *report.DirStatResult, writer io.Writer) error
	FormatRename(result *report.RenameResult, writer io.Writer) error
	FormatOrganize(result *report.OrganizeResult, writer io.Writer) error
	FormatSizeReport(result *report.SizeReportResult, writer io.Writer) error
}

// OutputFormat represents the supported output formats
type OutputFormat string

const (
	FormatText OutputFormat = "text"
	FormatJSON OutputFormat = "json"
	FormatXML  OutputFormat = "xml"
	FormatHTML OutputFormat = "html"
)

// NewFormatter creates a new formatter based on the specified format
func NewFormatter(format OutputFormat) OutputFormatter {
	switch format {
	case FormatJSON:
		return &JSONFormatter{}
	case FormatXML:
		return &XMLFormatter{}
	case FormatHTML:
		return &HTMLFormatter{}
	case FormatText:
		fallthrough
	default:
		return &TextFormatter{}
	}
}
