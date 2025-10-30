package output

import (
	"fmt"
	"html"
	"io"
	"sort"
	"strings"
)

// HTMLFormatter implements the OutputFormatter interface for HTML output
type HTMLFormatter struct{}

// FormatDuplicates formats duplicate results as HTML
func (f *HTMLFormatter) FormatDuplicates(result *DuplicateResult, writer io.Writer) error {
	htmlContent := f.generateHTML(result)
	_, err := writer.Write([]byte(htmlContent))
	return err
}

// FormatRename formats rename results as HTML
func (f *HTMLFormatter) FormatRename(result *RenameResult, writer io.Writer) error {
	htmlContent := f.generateHTMLRename(result)
	_, err := writer.Write([]byte(htmlContent))
	return err
}

// generateHTML creates the complete HTML document
func (f *HTMLFormatter) generateHTML(result *DuplicateResult) string {
	var sb strings.Builder

	sb.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Duplicate Files Report</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            padding: 30px;
        }
        h1 {
            color: #333;
            border-bottom: 2px solid #007acc;
            padding-bottom: 10px;
        }
        .no-duplicates {
            text-align: center;
            color: #666;
            font-size: 18px;
            padding: 40px;
        }
        .duplicate-group {
            margin-bottom: 30px;
            border: 1px solid #ddd;
            border-radius: 6px;
            overflow: hidden;
        }
        .group-header {
            background: #f8f9fa;
            padding: 15px 20px;
            border-bottom: 1px solid #ddd;
        }
        .group-hash {
            font-family: monospace;
            color: #007acc;
            font-weight: bold;
        }
        .group-size {
            color: #666;
            font-size: 14px;
        }
        .file-list {
            padding: 0;
            margin: 0;
        }
        .file-item {
            padding: 12px 20px;
            border-bottom: 1px solid #eee;
            display: flex;
            align-items: center;
        }
        .file-item:last-child {
            border-bottom: none;
        }
        .file-name {
            font-family: monospace;
            flex: 1;
        }
        .file-badge {
            background: #28a745;
            color: white;
            padding: 2px 8px;
            border-radius: 12px;
            font-size: 12px;
            font-weight: bold;
        }
        .file-badge.original {
            background: #007acc;
        }
        .file-badge.duplicate {
            background: #dc3545;
        }
        .summary {
            background: #e9ecef;
            padding: 15px;
            border-radius: 6px;
            margin-bottom: 20px;
        }
        .summary-stats {
            display: flex;
            gap: 20px;
        }
        .stat {
            font-weight: bold;
            color: #007acc;
        }
        .duplicate-group.collapsed .file-list {
            display: none;
        }
        .group-header::before {
            content: '▼';
            margin-right: 8px;
            transition: transform 0.2s;
        }
        .duplicate-group.collapsed .group-header::before {
            transform: rotate(-90deg);
        }
        .group-hash:hover {
            background-color: #e9ecef;
            border-radius: 3px;
            padding: 2px 4px;
        }
        .exclusions-section {
            margin-top: 40px;
            border-top: 1px solid #ddd;
            padding-top: 20px;
        }
        .exclusions-section h2 {
            color: #333;
            margin-bottom: 20px;
        }
        .exclusions-table {
            width: 100%;
            border-collapse: collapse;
            margin-top: 10px;
        }
        .exclusions-table th,
        .exclusions-table td {
            border: 1px solid #ddd;
            padding: 8px 12px;
            text-align: left;
        }
        .exclusions-table th {
            background-color: #f8f9fa;
            font-weight: 600;
        }
        .exclusions-table tr:nth-child(even) {
            background-color: #f8f9fa;
        }
        .exclusions-table tr:hover {
            background-color: #e9ecef;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Duplicate Files Report</h1>`)

	if !result.Found {
		sb.WriteString(`
        <div class="no-duplicates">
            No duplicate files found.
        </div>`)
	} else {
		totalGroups := len(result.Groups)
		totalFiles := 0
		for _, group := range result.Groups {
			totalFiles += len(group.Files)
		}

		sb.WriteString(fmt.Sprintf(`
        <div class="summary">
            <div class="summary-stats">
                <span class="stat">%d duplicate groups found</span>
                <span class="stat">%d total duplicate files</span>
            </div>
        </div>`, totalGroups, totalFiles))

		for i, group := range result.Groups {
			sb.WriteString(f.generateGroupHTML(group, i+1))
		}
	}

	// Add exclusions section if any
	if len(result.Exclusions) > 0 {
		sb.WriteString(`
        <div class="exclusions-section">
            <h2>Excluded Files and Directories</h2>
            <table class="exclusions-table">
                <thead>
                    <tr>
                        <th>Path</th>
                        <th>Reason</th>
                    </tr>
                </thead>
                <tbody>`)

		for _, exclusion := range result.Exclusions {
			sb.WriteString(fmt.Sprintf(`
                    <tr>
                        <td>%s</td>
                        <td>%s</td>
                    </tr>`, html.EscapeString(exclusion.Path), html.EscapeString(exclusion.Reason)))
		}

		sb.WriteString(`
                </tbody>
            </table>
        </div>`)
	}

	sb.WriteString(`
    </div>`)

	// Add footer with branding
	if result.Metadata != nil {
		flags := []string{}
		for _, f := range result.Metadata.Flags {
			flags = append(flags, fmt.Sprintf("%s=%s", f.Name, f.Value))
		}
		flagStr := strings.Join(flags, ", ")
		sb.WriteString(fmt.Sprintf(`
    <footer style="text-align: center; margin-top: 40px; color: #666; font-size: 14px;">
        Generated by %s %s v%s on %s<br>
        Flags: %s
    </footer>`,
			html.EscapeString(result.Metadata.ToolName),
			html.EscapeString(result.Metadata.SubCommand),
			html.EscapeString(result.Metadata.Version),
			html.EscapeString(result.Metadata.GeneratedAt),
			html.EscapeString(flagStr)))
	}

	sb.WriteString(`
    <script>
        (function() {
            // Feature detection for clipboard API
            var supportsClipboard = navigator && navigator.clipboard && navigator.clipboard.writeText;

            // Fallback for older browsers
            function fallbackCopyText(text) {
                var textArea = document.createElement("textarea");
                textArea.value = text;
                document.body.appendChild(textArea);
                textArea.focus();
                textArea.select();
                try {
                    document.execCommand('copy');
                } catch (err) {
                    console.warn('Fallback copy failed');
                }
                document.body.removeChild(textArea);
            }

            // Copy hash functionality
            function copyHash(event) {
                var hash = event.target.getAttribute('data-full-hash');
                var original = event.target.textContent;
                if (supportsClipboard) {
                    navigator.clipboard.writeText(hash).then(function() {
                        event.target.textContent = 'Copied!';
                        setTimeout(function() {
                            event.target.textContent = original;
                        }, 1000);
                    });
                } else {
                    fallbackCopyText(hash);
                    event.target.textContent = 'Copied!';
                    setTimeout(function() {
                        event.target.textContent = original;
                    }, 1000);
                }
            }

            // Collapsible groups functionality
            function toggleGroup(event) {
                // Only toggle if clicking on header, not on hash
                if (event.target.classList.contains('group-hash')) return;

                var group = event.currentTarget.closest('.duplicate-group');
                var fileList = group.querySelector('.file-list');
                var isCollapsed = fileList.style.display === 'none';

                fileList.style.display = isCollapsed ? 'block' : 'none';
                group.classList.toggle('collapsed', !isCollapsed);
            }

            // Initialize when DOM is ready
            document.addEventListener('DOMContentLoaded', function() {
                // Add click handlers for hashes
                var hashes = document.querySelectorAll('.group-hash');
                for (var i = 0; i < hashes.length; i++) {
                    hashes[i].addEventListener('click', copyHash);
                    hashes[i].style.cursor = 'pointer';
                    hashes[i].title = 'Click to copy hash';
                }

                // Add click handlers for group headers
                var headers = document.querySelectorAll('.group-header');
                for (var i = 0; i < headers.length; i++) {
                    headers[i].addEventListener('click', toggleGroup);
                    headers[i].style.cursor = 'pointer';
                }
            });
        })();
    </script>
</body>
</html>`)

	return sb.String()
}

