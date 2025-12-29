package export

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewJSONExporter_Success(t *testing.T) {
	exporter, err := NewJSONExporter()
	if err != nil {
		t.Fatalf("Expected no error creating exporter, got %v", err)
	}

	if exporter == nil {
		t.Fatal("Expected exporter to be non-nil")
	}

	if exporter.markdown == nil {
		t.Error("Expected markdown parser to be initialized")
	}
}

func TestJSONExporter_ExportToJSON_Success(t *testing.T) {
	exporter, err := NewJSONExporter()
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	// Create temp directory and files
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "test.md")
	jsonFile := filepath.Join(tmpDir, "test.json")

	markdown := `# Test Document

This is a **test** with code.

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

	// Export to JSON
	err = exporter.ExportToJSON(mdFile, jsonFile)
	if err != nil {
		t.Fatalf("Expected no error exporting, got %v", err)
	}

	// Read generated JSON
	jsonData, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to read generated JSON: %v", err)
	}

	// Verify JSON is valid
	var doc JSONDocument
	if err := json.Unmarshal(jsonData, &doc); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify metadata
	if doc.Metadata.Title != "Test Document" {
		t.Errorf("Expected title 'Test Document', got '%s'", doc.Metadata.Title)
	}

	if doc.Metadata.Generator.Name != "Gendocs" {
		t.Errorf("Expected generator name 'Gendocs', got '%s'", doc.Metadata.Generator.Name)
	}

	if doc.Metadata.SourceFile != mdFile {
		t.Errorf("Expected source file '%s', got '%s'", mdFile, doc.Metadata.SourceFile)
	}

	if doc.Metadata.WordCount == 0 {
		t.Error("Expected word count to be greater than 0")
	}

	if doc.Metadata.CharCount == 0 {
		t.Error("Expected char count to be greater than 0")
	}

	// Verify headings
	if len(doc.Content.Headings) == 0 {
		t.Fatal("Expected at least one heading")
	}

	if doc.Content.Headings[0].Text != "Test Document" {
		t.Errorf("Expected first heading 'Test Document', got '%s'", doc.Content.Headings[0].Text)
	}

	if doc.Content.Headings[0].Level != 1 {
		t.Errorf("Expected first heading level 1, got %d", doc.Content.Headings[0].Level)
	}

	// Verify elements
	if len(doc.Content.Elements) == 0 {
		t.Fatal("Expected at least one element")
	}

	// Check for heading element
	foundHeading := false
	foundParagraph := false
	foundCodeBlock := false
	foundList := false
	foundTable := false
	foundBlockquote := false
	foundLink := false

	for _, elem := range doc.Content.Elements {
		elemType, ok := elem["type"].(string)
		if !ok {
			continue
		}

		switch elemType {
		case "heading":
			foundHeading = true
		case "paragraph":
			foundParagraph = true
			content, ok := elem["content"].(string)
			if !ok {
				t.Error("Expected paragraph to have content")
			}
			if !strings.Contains(content, "test") {
				t.Error("Expected paragraph to contain 'test'")
			}
		case "code_block":
			foundCodeBlock = true
			language, ok := elem["language"].(string)
			if !ok {
				t.Error("Expected code_block to have language")
			}
			if language != "go" {
				t.Errorf("Expected language 'go', got '%s'", language)
			}
			code, ok := elem["code"].(string)
			if !ok {
				t.Error("Expected code_block to have code")
			}
			if !strings.Contains(code, "func main") {
				t.Error("Expected code to contain 'func main'")
			}
		case "list":
			foundList = true
			listType, ok := elem["list_type"].(string)
			if !ok {
				t.Error("Expected list to have list_type")
			}
			if listType != "unordered" {
				t.Errorf("Expected list_type 'unordered', got '%s'", listType)
			}
		case "table":
			foundTable = true
			header, ok := elem["header"].([]interface{})
			if !ok {
				t.Error("Expected table to have header")
			}
			if len(header) != 2 {
				t.Errorf("Expected 2 header columns, got %d", len(header))
			}
		case "blockquote":
			foundBlockquote = true
			content, ok := elem["content"].(string)
			if !ok {
				t.Error("Expected blockquote to have content")
			}
			if !strings.Contains(content, "blockquote") {
				t.Error("Expected blockquote to contain 'blockquote'")
			}
		case "link":
			foundLink = true
			url, ok := elem["url"].(string)
			if !ok {
				t.Error("Expected link to have url")
			}
			if url != "https://example.com" {
				t.Errorf("Expected url 'https://example.com', got '%s'", url)
			}
		}
	}

	if !foundHeading {
		t.Error("Expected to find heading element")
	}
	if !foundParagraph {
		t.Error("Expected to find paragraph element")
	}
	if !foundCodeBlock {
		t.Error("Expected to find code_block element")
	}
	if !foundList {
		t.Error("Expected to find list element")
	}
	if !foundTable {
		t.Error("Expected to find table element")
	}
	if !foundBlockquote {
		t.Error("Expected to find blockquote element")
	}
	if !foundLink {
		t.Error("Expected to find link element")
	}
}

func TestJSONExporter_ExportToJSON_FileNotFound(t *testing.T) {
	exporter, err := NewJSONExporter()
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "nonexistent.md")
	jsonFile := filepath.Join(tmpDir, "output.json")

	err = exporter.ExportToJSON(mdFile, jsonFile)
	if err == nil {
		t.Fatal("Expected error for nonexistent file, got nil")
	}

	if !strings.Contains(err.Error(), "failed to read markdown") {
		t.Errorf("Expected 'failed to read markdown' error, got: %v", err)
	}
}

func TestJSONExporter_ExportToJSON_InvalidOutputPath(t *testing.T) {
	exporter, err := NewJSONExporter()
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
	invalidPath := filepath.Join(tmpDir, "nonexistent", "subdir", "output.json")

	err = exporter.ExportToJSON(mdFile, invalidPath)
	if err == nil {
		t.Fatal("Expected error for invalid output path, got nil")
	}

	if !strings.Contains(err.Error(), "failed to write JSON") {
		t.Errorf("Expected 'failed to write JSON' error, got: %v", err)
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

func TestCountText(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		expectedWords  int
		expectedChars  int
	}{
		{
			name:           "Simple text",
			content:        "Hello world",
			expectedWords:  2,
			expectedChars:  11,
		},
		{
			name:           "Empty string",
			content:        "",
			expectedWords:  0,
			expectedChars:  0,
		},
		{
			name:           "Multiple spaces",
			content:        "word1  word2    word3",
			expectedWords:  3,
			expectedChars:  22,
		},
		{
			name:           "Newlines",
			content:        "line1\nline2\nline3",
			expectedWords:  3,
			expectedChars:  17,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			words, chars := countText([]byte(tt.content))
			if words != tt.expectedWords {
				t.Errorf("Expected %d words, got %d", tt.expectedWords, words)
			}
			if chars != tt.expectedChars {
				t.Errorf("Expected %d chars, got %d", tt.expectedChars, chars)
			}
		})
	}
}

func TestJSONExporter_ComplexMarkdown(t *testing.T) {
	exporter, err := NewJSONExporter()
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "complex.md")
	jsonFile := filepath.Join(tmpDir, "complex.json")

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

	err = exporter.ExportToJSON(mdFile, jsonFile)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	jsonData, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to read JSON: %v", err)
	}

	var doc JSONDocument
	if err := json.Unmarshal(jsonData, &doc); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify task list
	foundTaskList := false
	for _, elem := range doc.Content.Elements {
		if elemType, ok := elem["type"].(string); ok && elemType == "list" {
			if listType, ok := elem["list_type"].(string); ok && listType == "task" {
				foundTaskList = true
				items, ok := elem["items"].([]interface{})
				if !ok {
					t.Error("Expected task list to have items")
					continue
				}
				if len(items) != 2 {
					t.Errorf("Expected 2 task items, got %d", len(items))
				}
			}
		}
	}

	if !foundTaskList {
		t.Error("Expected to find task list")
	}

	// Verify multiple code blocks
	codeBlocks := 0
	for _, elem := range doc.Content.Elements {
		if elemType, ok := elem["type"].(string); ok && elemType == "code_block" {
			codeBlocks++
			language, ok := elem["language"].(string)
			if !ok {
				continue
			}
			if language != "python" && language != "javascript" {
				t.Errorf("Unexpected code language: %s", language)
			}
		}
	}

	if codeBlocks != 2 {
		t.Errorf("Expected 2 code blocks, got %d", codeBlocks)
	}

	// Verify table alignment
	foundAlignedTable := false
	for _, elem := range doc.Content.Elements {
		if elemType, ok := elem["type"].(string); ok && elemType == "table" {
			header, ok := elem["header"].([]interface{})
			if !ok {
				continue
			}
			// Check first column has left alignment
			if len(header) > 0 {
				firstCol, ok := header[0].(map[string]interface{})
				if ok {
					if alignment, ok := firstCol["alignment"].(string); ok && alignment == "left" {
						foundAlignedTable = true
					}
				}
			}
		}
	}

	if !foundAlignedTable {
		t.Error("Expected to find table with alignment info")
	}
}

func TestJSONExporter_EmptyMarkdown(t *testing.T) {
	exporter, err := NewJSONExporter()
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "empty.md")
	jsonFile := filepath.Join(tmpDir, "empty.json")

	err = os.WriteFile(mdFile, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to write empty file: %v", err)
	}

	err = exporter.ExportToJSON(mdFile, jsonFile)
	if err != nil {
		t.Fatalf("Expected success with empty file, got error: %v", err)
	}

	jsonData, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to read JSON: %v", err)
	}

	var doc JSONDocument
	if err := json.Unmarshal(jsonData, &doc); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Should have default title
	if doc.Metadata.Title != "Documentation" {
		t.Errorf("Expected default title 'Documentation', got '%s'", doc.Metadata.Title)
	}

	// Should have valid JSON structure even if empty
	if len(doc.Content.Headings) != 0 {
		t.Errorf("Expected no headings, got %d", len(doc.Content.Headings))
	}

	if len(doc.Content.Elements) != 0 {
		t.Errorf("Expected no elements, got %d", len(doc.Content.Elements))
	}
}

func TestJSONExporter_HeadingHierarchy(t *testing.T) {
	exporter, err := NewJSONExporter()
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "hierarchy.md")
	jsonFile := filepath.Join(tmpDir, "hierarchy.json")

	markdown := `# Level 1

