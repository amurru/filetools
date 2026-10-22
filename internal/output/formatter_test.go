package output

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/amurru/filetools/internal/report"
	"strings"
	"testing"
)

func createTestResult() *report.DuplicateResult {
	return &report.DuplicateResult{
		Found: true,
		Groups: []report.DuplicateGroup{
			{
				Hash:  "abc123def456",
				Size:  1024,
				Files: []string{"/path/to/file1.txt", "/path/to/file2.txt"},
			},
			{
				Hash:  "xyz789abc123",
				Size:  2048,
				Files: []string{"/path/to/file3.txt", "/path/to/file4.txt", "/path/to/file5.txt"},
			},
		},
	}
}

func createEmptyResult() *report.DuplicateResult {
	return &report.DuplicateResult{
		Found:  false,
		Groups: []report.DuplicateGroup{},
	}
}

func TestTextFormatter_FormatDuplicates(t *testing.T) {
	result := createTestResult()
	formatter := &TextFormatter{}

	var buf bytes.Buffer
	err := formatter.FormatDuplicates(result, &buf)
	if err != nil {
		t.Fatalf("FormatDuplicates failed: %v", err)
	}

	output := buf.String()

	// Check that output contains expected elements
	if !strings.Contains(output, "Duplicate files found:") {
		t.Error("Expected 'Duplicate files found:' in output")
	}
	if !strings.Contains(output, "abc123de...") {
		t.Error("Expected truncated hash 'abc123de...' in output")
	}
	if !strings.Contains(output, "1024 bytes") {
		t.Error("Expected file size '1024 bytes' in output")
	}
	if !strings.Contains(output, "/path/to/file1.txt") {
		t.Error("Expected file path in output")
	}
}

func TestTextFormatter_FormatDuplicates_NoDuplicates(t *testing.T) {
	result := createEmptyResult()
	formatter := &TextFormatter{}

	var buf bytes.Buffer
	err := formatter.FormatDuplicates(result, &buf)
	if err != nil {
		t.Fatalf("FormatDuplicates failed: %v", err)
	}

	output := buf.String()
	expected := "No duplicate files found.\n"

	if output != expected {
		t.Errorf("Expected '%s', got '%s'", expected, output)
	}
}

func TestJSONFormatter_FormatDuplicates(t *testing.T) {
	result := createTestResult()
	formatter := &JSONFormatter{}

	var buf bytes.Buffer
	err := formatter.FormatDuplicates(result, &buf)
	if err != nil {
		t.Fatalf("FormatDuplicates failed: %v", err)
	}

	output := buf.String()

	// Parse the JSON to verify structure
	var parsed report.DuplicateResult
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if parsed.Found != result.Found {
		t.Errorf("Expected Found=%v, got Found=%v", result.Found, parsed.Found)
	}

	if len(parsed.Groups) != len(result.Groups) {
		t.Errorf("Expected %d groups, got %d", len(result.Groups), len(parsed.Groups))
	}

	if parsed.Groups[0].Hash != result.Groups[0].Hash {
		t.Errorf("Expected hash %s, got %s", result.Groups[0].Hash, parsed.Groups[0].Hash)
	}
}

func TestXMLFormatter_FormatDuplicates(t *testing.T) {
	result := createTestResult()
	formatter := &XMLFormatter{}

	var buf bytes.Buffer
	err := formatter.FormatDuplicates(result, &buf)
	if err != nil {
		t.Fatalf("FormatDuplicates failed: %v", err)
	}

	output := buf.String()

	// Check XML structure
	if !strings.Contains(output, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Error("Expected XML header in output")
	}
	if !strings.Contains(output, "<DuplicateResult>") {
		t.Error("Expected DuplicateResult root element")
	}
	if !strings.Contains(output, "<hash>abc123def456</hash>") {
		t.Error("Expected hash element in output")
	}
	if !strings.Contains(output, "<size>1024</size>") {
		t.Error("Expected size element in output")
	}

	// Parse the XML to verify structure
	var parsed report.DuplicateResult
	if err := xml.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("Failed to parse XML output: %v", err)
	}

	if parsed.Found != result.Found {
		t.Errorf("Expected Found=%v, got Found=%v", result.Found, parsed.Found)
	}
}