// generateGroupHTML creates HTML for a single duplicate group
func (f *HTMLFormatter) generateGroupHTML(group DuplicateGroup, _ int) string {
	var sb strings.Builder

	// Sort files alphabetically
	files := make([]string, len(group.Files))
	copy(files, group.Files)
	sort.Strings(files)

	sizeStr := "unknown size"
	if group.Size >= 0 {
		sizeStr = fmt.Sprintf("%d bytes", group.Size)
	}

	hashDisplay := group.Hash
	if len(hashDisplay) > 12 {
		hashDisplay = hashDisplay[:12] + "..."
	}

	sb.WriteString(fmt.Sprintf(`
        <div class="duplicate-group">
            <div class="group-header">
                <span class="group-hash" data-full-hash="%s">%s</span>
                <span class="group-size">(%s)</span>
            </div>
            <ul class="file-list">`, html.EscapeString(group.Hash), html.EscapeString(hashDisplay), html.EscapeString(sizeStr)))

	for j, file := range files {
		badgeClass := "duplicate"
		badgeText := "DUPLICATE"
		if j == 0 {
			badgeClass = "original"
			badgeText = "ORIGINAL"
		}

		sb.WriteString(fmt.Sprintf(`
                <li class="file-item">
                    <span class="file-name">%s</span>
                    <span class="file-badge %s">%s</span>
                </li>`, html.EscapeString(file), badgeClass, badgeText))
	}

	sb.WriteString(`
            </ul>
        </div>`)

	return sb.String()
}