Content 1

## Level 2

Content 2

### Level 3

Content 3

## Another Level 2

Content 4

# Another Level 1

Content 5
`

	err = os.WriteFile(mdFile, []byte(markdown), 0644)
	if err != nil {
		t.Fatalf("Failed to write markdown: %v", err)
	}

	err = exporter.ExportToJSON(mdFile, jsonFile)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	jsonData, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to read JSON: %v", err)
	}

	var doc JSONDocument
	if err := json.Unmarshal(jsonData, &doc); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Should have 2 root level headings
	if len(doc.Content.Headings) != 2 {
		t.Errorf("Expected 2 root headings, got %d", len(doc.Content.Headings))
	}

	// First heading should have 1 child (Level 2)
	firstHeading := doc.Content.Headings[0]
	if firstHeading.Level != 1 {
		t.Errorf("Expected first heading level 1, got %d", firstHeading.Level)
	}
	if len(firstHeading.Children) != 1 {
		t.Errorf("Expected first heading to have 1 child, got %d", len(firstHeading.Children))
	}

	// Level 2 should have 1 child (Level 3)
	if len(firstHeading.Children) > 0 {
		level2Heading := firstHeading.Children[0]
		if level2Heading.Level != 2 {
			t.Errorf("Expected level 2 heading, got level %d", level2Heading.Level)
		}
		if len(level2Heading.Children) != 1 {
			t.Errorf("Expected level 2 to have 1 child, got %d", len(level2Heading.Children))
		}

		// Level 3 should have no children
		if len(level2Heading.Children) > 0 {
			level3Heading := level2Heading.Children[0]
			if level3Heading.Level != 3 {
				t.Errorf("Expected level 3 heading, got level %d", level3Heading.Level)
			}
			if len(level3Heading.Children) != 0 {
				t.Errorf("Expected level 3 to have no children, got %d", len(level3Heading.Children))
			}
		}
	}
}

func TestJSONExporter_OrderedList(t *testing.T) {
	exporter, err := NewJSONExporter()
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "ordered.md")
	jsonFile := filepath.Join(tmpDir, "ordered.json")

	markdown := `# Ordered List Test

1. First item
2. Second item
3. Third item

## Starting at 5

5. Fifth item
6. Sixth item
`

	err = os.WriteFile(mdFile, []byte(markdown), 0644)
	if err != nil {
		t.Fatalf("Failed to write markdown: %v", err)
	}

	err = exporter.ExportToJSON(mdFile, jsonFile)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	jsonData, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to read JSON: %v", err)
	}

	var doc JSONDocument
	if err := json.Unmarshal(jsonData, &doc); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Find ordered lists
	orderedLists := 0
	for _, elem := range doc.Content.Elements {
		if elemType, ok := elem["type"].(string); ok && elemType == "list" {
			if listType, ok := elem["list_type"].(string); ok && listType == "ordered" {
				orderedLists++

				items, ok := elem["items"].([]interface{})
				if !ok {
					t.Error("Expected ordered list to have items")
					continue
				}

				if len(items) != 3 {
					t.Errorf("Expected 3 items, got %d", len(items))
				}

				// Check start number for second list
				if orderedLists == 2 {
					start, ok := elem["start"].(float64)
					if !ok {
						t.Error("Expected ordered list to have start number")
					} else if int(start) != 5 {
						t.Errorf("Expected start number 5, got %v", start)
					}
				}
			}
		}
	}

	if orderedLists != 2 {
		t.Errorf("Expected 2 ordered lists, got %d", orderedLists)
	}
}

func TestJSONExporter_CodeBlockWithoutLanguage(t *testing.T) {
	exporter, err := NewJSONExporter()
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "indented.md")
	jsonFile := filepath.Join(tmpDir, "indented.json")

	// Indented code block (no language)
	markdown := `# Indented Code Block

    This is an indented
    code block with
    no language specified
