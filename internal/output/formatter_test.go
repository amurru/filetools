package output

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
	"testing"
)

func createTestResult() *DuplicateResult {
	return &DuplicateResult{
		Found: true,
		Groups: []DuplicateGroup{
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

func createEmptyResult() *DuplicateResult {
	return &DuplicateResult{
		Found:  false,
		Groups: []DuplicateGroup{},
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
	var parsed DuplicateResult
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
	var parsed DuplicateResult
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