// FormatDirStat formats directory statistics as HTML
func (f *HTMLFormatter) FormatDirStat(result *DirStatResult, writer io.Writer) error {
	htmlContent := f.generateDirStatHTML(result)
	_, err := writer.Write([]byte(htmlContent))
	return err
}

// generateDirStatHTML creates the complete HTML document for directory statistics
func (f *HTMLFormatter) generateDirStatHTML(result *DirStatResult) string {
	var sb strings.Builder

	sb.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Directory Statistics Report</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 1400px;
            margin: 0 auto;
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            padding: 30px;
        }
        h1 {
            color: #333;
            border-bottom: 2px solid #007acc;
            padding-bottom: 10px;
        }
        .summary {
            background: #e9ecef;
            padding: 20px;
            border-radius: 6px;
            margin-bottom: 30px;
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
        }
        .summary-item {
            text-align: center;
        }
        .summary-value {
            font-size: 24px;
            font-weight: bold;
            color: #007acc;
            display: block;
        }
        .summary-label {
            font-size: 14px;
            color: #666;
            margin-top: 5px;
        }
        .section {
            margin-bottom: 40px;
        }
        .section h2 {
            color: #333;
            border-bottom: 1px solid #ddd;
            padding-bottom: 8px;
            margin-bottom: 20px;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            margin-bottom: 20px;
            background: white;
            border-radius: 6px;
            overflow: hidden;
            box-shadow: 0 1px 3px rgba(0,0,0,0.1);
        }
        th, td {
            padding: 12px 15px;
            text-align: left;
            border-bottom: 1px solid #eee;
        }
        th {
            background: #f8f9fa;
            font-weight: 600;
            color: #333;
            position: sticky;
            top: 0;
        }
        tr:hover {
            background-color: #f8f9fa;
        }
        .size-col {
            text-align: right;
            font-family: monospace;
        }
        .count-col {
            text-align: right;
        }
        .percentage-col {
            text-align: right;
        }
        .percentage-bar {
            display: inline-block;
            height: 8px;
            background: #e9ecef;
            border-radius: 4px;
            width: 60px;
            margin-left: 10px;
            vertical-align: middle;
        }
        .percentage-fill {
            display: block;
            height: 100%;
            background: #007acc;
            border-radius: 4px;
        }
        .file-path {
            font-family: monospace;
            word-break: break-all;
        }
        .no-data {
            text-align: center;
            color: #666;
            padding: 40px;
            font-style: italic;
        }
        .sort-indicator {
            margin-left: 5px;
            opacity: 0.5;
        }
        .sort-indicator.active {
            opacity: 1;
        }
        .exclusions-section {
            margin-top: 40px;
            border-top: 1px solid #ddd;
            padding-top: 20px;
        }
        .exclusions-section h2 {
            color: #333;
            margin-bottom: 20px;
        }
        .exclusions-table {
            width: 100%;
            border-collapse: collapse;
            margin-top: 10px;
        }
        .exclusions-table th,
        .exclusions-table td {
            border: 1px solid #ddd;
            padding: 8px 12px;
            text-align: left;
        }
        .exclusions-table th {
            background-color: #f8f9fa;
            font-weight: 600;
        }
        .exclusions-table tr:nth-child(even) {
            background-color: #f8f9fa;
        }
        .exclusions-table tr:hover {
            background-color: #e9ecef;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Directory Statistics Report</h1>`)

	// Summary section
	sb.WriteString(`
        <div class="summary">`)
	sb.WriteString(fmt.Sprintf(`
            <div class="summary-item">
                <span class="summary-value">%d</span>
                <span class="summary-label">Total Files</span>
            </div>
            <div class="summary-item">
                <span class="summary-value">%s</span>
                <span class="summary-label">Total Size</span>
            </div>`, result.TotalFiles, formatSize(result.TotalSize)))

	if result.LargestFile != nil {
		sb.WriteString(fmt.Sprintf(`
            <div class="summary-item">
                <span class="summary-value">%s</span>
                <span class="summary-label">Largest File</span>
            </div>`, formatSize(result.LargestFile.Size)))
	}

	// Add exclusions section if any
	if len(result.Exclusions) > 0 {
		sb.WriteString(`
        <div class="exclusions-section">
            <h2>Excluded Files and Directories</h2>
            <table class="exclusions-table">
                <thead>
                    <tr>
                        <th>Path</th>
                        <th>Reason</th>
                    </tr>
                </thead>
                <tbody>`)

		for _, exclusion := range result.Exclusions {
			sb.WriteString(fmt.Sprintf(`
                    <tr>
                        <td>%s</td>
                        <td>%s</td>
                    </tr>`, html.EscapeString(exclusion.Path), html.EscapeString(exclusion.Reason)))
		}

		sb.WriteString(`
                </tbody>
            </table>
        </div>`)
	}

	sb.WriteString(`
    </div>`)

	// File types section
	if len(result.FileTypes) > 0 {
		sb.WriteString(`
        <div class="section">
            <h2>File Types</h2>
            <table id="file-types-table">
                <thead>
                    <tr>
                        <th>Extension</th>
                        <th class="count-col">Count</th>
                        <th class="size-col">Size</th>
                        <th class="percentage-col">Percentage</th>
                    </tr>
                </thead>
                <tbody>`)

		for _, ft := range result.FileTypes {
			percentage := ft.Percentage
			sb.WriteString(fmt.Sprintf(`
                    <tr>
                        <td>%s</td>
                        <td class="count-col">%d</td>
                        <td class="size-col">%s</td>
                        <td class="percentage-col">%.2f%%<span class="percentage-bar"><span class="percentage-fill" style="width: %.1f%%"></span></span></td>
                    </tr>`, html.EscapeString(ft.Extension), ft.Count, formatSize(ft.TotalSize), percentage, percentage))
		}

		sb.WriteString(`
                </tbody>
            </table>
        </div>`)
	}

	// Directories section
	if len(result.Directories) > 0 {
		sb.WriteString(`
        <div class="section">
            <h2>Subdirectories</h2>
            <table id="directories-table">
                <thead>
                    <tr>
                        <th>Path</th>
                        <th class="count-col">Files</th>
                        <th class="size-col">Size</th>
                        <th class="percentage-col">Percentage</th>
                    </tr>
                </thead>
                <tbody>`)

		for _, dir := range result.Directories {
			percentage := dir.Percentage
			sb.WriteString(fmt.Sprintf(`
                    <tr>
                        <td class="file-path">%s</td>
                        <td class="count-col">%d</td>
                        <td class="size-col">%s</td>
                        <td class="percentage-col">%.2f%%<span class="percentage-bar"><span class="percentage-fill" style="width: %.1f%%"></span></span></td>
                    </tr>`, html.EscapeString(dir.Path), dir.FileCount, formatSize(dir.TotalSize), percentage, percentage))
		}

		sb.WriteString(`
                </tbody>
            </table>
        </div>`)
	}

	sb.WriteString(`
    </div>`)

	// Add footer with branding
	if result.Metadata != nil {
		flags := []string{}
		for _, f := range result.Metadata.Flags {
			flags = append(flags, fmt.Sprintf("%s=%s", f.Name, f.Value))
		}
		flagStr := strings.Join(flags, ", ")
		sb.WriteString(fmt.Sprintf(`
    <footer style="text-align: center; margin-top: 40px; color: #666; font-size: 14px;">
        Generated by %s %s v%s on %s<br>
        Flags: %s
    </footer>`,
			html.EscapeString(result.Metadata.ToolName),
			html.EscapeString(result.Metadata.SubCommand),
			html.EscapeString(result.Metadata.Version),
			html.EscapeString(result.Metadata.GeneratedAt),
			html.EscapeString(flagStr)))
	}

	sb.WriteString(`
    <script>
        // Table sorting functionality
        function makeTableSortable(tableId) {
            var table = document.getElementById(tableId);
            if (!table) return;

            var headers = table.querySelectorAll('th');
            headers.forEach(function(header, index) {
                header.style.cursor = 'pointer';
                header.addEventListener('click', function() {
                    sortTable(table, index);
                });
            });
        }

        function sortTable(table, columnIndex) {
            var tbody = table.querySelector('tbody');
            var rows = Array.from(tbody.querySelectorAll('tr'));

            // Remove existing sort indicators
            table.querySelectorAll('.sort-indicator').forEach(function(indicator) {
                indicator.classList.remove('active');
            });

            // Determine sort direction
            var isNumeric = table.rows[0].cells[columnIndex].classList.contains('count-col') ||
                           table.rows[0].cells[columnIndex].classList.contains('size-col') ||
                           table.rows[0].cells[columnIndex].classList.contains('percentage-col');

            rows.sort(function(a, b) {
                var aVal = a.cells[columnIndex].textContent.trim();
                var bVal = b.cells[columnIndex].textContent.trim();

                if (isNumeric) {
                    // Extract numeric value (remove units like %, B, KB, etc.)
                    var aNum = parseFloat(aVal.replace(/[^\d.]/g, '')) || 0;
                    var bNum = parseFloat(bVal.replace(/[^\d.]/g, '')) || 0;
                    return bNum - aNum; // Descending for numeric
                } else {
                    return aVal.localeCompare(bVal);
                }
            });

            // Re-append sorted rows
            rows.forEach(function(row) {
                tbody.appendChild(row);
            });

            // Add sort indicator
            var header = table.querySelectorAll('th')[columnIndex];
            var indicator = header.querySelector('.sort-indicator') || document.createElement('span');
            indicator.className = 'sort-indicator active';
            indicator.textContent = '↓';
            header.appendChild(indicator);
        }

        // Initialize sortable tables
        document.addEventListener('DOMContentLoaded', function() {
            makeTableSortable('file-types-table');
            makeTableSortable('directories-table');
        });
    </script>
</body>
</html>`)

	return sb.String()
}