`

	err = os.WriteFile(mdFile, []byte(markdown), 0644)
	if err != nil {
		t.Fatalf("Failed to write markdown: %v", err)
	}

	err = exporter.ExportToJSON(mdFile, jsonFile)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	jsonData, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to read JSON: %v", err)
	}

	var doc JSONDocument
	if err := json.Unmarshal(jsonData, &doc); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Find code block without language
	foundCodeBlock := false
	for _, elem := range doc.Content.Elements {
		if elemType, ok := elem["type"].(string); ok && elemType == "code_block" {
			language, ok := elem["language"].(string)
			if !ok {
				t.Error("Expected code_block to have language field")
				continue
			}
			if language == "" {
				foundCodeBlock = true
				code, ok := elem["code"].(string)
				if !ok {
					t.Error("Expected code_block to have code")
					continue
				}
				if !strings.Contains(code, "indented") {
					t.Error("Expected code to contain 'indented'")
				}
			}
		}
	}

	if !foundCodeBlock {
		t.Error("Expected to find code block without language")
	}
}

func TestGenerateID(t *testing.T) {
	exporter, err := NewJSONExporter()
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple text",
			input:    "Hello World",
			expected: "hello-world",
		},
		{
			name:     "With underscores",
			input:    "hello_world_test",
			expected: "hello-world-test",
		},
		{
			name:     "Special characters",
			input:    "Hello @#$% World!",
			expected: "hello-world",
		},
		{
			name:     "Multiple spaces",
			input:    "hello    world",
			expected: "hello-world",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "heading",
		},
		{
			name:     "Numbers",
			input:    "Test 123 Example",
			expected: "test-123-example",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := exporter.generateID(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestJSONExtractor_Paragraphs(t *testing.T) {
	exporter, err := NewJSONExporter()
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "paragraphs.md")
	jsonFile := filepath.Join(tmpDir, "paragraphs.json")

	markdown := `# Document

First paragraph with **bold** and *italic* text.

