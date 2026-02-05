package tools

import (
	"fmt"
	"strings"

	"github.com/ledongthuc/pdf"
	"github.com/xuri/excelize/v2"
)

// parsePDF reads a PDF file and extracts its text content.
func parsePDF(path string) (string, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var sb strings.Builder
	totalPage := r.NumPage()

	for pageIndex := 1; pageIndex <= totalPage; pageIndex++ {
		p := r.Page(pageIndex)
		if p.V.IsNull() {
			continue
		}

		text, err := p.GetPlainText(nil)
		if err != nil {
			// If we fail to read a page, just skip or note it?
			// Let's note it but continue.
			fmt.Fprintf(&sb, "[Error reading page %d: %v]\n", pageIndex, err)
			continue
		}

		// Clean up text: replace excessive newlines
		text = cleanText(text)

		fmt.Fprintf(&sb, "--- Page %d ---\n%s\n\n", pageIndex, text)
	}

	return sb.String(), nil
}

// cleanText removes excessive whitespace and artifacts common in PDF extraction
func cleanText(text string) string {
	// PDF extraction often results in disjointed lines.
	// We'll keep it simple: normalize line breaks.
	// Convert Windows line endings
	text = strings.ReplaceAll(text, "\r\n", "\n")
	// Remove null bytes
	text = strings.ReplaceAll(text, "\x00", "")
	return strings.TrimSpace(text)
}

// parseExcel reads an Excel file and converts sheets to Markdown tables.
func parseExcel(path string) (string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var sb strings.Builder
	sheets := f.GetSheetList()

	for _, sheet := range sheets {
		rows, err := f.GetRows(sheet)
		if err != nil {
			fmt.Fprintf(&sb, "--- Sheet: %s (Error: %v) ---\n\n", sheet, err)
			continue
		}

		if len(rows) == 0 {
			fmt.Fprintf(&sb, "--- Sheet: %s (Empty) ---\n\n", sheet)
			continue
		}

		fmt.Fprintf(&sb, "--- Sheet: %s ---\n", sheet)

		// Limit number of rows to avoid context overflow?
		// Let's assume the tool wrapper handles truncation if needed,
		// but providing a reasonable limit here is good practice for "View".
		// We'll process all for now, relying on file_read limit logic if we integrate it there?
		// Actually file_read has line limits. We should probably produce the full string
		// and let the caller truncate, OR we incorporate limits here.
		// For now, let's render the table.

		sb.WriteString(rowsToMarkdown(rows))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// rowsToMarkdown converts a slice of string slices into a Markdown table
func rowsToMarkdown(rows [][]string) string {
	if len(rows) == 0 {
		return ""
	}

	var sb strings.Builder

	// Calculate max columns to ensure alignment
	maxCols := 0
	for _, row := range rows {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}

	// 1. Header row
	header := rows[0]
	// Pad header if necessary
	for len(header) < maxCols {
		header = append(header, "")
	}

	sb.WriteString("| " + strings.Join(header, " | ") + " |\n")

	// 2. Separator row
	sb.WriteString("|")
	for i := 0; i < maxCols; i++ {
		sb.WriteString(" --- |")
	}
	sb.WriteString("\n")

	// 3. Data rows
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		// Pad row if necessary
		for len(row) < maxCols {
			row = append(row, "")
		}

		// Escape pipes in content to prevent breaking table
		for j := range row {
			row[j] = strings.ReplaceAll(row[j], "|", "\\|")
			row[j] = strings.ReplaceAll(row[j], "\n", " ") // flatten newlines
		}

		sb.WriteString("| " + strings.Join(row, " | ") + " |\n")
	}

	return sb.String()
}