// generateHTMLRename creates the complete HTML document for rename results
func (f *HTMLFormatter) generateHTMLRename(result *RenameResult) string {
	var sb strings.Builder

	sb.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Rename Files Report</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            padding: 30px;
        }
        h1 {
            color: #333;
            border-bottom: 2px solid #007acc;
            padding-bottom: 10px;
        }
        .metadata {
            background: #f8f9fa;
            padding: 15px;
            border-radius: 5px;
            margin-bottom: 20px;
            font-size: 14px;
        }
        .dry-run {
            background: #fff3cd;
            color: #856404;
            padding: 10px;
            border-radius: 5px;
            margin-bottom: 20px;
            font-weight: bold;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            margin-top: 20px;
        }
        th, td {
            padding: 12px;
            text-align: left;
            border-bottom: 1px solid #ddd;
        }
        th {
            background-color: #007acc;
            color: white;
            cursor: pointer;
            position: relative;
        }
        th:hover {
            background-color: #0056b3;
        }
        .sort-indicator {
            margin-left: 5px;
            opacity: 0.7;
        }
        .sort-indicator.active {
            opacity: 1;
        }
        .error {
            color: #dc3545;
            font-weight: bold;
        }
        .footer {
            margin-top: 40px;
            padding-top: 20px;
            border-top: 1px solid #ddd;
            text-align: center;
            color: #666;
            font-size: 14px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Rename Files Report</h1>`)

	// Metadata
	if result.Metadata != nil {
		sb.WriteString("<div class=\"metadata\">")
		sb.WriteString(fmt.Sprintf("<strong>Generated by:</strong> %s %s v%s<br>",
			html.EscapeString(result.Metadata.ToolName),
			html.EscapeString(result.Metadata.SubCommand),
			html.EscapeString(result.Metadata.Version)))
		sb.WriteString(fmt.Sprintf("<strong>Generated at:</strong> %s<br>",
			html.EscapeString(result.Metadata.GeneratedAt)))

		if len(result.Metadata.Flags) > 0 {
			sb.WriteString("<strong>Flags:</strong> ")
			flags := []string{}
			for _, flag := range result.Metadata.Flags {
				flags = append(flags, fmt.Sprintf("%s=%s", html.EscapeString(flag.Name), html.EscapeString(flag.Value)))
			}
			sb.WriteString(strings.Join(flags, ", "))
		}
		sb.WriteString("</div>")
	}

	// Dry run notice
	if result.DryRun {
		sb.WriteString("<div class=\"dry-run\">DRY RUN - No files were actually renamed</div>")
	}

	// Operations table
	if len(result.Operations) > 0 {
		sb.WriteString(`
        <table id="operations-table">
            <thead>
                <tr>
                    <th>Old Path</th>
                    <th>New Path</th>
                    <th>Status</th>
                </tr>
            </thead>
            <tbody>`)

		for _, op := range result.Operations {
			status := "OK"
			statusClass := ""
			if op.Error != "" {
				status = fmt.Sprintf("ERROR: %s", html.EscapeString(op.Error))
				statusClass = "error"
			}

			sb.WriteString(fmt.Sprintf(`
                <tr>
                    <td>%s</td>
                    <td>%s</td>
                    <td class="%s">%s</td>
                </tr>`,
				html.EscapeString(op.OldPath),
				html.EscapeString(op.NewPath),
				statusClass,
				status))
		}

		sb.WriteString(`
            </tbody>
        </table>`)
	} else {
		sb.WriteString("<p>No files matched the pattern.</p>")
	}

	// Exclusions
	if len(result.Exclusions) > 0 {
		sb.WriteString(`
        <h2>Exclusions</h2>
        <table id="exclusions-table">
            <thead>
                <tr>
                    <th>Path</th>
                    <th>Reason</th>
                </tr>
            </thead>
            <tbody>`)

		for _, exclusion := range result.Exclusions {
			sb.WriteString(fmt.Sprintf(`
                <tr>
                    <td>%s</td>
                    <td>%s</td>
                </tr>`,
				html.EscapeString(exclusion.Path),
				html.EscapeString(exclusion.Reason)))
		}

		sb.WriteString(`
            </tbody>
        </table>`)
	}

	// Footer
	sb.WriteString(`
        <div class="footer">
            Generated by filetools
        </div>
    </div>
</body>
</html>`)

	return sb.String()
}
