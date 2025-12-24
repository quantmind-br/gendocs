package export

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewHTMLExporter_Success(t *testing.T) {
	exporter, err := NewHTMLExporter()
	if err != nil {
		t.Fatalf("Expected no error creating exporter, got %v", err)
	}

	if exporter == nil {
		t.Fatal("Expected exporter to be non-nil")
	}

	if exporter.markdown == nil {
		t.Error("Expected markdown renderer to be initialized")
	}

	if exporter.htmlTemplate == nil {
		t.Error("Expected HTML template to be initialized")
	}
}

func TestHTMLExporter_ExportToHTML_Success(t *testing.T) {
	exporter, err := NewHTMLExporter()
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	// Create temp directory and files
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "test.md")
	htmlFile := filepath.Join(tmpDir, "test.html")

	markdown := `# Test Document

This is a **test** with code:

` + "```go\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```" + `

## Section 2

- Item 1
- Item 2
- Item 3

### Subsection

| Column 1 | Column 2 |
|----------|----------|
| Value 1  | Value 2  |
| Value 3  | Value 4  |

> This is a blockquote

[Link text](https://example.com)
`

	err = os.WriteFile(mdFile, []byte(markdown), 0644)
	if err != nil {
		t.Fatalf("Failed to write test markdown: %v", err)
	}

	// Export to HTML
	err = exporter.ExportToHTML(mdFile, htmlFile)
	if err != nil {
		t.Fatalf("Expected no error exporting, got %v", err)
	}

	// Read generated HTML
	html, err := os.ReadFile(htmlFile)
	if err != nil {
		t.Fatalf("Failed to read generated HTML: %v", err)
	}

	htmlStr := string(html)

	// Verify HTML structure
	if !strings.Contains(htmlStr, "<!DOCTYPE html>") {
		t.Error("Expected DOCTYPE declaration")
	}

	if !strings.Contains(htmlStr, "<html lang=\"en\">") {
		t.Error("Expected html lang attribute")
	}

	if !strings.Contains(htmlStr, "<title>Test Document</title>") {
		t.Error("Expected title to be 'Test Document'")
	}

	// Verify content conversion
	if !strings.Contains(htmlStr, "<h1>Test Document</h1>") {
		t.Error("Expected H1 heading")
	}

	if !strings.Contains(htmlStr, "<h2>Section 2</h2>") {
		t.Error("Expected H2 heading")
	}

	if !strings.Contains(htmlStr, "<strong>test</strong>") {
		t.Error("Expected bold text")
	}

	if !strings.Contains(htmlStr, "<code") {
		t.Error("Expected code blocks")
	}

	// Check for code content (syntax highlighting splits into spans)
	if !strings.Contains(htmlStr, "func") || !strings.Contains(htmlStr, "main") {
		t.Error("Expected code content with 'func' and 'main'")
	}

	// Verify list
	if !strings.Contains(htmlStr, "<li>Item 1</li>") {
		t.Error("Expected list items")
	}

	// Verify table
	if !strings.Contains(htmlStr, "<table>") {
		t.Error("Expected table")
	}

	if !strings.Contains(htmlStr, "<th>Column 1</th>") {
		t.Error("Expected table headers")
	}

	if !strings.Contains(htmlStr, "<td>Value 1</td>") {
		t.Error("Expected table data")
	}

	// Verify blockquote
	if !strings.Contains(htmlStr, "<blockquote>") {
		t.Error("Expected blockquote")
	}

	// Verify link
	if !strings.Contains(htmlStr, "<a href=\"https://example.com\">Link text</a>") {
		t.Error("Expected link")
	}

	// Verify CSS is embedded
	if !strings.Contains(htmlStr, "font-family:") {
		t.Error("Expected embedded CSS")
	}

	// Verify footer
	if !strings.Contains(htmlStr, "Generated with Gendocs") {
		t.Error("Expected generator badge")
	}

	if !strings.Contains(htmlStr, "Generated on") {
		t.Error("Expected generation timestamp")
	}
}

func TestHTMLExporter_ExportToHTML_FileNotFound(t *testing.T) {
	exporter, err := NewHTMLExporter()
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "nonexistent.md")
	htmlFile := filepath.Join(tmpDir, "output.html")

	err = exporter.ExportToHTML(mdFile, htmlFile)
	if err == nil {
		t.Fatal("Expected error for nonexistent file, got nil")
	}

	if !strings.Contains(err.Error(), "failed to read markdown") {
		t.Errorf("Expected 'failed to read markdown' error, got: %v", err)
	}
}