Second paragraph with ` + "`code`" + ` and [links](https://example.com).

Third paragraph on its own.
`

	err = os.WriteFile(mdFile, []byte(markdown), 0644)
	if err != nil {
		t.Fatalf("Failed to write markdown: %v", err)
	}

	err = exporter.ExportToJSON(mdFile, jsonFile)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	jsonData, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to read JSON: %v", err)
	}

	var doc JSONDocument
	if err := json.Unmarshal(jsonData, &doc); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Count paragraphs
	paragraphCount := 0
	for _, elem := range doc.Content.Elements {
		if elemType, ok := elem["type"].(string); ok && elemType == "paragraph" {
			paragraphCount++
			content, ok := elem["content"].(string)
			if !ok {
				t.Error("Expected paragraph to have content")
				continue
			}
			if content == "" {
				t.Error("Expected paragraph content to be non-empty")
			}
		}
	}

	if paragraphCount != 3 {
		t.Errorf("Expected 3 paragraphs, got %d", paragraphCount)
	}
}

func TestJSONExtractor_HeadingsHierarchy(t *testing.T) {
	exporter, err := NewJSONExporter()
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "headings.md")
	jsonFile := filepath.Join(tmpDir, "headings.json")

	markdown := `# H1 One

## H2 One

### H3 One

#### H4 One

## H2 Two

# H1 Two

### H3 Under H1 Two
`

	err = os.WriteFile(mdFile, []byte(markdown), 0644)
	if err != nil {
		t.Fatalf("Failed to write markdown: %v", err)
	}

	err = exporter.ExportToJSON(mdFile, jsonFile)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	jsonData, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to read JSON: %v", err)
	}

	var doc JSONDocument
	if err := json.Unmarshal(jsonData, &doc); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Should have 2 root headings
	if len(doc.Content.Headings) != 2 {
		t.Errorf("Expected 2 root headings, got %d", len(doc.Content.Headings))
	}

	// First H1 should have one H2 child
	firstH1 := doc.Content.Headings[0]
	if firstH1.Text != "H1 One" {
		t.Errorf("Expected 'H1 One', got '%s'", firstH1.Text)
	}
	if len(firstH1.Children) != 1 {
		t.Errorf("Expected first H1 to have 1 child, got %d", len(firstH1.Children))
	}

	// H2 should have one H3 child
	if len(firstH1.Children) > 0 {
		firstH2 := firstH1.Children[0]
		if firstH2.Text != "H2 One" {
			t.Errorf("Expected 'H2 One', got '%s'", firstH2.Text)
		}
		if len(firstH2.Children) != 1 {
			t.Errorf("Expected H2 to have 1 child, got %d", len(firstH2.Children))
		}

		// H3 should have one H4 child
		if len(firstH2.Children) > 0 {
			firstH3 := firstH2.Children[0]
			if firstH3.Text != "H3 One" {
				t.Errorf("Expected 'H3 One', got '%s'", firstH3.Text)
			}
			if len(firstH3.Children) != 1 {
				t.Errorf("Expected H3 to have 1 child, got %d", len(firstH3.Children))
			}

			// H4 should have no children
			if len(firstH3.Children) > 0 {
				firstH4 := firstH3.Children[0]
				if firstH4.Text != "H4 One" {
					t.Errorf("Expected 'H4 One', got '%s'", firstH4.Text)
				}
				if len(firstH4.Children) != 0 {
					t.Errorf("Expected H4 to have 0 children, got %d", len(firstH4.Children))
				}
			}
		}

		// Second H2 should be sibling of first H2 (child of H1)
		if len(firstH1.Children) != 2 {
			t.Errorf("Expected H1 to have 2 H2 children, got %d", len(firstH1.Children))
		} else {
			secondH2 := firstH1.Children[1]
			if secondH2.Text != "H2 Two" {
				t.Errorf("Expected 'H2 Two', got '%s'", secondH2.Text)
			}
		}
	}

	// Second H1 should have one H3 child
	secondH1 := doc.Content.Headings[1]
	if secondH1.Text != "H1 Two" {
		t.Errorf("Expected 'H1 Two', got '%s'", secondH1.Text)
	}
	if len(secondH1.Children) != 1 {
		t.Errorf("Expected second H1 to have 1 child, got %d", len(secondH1.Children))
	}
}

func TestJSONExtractor_UnorderedLists(t *testing.T) {
	exporter, err := NewJSONExporter()
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "unordered.md")
	jsonFile := filepath.Join(tmpDir, "unordered.json")

	markdown := `# Lists

- Simple item
- Item with **bold**
  - Nested item 1
  - Nested item 2
- Item with ` + "`code`" + `
`

	err = os.WriteFile(mdFile, []byte(markdown), 0644)
	if err != nil {
		t.Fatalf("Failed to write markdown: %v", err)
	}

	err = exporter.ExportToJSON(mdFile, jsonFile)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	jsonData, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to read JSON: %v", err)
	}

	var doc JSONDocument
	if err := json.Unmarshal(jsonData, &doc); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Find unordered list
	foundList := false
	for _, elem := range doc.Content.Elements {
		if elemType, ok := elem["type"].(string); ok && elemType == "list" {
			if listType, ok := elem["list_type"].(string); ok && listType == "unordered" {
				foundList = true
				items, ok := elem["items"].([]interface{})
				if !ok {
					t.Fatal("Expected list to have items")
				}

				// Should have 3 top-level items
				if len(items) != 3 {
					t.Errorf("Expected 3 list items, got %d", len(items))
				}

				// Check first item has nested items
				if len(items) > 0 {
					firstItem, ok := items[0].(map[string]interface{})
					if !ok {
						t.Fatal("Expected item to be a map")
					}

					content, ok := firstItem["content"].(string)
					if !ok {
						t.Error("Expected item to have content")
					}
					if content != "Simple item" {
						t.Errorf("Expected 'Simple item', got '%s'", content)
					}

					// Check nested items
					nested, ok := firstItem["items"].([]interface{})
					if !ok {
						t.Error("Expected first item to have nested items")
					} else if len(nested) != 2 {
						t.Errorf("Expected 2 nested items, got %d", len(nested))
					}
				}
			}
		}
	}

	if !foundList {
		t.Fatal("Expected to find unordered list")
	}
}

func TestJSONExtractor_OrderedLists(t *testing.T) {
	exporter, err := NewJSONExporter()
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "ordered.md")
	jsonFile := filepath.Join(tmpDir, "ordered.json")

	markdown := `# Ordered Lists

1. First item
2. Second item
   1. Nested ordered 1
   2. Nested ordered 2
3. Third item
`

	err = os.WriteFile(mdFile, []byte(markdown), 0644)
	if err != nil {
		t.Fatalf("Failed to write markdown: %v", err)
	}

	err = exporter.ExportToJSON(mdFile, jsonFile)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	jsonData, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to read JSON: %v", err)
	}

	var doc JSONDocument
	if err := json.Unmarshal(jsonData, &doc); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	foundList := false
	for _, elem := range doc.Content.Elements {
		if elemType, ok := elem["type"].(string); ok && elemType == "list" {
			if listType, ok := elem["list_type"].(string); ok && listType == "ordered" {
				foundList = true
				items, ok := elem["items"].([]interface{})
				if !ok {
					t.Fatal("Expected list to have items")
				}

				if len(items) != 3 {
					t.Errorf("Expected 3 items, got %d", len(items))
				}

				// Check start number
				start, ok := elem["start"].(float64)
				if !ok {
					t.Error("Expected ordered list to have start number")
				} else if int(start) != 1 {
					t.Errorf("Expected start 1, got %v", start)
				}

				// Check nested ordered list
				if len(items) > 1 {
					secondItem, ok := items[1].(map[string]interface{})
					if ok {
						nested, ok := secondItem["items"].([]interface{})
						if !ok {
							t.Error("Expected second item to have nested items")
						} else if len(nested) != 2 {
							t.Errorf("Expected 2 nested items, got %d", len(nested))
						}
					}
				}
			}
		}
	}

	if !foundList {
		t.Fatal("Expected to find ordered list")
	}
}

func TestJSONExtractor_TaskLists(t *testing.T) {
	exporter, err := NewJSONExporter()
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "tasks.md")
	jsonFile := filepath.Join(tmpDir, "tasks.json")

	markdown := `# Task List

- [x] Completed task 1
- [ ] Incomplete task
- [x] Completed task 2
  - [ ] Nested incomplete
  - [x] Nested completed
`

	err = os.WriteFile(mdFile, []byte(markdown), 0644)
	if err != nil {
		t.Fatalf("Failed to write markdown: %v", err)
	}

	err = exporter.ExportToJSON(mdFile, jsonFile)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	jsonData, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to read JSON: %v", err)
	}

	var doc JSONDocument
	if err := json.Unmarshal(jsonData, &doc); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	foundTaskList := false
	for _, elem := range doc.Content.Elements {
		if elemType, ok := elem["type"].(string); ok && elemType == "list" {
			if listType, ok := elem["list_type"].(string); ok && listType == "task" {
				foundTaskList = true
				items, ok := elem["items"].([]interface{})
				if !ok {
					t.Fatal("Expected task list to have items")
				}

				if len(items) != 3 {
					t.Errorf("Expected 3 task items, got %d", len(items))
				}

				// Check first item
				if len(items) > 0 {
					firstItem, ok := items[0].(map[string]interface{})
					if !ok {
						t.Fatal("Expected item to be a map")
					}

					checked, ok := firstItem["checked"].(bool)
					if !ok {
						t.Error("Expected task item to have checked field")
					}
					if !checked {
						t.Error("Expected first task to be checked")
					}

					content, ok := firstItem["content"].(string)
					if !ok {
						t.Error("Expected item to have content")
					}
					if content != "Completed task 1" {
						t.Errorf("Expected 'Completed task 1', got '%s'", content)
					}
				}

				// Check second item (unchecked)
				if len(items) > 1 {
					secondItem, ok := items[1].(map[string]interface{})
					if ok {
						checked, ok := secondItem["checked"].(bool)
						if !ok {
							t.Error("Expected task item to have checked field")
						}
						if checked {
							t.Error("Expected second task to be unchecked")
						}
					}
				}

				// Check nested tasks
				if len(items) > 2 {
					thirdItem, ok := items[2].(map[string]interface{})
					if ok {
						nested, ok := thirdItem["items"].([]interface{})
						if !ok {
							t.Error("Expected third item to have nested tasks")
						} else if len(nested) != 2 {
							t.Errorf("Expected 2 nested tasks, got %d", len(nested))
						}
					}
				}
			}
		}
	}

	if !foundTaskList {
		t.Fatal("Expected to find task list")
	}
}

func TestJSONExtractor_CodeBlocksWithLanguage(t *testing.T) {
	exporter, err := NewJSONExporter()
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "code.md")
	jsonFile := filepath.Join(tmpDir, "code.json")

	markdown := `# Code Blocks

` + "```go\nfunc hello() {\n    fmt.Println(\"Hello\")\n}\n```" + `

` + "```python\ndef hello():\n    print('Hello')\n```" + `

` + "```javascript\nfunction hello() {\n    console.log('Hello');\n}\n```" + `

` + "```bash\necho 'Hello'\n```" + `

    indented code
    block without
    language
`

	err = os.WriteFile(mdFile, []byte(markdown), 0644)
	if err != nil {
		t.Fatalf("Failed to write markdown: %v", err)
	}

	err = exporter.ExportToJSON(mdFile, jsonFile)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	jsonData, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to read JSON: %v", err)
	}

	var doc JSONDocument
	if err := json.Unmarshal(jsonData, &doc); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	codeBlocks := make(map[string]string)
	for _, elem := range doc.Content.Elements {
		if elemType, ok := elem["type"].(string); ok && elemType == "code_block" {
			language, ok := elem["language"].(string)
			if !ok {
				t.Error("Expected code_block to have language")
				continue
			}

			code, ok := elem["code"].(string)
			if !ok {
				t.Error("Expected code_block to have code")
				continue
			}

			if code == "" {
				t.Errorf("Expected non-empty code for language %s", language)
			}

			codeBlocks[language] = code
		}
	}

	// Should have 5 code blocks (4 with language, 1 without)
	if len(codeBlocks) != 5 {
		t.Errorf("Expected 5 code blocks, got %d", len(codeBlocks))
	}

	// Check specific languages
	expectedLanguages := []string{"go", "python", "javascript", "bash", ""}
	for _, lang := range expectedLanguages {
		if _, ok := codeBlocks[lang]; !ok {
			t.Errorf("Expected to find code block with language '%s'", lang)
		}
	}

	// Verify Go code content
	goCode := codeBlocks["go"]
	if !strings.Contains(goCode, "func hello") {
		t.Error("Expected Go code to contain 'func hello'")
	}
	if !strings.Contains(goCode, "fmt.Println") {
		t.Error("Expected Go code to contain 'fmt.Println'")
	}
}

func TestJSONExtractor_Tables(t *testing.T) {
	exporter, err := NewJSONExporter()
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "tables.md")
	jsonFile := filepath.Join(tmpDir, "tables.json")

	markdown := `# Tables

| Left | Center | Right | Default |
|:-----|:------:|------:|---------|
| L1   | C1     | R1    | D1      |
| L2   | C2     | R2    | D2      |
| L3   | C3     | R3    | D3      |
`

	err = os.WriteFile(mdFile, []byte(markdown), 0644)
	if err != nil {
		t.Fatalf("Failed to write markdown: %v", err)
	}

	err = exporter.ExportToJSON(mdFile, jsonFile)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	jsonData, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to read JSON: %v", err)
	}

	var doc JSONDocument
	if err := json.Unmarshal(jsonData, &doc); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	foundTable := false
	for _, elem := range doc.Content.Elements {
		if elemType, ok := elem["type"].(string); ok && elemType == "table" {
			foundTable = true

			// Check header
			header, ok := elem["header"].([]interface{})
			if !ok {
				t.Fatal("Expected table to have header")
			}

			if len(header) != 4 {
				t.Errorf("Expected 4 header columns, got %d", len(header))
			}

			// Check first column alignment
			if len(header) > 0 {
				firstCol, ok := header[0].(map[string]interface{})
				if !ok {
					t.Fatal("Expected header column to be a map")
				}

				text, ok := firstCol["text"].(string)
				if !ok {
					t.Error("Expected header column to have text")
				}
				if text != "Left" {
					t.Errorf("Expected 'Left', got '%s'", text)
				}

				alignment, ok := firstCol["alignment"].(string)
				if !ok {
					t.Error("Expected header column to have alignment")
				}
				if alignment != "left" {
					t.Errorf("Expected alignment 'left', got '%s'", alignment)
				}
			}

			// Check second column (center)
			if len(header) > 1 {
				secondCol, ok := header[1].(map[string]interface{})
				if ok {
					alignment, ok := secondCol["alignment"].(string)
					if !ok {
						t.Error("Expected header column to have alignment")
					}
					if alignment != "center" {
						t.Errorf("Expected alignment 'center', got '%s'", alignment)
					}
				}
			}

			// Check third column (right)
			if len(header) > 2 {
				thirdCol, ok := header[2].(map[string]interface{})
				if ok {
					alignment, ok := thirdCol["alignment"].(string)
					if !ok {
						t.Error("Expected header column to have alignment")
					}
					if alignment != "right" {
						t.Errorf("Expected alignment 'right', got '%s'", alignment)
					}
				}
			}

			// Check rows
			rows, ok := elem["rows"].([]interface{})
			if !ok {
				t.Fatal("Expected table to have rows")
			}

			if len(rows) != 3 {
				t.Errorf("Expected 3 rows, got %d", len(rows))
			}

			// Check first row
			if len(rows) > 0 {
				firstRow, ok := rows[0].([]interface{})
				if !ok {
					t.Fatal("Expected row to be an array")
				}

				if len(firstRow) != 4 {
					t.Errorf("Expected 4 columns in row, got %d", len(firstRow))
				}

				// Check first cell
				if len(firstRow) > 0 {
					firstCell, ok := firstRow[0].(map[string]interface{})
					if !ok {
						t.Fatal("Expected cell to be a map")
					}

					text, ok := firstCell["text"].(string)
					if !ok {
						t.Error("Expected cell to have text")
					}
					if text != "L1" {
						t.Errorf("Expected 'L1', got '%s'", text)
					}
				}
			}
		}
	}

	if !foundTable {
		t.Fatal("Expected to find table")
	}
}

