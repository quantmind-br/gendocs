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
