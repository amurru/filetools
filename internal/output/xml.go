package output

import (
	"encoding/xml"
	"github.com/amurru/filetools/internal/report"
	"io"
)

// XMLFormatter implements the OutputFormatter interface for XML output
// XMLFormatter renders a report as XML.
type XMLFormatter struct{}

// FormatDuplicates formats duplicate results as XML
func (f *XMLFormatter) FormatDuplicates(result *report.DuplicateResult, writer io.Writer) error {
	encoder := xml.NewEncoder(writer)
	encoder.Indent("", "  ")

	// Write XML header
	if _, err := writer.Write([]byte(xml.Header)); err != nil {
		return err
	}

	// Encode the result
	if err := encoder.Encode(result); err != nil {
		return err
	}

	// Write a newline at the end
	_, err := writer.Write([]byte("\n"))
	return err
}

// FormatDirStat formats directory statistics as XML
func (f *XMLFormatter) FormatDirStat(result *report.DirStatResult, writer io.Writer) error {
	encoder := xml.NewEncoder(writer)
	encoder.Indent("", "  ")

	// Write XML header
	if _, err := writer.Write([]byte(xml.Header)); err != nil {
		return err
	}

	// Encode the result
	if err := encoder.Encode(result); err != nil {
		return err
	}

	// Write a newline at the end
	_, err := writer.Write([]byte("\n"))
	return err
}

// FormatRename formats rename results as XML
func (f *XMLFormatter) FormatRename(result *report.RenameResult, writer io.Writer) error {
	encoder := xml.NewEncoder(writer)
	encoder.Indent("", "  ")

	// Write XML header
	if _, err := writer.Write([]byte(xml.Header)); err != nil {
		return err
	}

	// Encode the result
	if err := encoder.Encode(result); err != nil {
		return err
	}

	// Write a newline at the end
	_, err := writer.Write([]byte("\n"))
	return err
}

// FormatOrganize formats organize results as XML
func (f *XMLFormatter) FormatOrganize(result *report.OrganizeResult, writer io.Writer) error {
	encoder := xml.NewEncoder(writer)
	encoder.Indent("", "  ")

	// Write XML header
	if _, err := writer.Write([]byte(xml.Header)); err != nil {
		return err
	}

	// Encode the result
	if err := encoder.Encode(result); err != nil {
		return err
	}

	// Write a newline at the end
	_, err := writer.Write([]byte("\n"))
	return err
}

// FormatSizeReport formats a size analysis as XML
func (f *XMLFormatter) FormatSizeReport(result *report.SizeReportResult, writer io.Writer) error {
	encoder := xml.NewEncoder(writer)
	encoder.Indent("", "  ")

	// Write XML header
	if _, err := writer.Write([]byte(xml.Header)); err != nil {
		return err
	}

	// Encode the result
	if err := encoder.Encode(result); err != nil {
		return err
	}

	// Write a newline at the end
	_, err := writer.Write([]byte("\n"))
	return err
}