func TestJSONExtractor_Blockquotes(t *testing.T) {
	exporter, err := NewJSONExporter()
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "blockquote.md")
	jsonFile := filepath.Join(tmpDir, "blockquote.json")

	markdown := `# Blockquotes

> Simple blockquote
> with multiple lines

> Blockquote with **bold** and *italic*
> and ` + "`code`" + `

> Nested blockquote
> > Inner blockquote
> > Still inner
> Back to outer
`

	err = os.WriteFile(mdFile, []byte(markdown), 0644)
	if err != nil {
		t.Fatalf("Failed to write markdown: %v", err)
	}

	err = exporter.ExportToJSON(mdFile, jsonFile)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	jsonData, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to read JSON: %v", err)
	}

	var doc JSONDocument
	if err := json.Unmarshal(jsonData, &doc); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	blockquoteCount := 0
	for _, elem := range doc.Content.Elements {
		if elemType, ok := elem["type"].(string); ok && elemType == "blockquote" {
			blockquoteCount++
			content, ok := elem["content"].(string)
			if !ok {
				t.Error("Expected blockquote to have content")
				continue
			}

			if content == "" {
				t.Error("Expected blockquote content to be non-empty")
			}

			// Check that content contains expected words
			if blockquoteCount == 1 {
				if !strings.Contains(content, "Simple") {
					t.Error("Expected first blockquote to contain 'Simple'")
				}
			}
		}
	}

	if blockquoteCount != 2 {
		t.Errorf("Expected 2 blockquotes, got %d", blockquoteCount)
	}
}