func TestHTMLFormatter_FormatDuplicates(t *testing.T) {
	result := createTestResult()
	formatter := &HTMLFormatter{}

	var buf bytes.Buffer
	err := formatter.FormatDuplicates(result, &buf)
	if err != nil {
		t.Fatalf("FormatDuplicates failed: %v", err)
	}

	output := buf.String()

	// Check HTML structure
	if !strings.Contains(output, "<!DOCTYPE html>") {
		t.Error("Expected HTML doctype")
	}
	if !strings.Contains(output, "<title>Duplicate Files Report</title>") {
		t.Error("Expected HTML title")
	}
	if !strings.Contains(output, "Duplicate Files Report") {
		t.Error("Expected page heading")
	}
	if !strings.Contains(output, "abc123def456") {
		t.Error("Expected hash in HTML output")
	}
	if !strings.Contains(output, "1024 bytes") {
		t.Error("Expected file size in HTML output")
	}
}

func TestHTMLFormatter_FormatDuplicates_NoDuplicates(t *testing.T) {
	result := createEmptyResult()
	formatter := &HTMLFormatter{}

	var buf bytes.Buffer
	err := formatter.FormatDuplicates(result, &buf)
	if err != nil {
		t.Fatalf("FormatDuplicates failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "No duplicate files found.") {
		t.Error("Expected 'No duplicate files found.' message in HTML")
	}
}

func TestNewFormatter(t *testing.T) {
	tests := []struct {
		format   OutputFormat
		expected string
	}{
		{FormatText, "*output.TextFormatter"},
		{FormatJSON, "*output.JSONFormatter"},
		{FormatXML, "*output.XMLFormatter"},
		{FormatHTML, "*output.HTMLFormatter"},
	}

	for _, test := range tests {
		formatter := NewFormatter(test.format)
		actual := fmt.Sprintf("%T", formatter)
		if actual != test.expected {
			t.Errorf("NewFormatter(%s) = %s, expected %s", test.format, actual, test.expected)
		}
	}
}

func TestNewFormatter_Default(t *testing.T) {
	formatter := NewFormatter("invalid")
	actual := fmt.Sprintf("%T", formatter)
	expected := "*output.TextFormatter"

	if actual != expected {
		t.Errorf("NewFormatter(invalid) = %s, expected %s", actual, expected)
	}
}

func createTestOrganizeResult() *report.OrganizeResult {
	return &report.OrganizeResult{
		DryRun: true,
		Operations: []report.OrganizeOperation{
			{OldPath: "photo.jpg", NewPath: "images/photo.jpg", Category: "images", Size: 2048},
			{OldPath: "notes.txt", NewPath: "documents/notes.txt", Category: "documents", Size: 512, Error: "target file already exists"},
		},
		Categories: []report.OrganizeCategory{
			{Category: "documents", FileCount: 1, TotalSize: 512},
			{Category: "images", FileCount: 1, TotalSize: 2048},
		},
	}
}

func TestTextFormatter_FormatOrganize(t *testing.T) {
	result := createTestOrganizeResult()
	formatter := &TextFormatter{}

	var buf bytes.Buffer
	if err := formatter.FormatOrganize(result, &buf); err != nil {
		t.Fatalf("FormatOrganize failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "DRY RUN") {
		t.Error("Expected dry run notice in output")
	}
	if !strings.Contains(output, "images/photo.jpg") {
		t.Error("Expected target path in output")
	}
	if !strings.Contains(output, "ERROR: target file already exists") {
		t.Error("Expected error message in output")
	}
}

func TestJSONFormatter_FormatOrganize(t *testing.T) {
	result := createTestOrganizeResult()
	formatter := &JSONFormatter{}

	var buf bytes.Buffer
	if err := formatter.FormatOrganize(result, &buf); err != nil {
		t.Fatalf("FormatOrganize failed: %v", err)
	}

	var parsed report.OrganizeResult
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if !parsed.DryRun {
		t.Error("Expected DryRun to be true")
	}
	if len(parsed.Operations) != 2 {
		t.Errorf("Expected 2 operations, got %d", len(parsed.Operations))
	}
	if len(parsed.Categories) != 2 {
		t.Errorf("Expected 2 categories, got %d", len(parsed.Categories))
	}
	if parsed.Operations[0].Category != "images" {
		t.Errorf("Expected category 'images', got '%s'", parsed.Operations[0].Category)
	}
}

func TestXMLFormatter_FormatOrganize(t *testing.T) {
	result := createTestOrganizeResult()
	formatter := &XMLFormatter{}

	var buf bytes.Buffer
	if err := formatter.FormatOrganize(result, &buf); err != nil {
		t.Fatalf("FormatOrganize failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Error("Expected XML header in output")
	}
	if !strings.Contains(output, "<OrganizeResult>") {
		t.Error("Expected OrganizeResult root element")
	}

	var parsed report.OrganizeResult
	if err := xml.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("Failed to parse XML output: %v", err)
	}
	if len(parsed.Operations) != 2 {
		t.Errorf("Expected 2 operations, got %d", len(parsed.Operations))
	}
}

func TestHTMLFormatter_FormatOrganize(t *testing.T) {
	result := createTestOrganizeResult()
	formatter := &HTMLFormatter{}

	var buf bytes.Buffer
	if err := formatter.FormatOrganize(result, &buf); err != nil {
		t.Fatalf("FormatOrganize failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "<!DOCTYPE html>") {
		t.Error("Expected HTML doctype")
	}
	if !strings.Contains(output, "<title>File Organizer Report</title>") {
		t.Error("Expected HTML title")
	}
	if !strings.Contains(output, "images/photo.jpg") {
		t.Error("Expected target path in HTML output")
	}
}

func createTestSizeResult() *report.SizeReportResult {
	return &report.SizeReportResult{
		TotalFiles: 42,
		TotalSize:  1024 * 1024,
		TotalDirs:  7,
		LargestFiles: []report.FileInfo{
			{Name: "big.bin", Size: 512 * 1024, Path: "sub/big.bin"},
		},
		Buckets: []report.SizeBucket{
			{Label: "< 1 KB", FileCount: 40, TotalSize: 1024, Percentage: 0.1},
			{Label: "> 100 MB", FileCount: 2, TotalSize: 1024 * 1024, Percentage: 99.9},
		},
	}
}

func TestTextFormatter_FormatSizeReport(t *testing.T) {
	result := createTestSizeResult()
	formatter := &TextFormatter{}

	var buf bytes.Buffer
	if err := formatter.FormatSizeReport(result, &buf); err != nil {
		t.Fatalf("FormatSizeReport failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "Size Analysis") {
		t.Error("Expected heading in output")
	}
	if !strings.Contains(output, "Total Files: 42") {
		t.Error("Expected file count in output")
	}
	if !strings.Contains(output, "sub/big.bin") {
		t.Error("Expected largest file in output")
	}
}

func TestJSONFormatter_FormatSizeReport(t *testing.T) {
	result := createTestSizeResult()
	formatter := &JSONFormatter{}

	var buf bytes.Buffer
	if err := formatter.FormatSizeReport(result, &buf); err != nil {
		t.Fatalf("FormatSizeReport failed: %v", err)
	}

	var parsed report.SizeReportResult
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if parsed.TotalFiles != 42 {
		t.Errorf("Expected 42 files, got %d", parsed.TotalFiles)
	}
	if len(parsed.Buckets) != 2 {
		t.Errorf("Expected 2 buckets, got %d", len(parsed.Buckets))
	}
	if len(parsed.LargestFiles) != 1 {
		t.Errorf("Expected 1 largest file, got %d", len(parsed.LargestFiles))
	}
}

func TestXMLFormatter_FormatSizeReport(t *testing.T) {
	result := createTestSizeResult()
	formatter := &XMLFormatter{}

	var buf bytes.Buffer
	if err := formatter.FormatSizeReport(result, &buf); err != nil {
		t.Fatalf("FormatSizeReport failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "<SizeReportResult>") {
		t.Error("Expected SizeReportResult root element")
	}

	var parsed report.SizeReportResult
	if err := xml.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("Failed to parse XML output: %v", err)
	}
	if parsed.TotalDirs != 7 {
		t.Errorf("Expected 7 directories, got %d", parsed.TotalDirs)
	}
}

func TestHTMLFormatter_FormatSizeReport(t *testing.T) {
	result := createTestSizeResult()
	formatter := &HTMLFormatter{}

	var buf bytes.Buffer
	if err := formatter.FormatSizeReport(result, &buf); err != nil {
		t.Fatalf("FormatSizeReport failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "<!DOCTYPE html>") {
		t.Error("Expected HTML doctype")
	}
	if !strings.Contains(output, "<title>Size Analysis Report</title>") {
		t.Error("Expected HTML title")
	}
	if !strings.Contains(output, "sub/big.bin") {
		t.Error("Expected largest file in HTML output")
	}
}