func TestHTMLExporter_ExportToHTML_InvalidOutputPath(t *testing.T) {
	exporter, err := NewHTMLExporter()
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "test.md")

	markdown := "# Test\n\nContent"
	err = os.WriteFile(mdFile, []byte(markdown), 0644)
	if err != nil {
		t.Fatalf("Failed to write test markdown: %v", err)
	}

	// Try to write to a directory that doesn't exist
	invalidPath := filepath.Join(tmpDir, "nonexistent", "subdir", "output.html")

	err = exporter.ExportToHTML(mdFile, invalidPath)
	if err == nil {
		t.Fatal("Expected error for invalid output path, got nil")
	}
}

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		expected string
	}{
		{
			name:     "First line H1",
			markdown: "# My Title\n\nContent",
			expected: "My Title",
		},
		{
			name:     "H1 with whitespace",
			markdown: "  # Title with Spaces  \n\nContent",
			expected: "Title with Spaces",
		},
		{
			name:     "H1 not first line",
			markdown: "Some content\n# Title\n\nMore content",
			expected: "Title",
		},
		{
			name:     "No H1",
			markdown: "## H2 Only\n\nContent",
			expected: "Documentation",
		},
		{
			name:     "Empty markdown",
			markdown: "",
			expected: "Documentation",
		},
		{
			name:     "Only whitespace",
			markdown: "   \n\n   \n",
			expected: "Documentation",
		},
		{
			name:     "H1 with special characters",
			markdown: "# Title with **bold** and `code`\n",
			expected: "Title with **bold** and `code`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTitle(tt.markdown)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestGetDefaultCSS(t *testing.T) {
	css := getDefaultCSS()

	// Verify CSS contains key styles
	requiredStyles := []string{
		"font-family:",
		".container",
		"max-width:",
		"pre {",
		"code {",
		"table",
		"@media",
		"blockquote",
	}

	for _, style := range requiredStyles {
		if !strings.Contains(css, style) {
			t.Errorf("Expected CSS to contain '%s'", style)
		}
	}
}

func TestHTMLExporter_ComplexMarkdown(t *testing.T) {
	exporter, err := NewHTMLExporter()
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "complex.md")
	htmlFile := filepath.Join(tmpDir, "complex.html")

	// Test with GitHub Flavored Markdown extensions
	markdown := `# Complex Document

## Task Lists

- [x] Completed task
- [ ] Incomplete task

## Strikethrough

~~This text is crossed out~~

## Tables with alignment

| Left | Center | Right |
|:-----|:------:|------:|
| L1   | C1     | R1    |
| L2   | C2     | R2    |

## Multiple code blocks

` + "```python\ndef hello():\n    print(\"Hello\")\n```" + `

` + "```javascript\nfunction hello() {\n    console.log(\"Hello\");\n}\n```" + `

## Nested lists

1. First
   - Sub 1
   - Sub 2
2. Second
   1. Sub A
   2. Sub B

## Inline elements

**Bold**, *italic*, ***bold italic***, ~~strikethrough~~, ` + "`code`" + `
`

	err = os.WriteFile(mdFile, []byte(markdown), 0644)
	if err != nil {
		t.Fatalf("Failed to write markdown: %v", err)
	}

	err = exporter.ExportToHTML(mdFile, htmlFile)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	html, err := os.ReadFile(htmlFile)
	if err != nil {
		t.Fatalf("Failed to read HTML: %v", err)
	}

	htmlStr := string(html)

	// Verify task lists
	if !strings.Contains(htmlStr, "type=\"checkbox\"") {
		t.Error("Expected task list checkboxes")
	}

	// Verify strikethrough
	if !strings.Contains(htmlStr, "<del>") || !strings.Contains(htmlStr, "crossed out") {
		t.Error("Expected strikethrough")
	}

	// Verify tables
	if !strings.Contains(htmlStr, "<table>") {
		t.Error("Expected table")
	}

	// Verify multiple code blocks (syntax highlighting splits tokens)
	hasPython := strings.Contains(htmlStr, "def") && strings.Contains(htmlStr, "hello") && strings.Contains(htmlStr, "print")
	hasJS := strings.Contains(htmlStr, "function") && strings.Contains(htmlStr, "console")

	if !hasPython || !hasJS {
		t.Error("Expected both Python and JavaScript code blocks")
	}
}

func TestHTMLExporter_EmptyMarkdown(t *testing.T) {
	exporter, err := NewHTMLExporter()
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "empty.md")
	htmlFile := filepath.Join(tmpDir, "empty.html")

	err = os.WriteFile(mdFile, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to write empty file: %v", err)
	}

	err = exporter.ExportToHTML(mdFile, htmlFile)
	if err != nil {
		t.Fatalf("Expected success with empty file, got error: %v", err)
	}

	html, err := os.ReadFile(htmlFile)
	if err != nil {
		t.Fatalf("Failed to read HTML: %v", err)
	}

	htmlStr := string(html)

	// Should still have valid HTML structure
	if !strings.Contains(htmlStr, "<!DOCTYPE html>") {
		t.Error("Expected valid HTML document")
	}

	// Should use default title
	if !strings.Contains(htmlStr, "<title>Documentation</title>") {
		t.Error("Expected default title 'Documentation'")
	}
}