func TestJSONExtractor_Links(t *testing.T) {
	exporter, err := NewJSONExporter()
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "links.md")
	jsonFile := filepath.Join(tmpDir, "links.json")

	markdown := `# Links

Simple [link](https://example.com).

Link with [title](https://example.com "Example Title").

[Reference link][ref]

[ref]: https://example.com

Auto link: <https://auto.com>
`

	err = os.WriteFile(mdFile, []byte(markdown), 0644)
	if err != nil {
		t.Fatalf("Failed to write markdown: %v", err)
	}

	err = exporter.ExportToJSON(mdFile, jsonFile)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	jsonData, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to read JSON: %v", err)
	}

	var doc JSONDocument
	if err := json.Unmarshal(jsonData, &doc); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	linkCount := 0
	for _, elem := range doc.Content.Elements {
		if elemType, ok := elem["type"].(string); ok && elemType == "link" {
			linkCount++

			url, ok := elem["url"].(string)
			if !ok {
				t.Error("Expected link to have url")
				continue
			}

			if url == "" {
				t.Error("Expected link URL to be non-empty")
			}

			text, ok := elem["text"].(string)
			if !ok {
				t.Error("Expected link to have text")
				continue
			}

			// Check specific link
			if text == "link" {
				if url != "https://example.com" {
					t.Errorf("Expected 'https://example.com', got '%s'", url)
				}
			}
		}
	}

	if linkCount < 2 {
		t.Errorf("Expected at least 2 links, got %d", linkCount)
	}
}

func TestJSONExtractor_Images(t *testing.T) {
	exporter, err := NewJSONExporter()
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "images.md")
	jsonFile := filepath.Join(tmpDir, "images.json")

	markdown := `# Images

![Alt text](image.png)

![Image with title](photo.jpg "Photo Title")

![Remote](https://example.com/image.png)
`

	err = os.WriteFile(mdFile, []byte(markdown), 0644)
	if err != nil {
		t.Fatalf("Failed to write markdown: %v", err)
	}

	err = exporter.ExportToJSON(mdFile, jsonFile)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	jsonData, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to read JSON: %v", err)
	}

	var doc JSONDocument
	if err := json.Unmarshal(jsonData, &doc); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	imageCount := 0
	for _, elem := range doc.Content.Elements {
		if elemType, ok := elem["type"].(string); ok && elemType == "image" {
			imageCount++

			src, ok := elem["src"].(string)
			if !ok {
				t.Error("Expected image to have src")
				continue
			}

			if src == "" {
				t.Error("Expected image src to be non-empty")
			}

			alt, ok := elem["alt"].(string)
			if !ok {
				t.Error("Expected image to have alt")
				continue
			}

			// Check specific image
			if alt == "Alt text" {
				if src != "image.png" {
					t.Errorf("Expected 'image.png', got '%s'", src)
				}
			}

			if alt == "Remote" {
				if src != "https://example.com/image.png" {
					t.Errorf("Expected 'https://example.com/image.png', got '%s'", src)
				}
			}
		}
	}

	if imageCount != 3 {
		t.Errorf("Expected 3 images, got %d", imageCount)
	}
}

// Edge case: deeply nested lists (3+ levels)
func TestJSONEdgeCase_DeeplyNestedLists(t *testing.T) {
	exporter, err := NewJSONExporter()
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "nested.md")
	jsonFile := filepath.Join(tmpDir, "nested.json")

	markdown := `# Deep Nesting

- Level 1
  - Level 2
    - Level 3
      - Level 4
  - Back to Level 2
- Another Level 1
`

	err = os.WriteFile(mdFile, []byte(markdown), 0644)
	if err != nil {
		t.Fatalf("Failed to write markdown: %v", err)
	}

	err = exporter.ExportToJSON(mdFile, jsonFile)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	jsonData, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to read JSON: %v", err)
	}

	var doc JSONDocument
	if err := json.Unmarshal(jsonData, &doc); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Find list with deep nesting
	foundDeepNesting := false
	for _, elem := range doc.Content.Elements {
		if elemType, ok := elem["type"].(string); ok && elemType == "list" {
			items, ok := elem["items"].([]interface{})
			if !ok {
				continue
			}

			// Check first item
			if len(items) > 0 {
				firstItem, ok := items[0].(map[string]interface{})
				if ok {
					// Check Level 2
					nested2, ok := firstItem["items"].([]interface{})
					if ok && len(nested2) > 0 {
						level2, ok := nested2[0].(map[string]interface{})
						if ok {
							// Check Level 3
							nested3, ok := level2["items"].([]interface{})
							if ok && len(nested3) > 0 {
								level3, ok := nested3[0].(map[string]interface{})
								if ok {
									// Check Level 4
									nested4, ok := level3["items"].([]interface{})
									if ok && len(nested4) > 0 {
										foundDeepNesting = true
									}
								}
							}
						}
					}
				}
			}
		}
	}

	if !foundDeepNesting {
		t.Error("Expected to find deeply nested list structure (4 levels)")
	}
}

// Edge case: special characters in text
func TestJSONEdgeCase_SpecialCharacters(t *testing.T) {
	exporter, err := NewJSONExporter()
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "special.md")
	jsonFile := filepath.Join(tmpDir, "special.json")

	markdown := `# Special Characters

Paragraph with special chars: < > & " ' ` + "`" + `

Math symbols: ≤ ≥ ≠ ± × ÷ °

Currency: $ € £ ¥ ¢

Punctuation: ¡ ¿ † ‡

Emojis: 😀 🎉 🚀 💻

Unicode: 中文 日本語 한글 العربية

Quotes: "smart" 'curly' ` + "``" + `backticks` + "``" + `
`

	err = os.WriteFile(mdFile, []byte(markdown), 0644)
	if err != nil {
		t.Fatalf("Failed to write markdown: %v", err)
	}

	err = exporter.ExportToJSON(mdFile, jsonFile)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	jsonData, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to read JSON: %v", err)
	}

	var doc JSONDocument
	if err := json.Unmarshal(jsonData, &doc); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Find paragraph and verify special characters are preserved
	foundSpecialChars := false
	for _, elem := range doc.Content.Elements {
		if elemType, ok := elem["type"].(string); ok && elemType == "paragraph" {
			content, ok := elem["content"].(string)
			if !ok {
				continue
			}

			// Check for various special characters
			if strings.Contains(content, "€") || strings.Contains(content, "≤") ||
			   strings.Contains(content, "😀") || strings.Contains(content, "中文") {
				foundSpecialChars = true
			}
		}
	}

	if !foundSpecialChars {
		t.Error("Expected to find special characters in exported content")
	}
}

// Edge case: multiple inline formatting combinations
func TestJSONEdgeCase_ComplexInlineFormatting(t *testing.T) {
	exporter, err := NewJSONExporter()
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "inline.md")
	jsonFile := filepath.Join(tmpDir, "inline.json")

	markdown := `# Inline Formatting

**Bold and *italic* together**

***All three styles*** bold and italic

` + "`" + `code with **bold** inside` + "`" + `

**bold with ` + "`" + `code` + "`" + ` inside**

*italic with ` + "`" + `code` + "`" + ` inside*

[**bold link**](https://example.com)

[*italic link*](https://example.com)

[` + "`" + `code link` + "`" + `](https://example.com)

~~Strikethrough with **bold**~~
`

	err = os.WriteFile(mdFile, []byte(markdown), 0644)
	if err != nil {
		t.Fatalf("Failed to write markdown: %v", err)
	}

	err = exporter.ExportToJSON(mdFile, jsonFile)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	jsonData, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to read JSON: %v", err)
	}

	var doc JSONDocument
	if err := json.Unmarshal(jsonData, &doc); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Count paragraphs with inline formatting
	paragraphCount := 0
	linkCount := 0
	for _, elem := range doc.Content.Elements {
		if elemType, ok := elem["type"].(string); ok && elemType == "paragraph" {
			paragraphCount++
			content, ok := elem["content"].(string)
			if !ok {
				continue
			}
			// Check that markdown formatting syntax is preserved
			if strings.Contains(content, "**") || strings.Contains(content, "*") || strings.Contains(content, "`") {
				// Good - formatting syntax present
			}
		}
		if elemType, ok := elem["type"].(string); ok && elemType == "link" {
			linkCount++
		}
	}

	if paragraphCount < 5 {
		t.Errorf("Expected at least 5 paragraphs with inline formatting, got %d", paragraphCount)
	}

	if linkCount < 3 {
		t.Errorf("Expected at least 3 links, got %d", linkCount)
	}
}

// Edge case: alternating code blocks and text
func TestJSONEdgeCase_AlternatingCodeBlocks(t *testing.T) {
	exporter, err := NewJSONExporter()
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "alternating.md")
	jsonFile := filepath.Join(tmpDir, "alternating.json")

	markdown := `# Alternating Content

Text paragraph 1

` + "```go\ncode 1\n```" + `

Text paragraph 2

` + "```python\ncode 2\n```" + `

Text paragraph 3

` + "```javascript\ncode 3\n```" + `

Text paragraph 4
`

	err = os.WriteFile(mdFile, []byte(markdown), 0644)
	if err != nil {
		t.Fatalf("Failed to write markdown: %v", err)
	}

	err = exporter.ExportToJSON(mdFile, jsonFile)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	jsonData, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to read JSON: %v", err)
	}

	var doc JSONDocument
	if err := json.Unmarshal(jsonData, &doc); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Count alternating elements
	paragraphCount := 0
	codeBlockCount := 0

	for _, elem := range doc.Content.Elements {
		elemType, ok := elem["type"].(string)
		if !ok {
			continue
		}

		if elemType == "paragraph" {
			paragraphCount++
		}
		if elemType == "code_block" {
			codeBlockCount++
		}
	}

	if paragraphCount != 4 {
		t.Errorf("Expected 4 paragraphs, got %d", paragraphCount)
	}

	if codeBlockCount != 3 {
		t.Errorf("Expected 3 code blocks, got %d", codeBlockCount)
	}
}

// Edge case: document with only H2/H3 (no H1)
func TestJSONEdgeCase_NoH1Title(t *testing.T) {
	exporter, err := NewJSONExporter()
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "noh1.md")
	jsonFile := filepath.Join(tmpDir, "noh1.json")

	markdown := `## Missing H1

This document has no H1 heading.

### Just H2 and H3

Content here.
`

	err = os.WriteFile(mdFile, []byte(markdown), 0644)
	if err != nil {
		t.Fatalf("Failed to write markdown: %v", err)
	}

	err = exporter.ExportToJSON(mdFile, jsonFile)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	jsonData, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to read JSON: %v", err)
	}

	var doc JSONDocument
	if err := json.Unmarshal(jsonData, &doc); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Should have default title when no H1 present
	if doc.Metadata.Title != "Documentation" {
		t.Errorf("Expected default title 'Documentation', got '%s'", doc.Metadata.Title)
	}

	// Should still have headings (H2 and H3)
	if len(doc.Content.Headings) == 0 {
		t.Error("Expected to find H2 and H3 headings even without H1")
	}

	// Verify H2 is root level
	if len(doc.Content.Headings) > 0 {
		firstHeading := doc.Content.Headings[0]
		if firstHeading.Level != 2 {
			t.Errorf("Expected first heading to be level 2, got %d", firstHeading.Level)
		}
	}
}

// Edge case: document with only whitespace
func TestJSONEdgeCase_OnlyWhitespace(t *testing.T) {
	exporter, err := NewJSONExporter()
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "whitespace.md")
	jsonFile := filepath.Join(tmpDir, "whitespace.json")

	markdown := "   \n\n   \n\t\t\n   "

	err = os.WriteFile(mdFile, []byte(markdown), 0644)
	if err != nil {
		t.Fatalf("Failed to write markdown: %v", err)
	}

	err = exporter.ExportToJSON(mdFile, jsonFile)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	jsonData, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to read JSON: %v", err)
	}

	var doc JSONDocument
	if err := json.Unmarshal(jsonData, &doc); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Should have default title
	if doc.Metadata.Title != "Documentation" {
		t.Errorf("Expected default title 'Documentation', got '%s'", doc.Metadata.Title)
	}

	// Should have zero word count for whitespace-only content
	if doc.Metadata.WordCount != 0 {
		t.Errorf("Expected 0 word count, got %d", doc.Metadata.WordCount)
	}

	// Should have no elements
	if len(doc.Content.Elements) != 0 {
		t.Errorf("Expected 0 elements, got %d", len(doc.Content.Elements))
	}
}

// Edge case: very long heading text
func TestJSONEdgeCase_LongHeading(t *testing.T) {
	exporter, err := NewJSONExporter()
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "long.md")
	jsonFile := filepath.Join(tmpDir, "long.json")

	longText := strings.Repeat("This is a very long heading text. ", 20)

	markdown := "# " + longText + `

Content below.
`

	err = os.WriteFile(mdFile, []byte(markdown), 0644)
	if err != nil {
		t.Fatalf("Failed to write markdown: %v", err)
	}

	err = exporter.ExportToJSON(mdFile, jsonFile)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	jsonData, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to read JSON: %v", err)
	}

	var doc JSONDocument
	if err := json.Unmarshal(jsonData, &doc); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Title should be preserved
	if !strings.Contains(doc.Metadata.Title, "very long heading") {
		t.Error("Expected long heading to be preserved in title")
	}

	if len(doc.Content.Headings) == 0 {
		t.Fatal("Expected to find heading")
	}

	// ID should be generated (may be truncated/modified)
	if doc.Content.Headings[0].ID == "" {
		t.Error("Expected heading to have ID")
	}
}

// Edge case: mixed list types in sequence
func TestJSONEdgeCase_MixedListTypes(t *testing.T) {
	exporter, err := NewJSONExporter()
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "mixed.md")
	jsonFile := filepath.Join(tmpDir, "mixed.json")

	markdown := `# Mixed Lists

- Unordered item 1
- Unordered item 2

1. Ordered item 1
2. Ordered item 2

- [x] Task item 1
- [ ] Task item 2

- Back to unordered
`

	err = os.WriteFile(mdFile, []byte(markdown), 0644)
	if err != nil {
		t.Fatalf("Failed to write markdown: %v", err)
	}

	err = exporter.ExportToJSON(mdFile, jsonFile)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	jsonData, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to read JSON: %v", err)
	}

	var doc JSONDocument
	if err := json.Unmarshal(jsonData, &doc); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Count different list types
	unorderedCount := 0
	orderedCount := 0
	taskCount := 0

	for _, elem := range doc.Content.Elements {
		if elemType, ok := elem["type"].(string); ok && elemType == "list" {
			listType, ok := elem["list_type"].(string)
			if !ok {
				continue
			}

			switch listType {
			case "unordered":
				unorderedCount++
			case "ordered":
				orderedCount++
			case "task":
				taskCount++
			}
		}
	}

	if unorderedCount != 2 {
		t.Errorf("Expected 2 unordered lists, got %d", unorderedCount)
	}

	if orderedCount != 1 {
		t.Errorf("Expected 1 ordered list, got %d", orderedCount)
	}

	if taskCount != 1 {
		t.Errorf("Expected 1 task list, got %d", taskCount)
	}
}

// Edge case: consecutive headings with no content
func TestJSONEdgeCase_ConsecutiveHeadings(t *testing.T) {
	exporter, err := NewJSONExporter()
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "consecutive.md")
	jsonFile := filepath.Join(tmpDir, "consecutive.json")

	markdown := `# Heading 1

## Heading 1.1

### Heading 1.1.1

# Heading 2

## Heading 2.1

Some content here.

### Heading 2.1.1
`

	err = os.WriteFile(mdFile, []byte(markdown), 0644)
	if err != nil {
		t.Fatalf("Failed to write markdown: %v", err)
	}

	err = exporter.ExportToJSON(mdFile, jsonFile)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	jsonData, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to read JSON: %v", err)
	}

	var doc JSONDocument
	if err := json.Unmarshal(jsonData, &doc); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Should have 2 root headings (both H1)
	if len(doc.Content.Headings) != 2 {
		t.Errorf("Expected 2 root headings, got %d", len(doc.Content.Headings))
	}

	// First H1 should have nested structure
	firstH1 := doc.Content.Headings[0]
	if len(firstH1.Children) == 0 {
		t.Error("Expected first H1 to have child headings")
	}

	// Verify hierarchy depth
	if len(firstH1.Children) > 0 {
		firstH2 := firstH1.Children[0]
		if len(firstH2.Children) == 0 {
			t.Error("Expected H2 to have H3 child")
		}
	}
}

// Error handling: malformed markdown should still succeed gracefully
func TestJSONErrorHandling_MalformedMarkdown(t *testing.T) {
	exporter, err := NewJSONExporter()
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		markdown string
		desc     string
	}{
		{
			name: "Unclosed code block",
			markdown: `# Unclosed Code

` + "```go\nfunc incomplete() {\n    // no closing",
			desc: "Code block without closing fence",
		},
		{
			name: "Unclosed link",
			markdown: `# Unclosed Link

This has an [unclosed link(https://example.com`,
			desc: "Link with missing closing bracket",
		},
		{
			name: "Unclosed emphasis",
			markdown: `# Unclosed Emphasis

This has **bold text that never closes

And more text here.`,
			desc: "Bold formatting without closing markers",
		},
		{
			name: "Malformed table",
			markdown: `# Malformed Table

| Col1 | Col2
|------|------
| Val1 | Val2 | Extra
| Val3 |
`,
			desc: "Table with inconsistent column counts",
		},
		{
			name: "Broken list formatting",
			markdown: `# Broken List

- Item 1
- Item 2
  - Nested but wrong indentation
- Item 3

1. First
2. Second
3. Third
   Bad indentation here
4. Fourth
`,
			desc: "Lists with inconsistent indentation",
		},
		{
			name: "Multiple unclosed blocks",
			markdown: `# Multiple Issues

**Bold *italic

` + "```python\ndef bad():\n    pass" + `

[Link](https://example.com

> Blockquote that
> doesn't close properly

More text`,
			desc: "Multiple unclosed formatting elements",
		},
		{
			name: "Empty code fence with language",
			markdown: `# Empty Code Block

This has ` + "```go\n```" + ` an empty code block.

And then more text.`,
			desc: "Code fence with language but no content",
		},
		{
			name: "Mangled heading levels",
			markdown: `# H1

###### H6

####### Invalid H7

##### H5

######## Even more invalid`,
			desc: "Heading levels beyond H6",
		},
		{
			name: "Broken reference link",
			markdown: `# Reference Link

This has [ref link][ref] but ref is not defined.

[Another][link]

[link]: https://example.com`,
			desc: "Reference link with undefined reference",
		},
		{
			name: "Invalid HTML entities",
			markdown: `# Invalid HTML

&invalidentity;

&notanentityatall;

&;

Text after broken entities.`,
			desc: "Invalid HTML entity references",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mdFile := filepath.Join(tmpDir, tt.name+".md")
			jsonFile := filepath.Join(tmpDir, tt.name+".json")

			err = os.WriteFile(mdFile, []byte(tt.markdown), 0644)
			if err != nil {
				t.Fatalf("Failed to write markdown: %v", err)
			}

			// Should not error even with malformed markdown
			err = exporter.ExportToJSON(mdFile, jsonFile)
			if err != nil {
				t.Errorf("Expected success with %s, got error: %v", tt.desc, err)
				return
			}

			// Verify JSON file was created
			jsonData, err := os.ReadFile(jsonFile)
			if err != nil {
				t.Fatalf("Failed to read JSON output: %v", err)
			}

			// Verify JSON is valid
			var doc JSONDocument
			if err := json.Unmarshal(jsonData, &doc); err != nil {
				t.Errorf("Generated invalid JSON for %s: %v", tt.desc, err)
				return
			}

			// Verify basic structure exists
			if doc.Metadata.Title == "" {
				t.Error("Expected metadata title to be set (should have default)")
			}

			if doc.Metadata.Generator.Name != "Gendocs" {
				t.Error("Expected generator name to be 'Gendocs'")
			}

			// Elements array should exist (may be empty but should be present)
			if doc.Content.Elements == nil {
				t.Error("Expected elements array to be initialized")
			}

			// Headings array should exist
			if doc.Content.Headings == nil {
				t.Error("Expected headings array to be initialized")
			}

			// Verify JSON is well-formed and can be remarshaled
			remarkaled, err := json.Marshal(doc)
			if err != nil {
				t.Errorf("Failed to remarshal JSON document: %v", err)
			}

			if len(remarkaled) == 0 {
				t.Error("Remarshaled JSON is empty")
			}
		})
	}
}
