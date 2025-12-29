package export

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/ast"
)

// JSONDocument represents the complete JSON output structure
type JSONDocument struct {
	Metadata Metadata       `json:"metadata"`
	Content  ContentSection `json:"content"`
}

// Metadata contains document metadata
type Metadata struct {
	Title      string    `json:"title"`
	GeneratedAt time.Time `json:"generated_at"`
	Generator  Generator `json:"generator"`
	SourceFile string    `json:"source_file"`
	WordCount  int       `json:"word_count,omitempty"`
	CharCount  int       `json:"char_count,omitempty"`
}

// Generator information
type Generator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	URL     string `json:"url"`
}

// ContentSection contains the document content
type ContentSection struct {
	Headings []Heading `json:"headings"`
	Elements []Element `json:"elements"`
}

// Heading represents a heading in the hierarchy
type Heading struct {
	ID       string    `json:"id"`
	Level    int       `json:"level"`
	Text     string    `json:"text"`
	Children []Heading `json:"children"`
}

// Element is a wrapper for different element types
type Element map[string]interface{}

// ParagraphElement represents a paragraph
type ParagraphElement struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

// HeadingElement represents a heading element in the elements array
type HeadingElement struct {
	Type  string `json:"type"`
	Level int    `json:"level"`
	Text  string `json:"text"`
}

// CodeBlockElement represents a code block
type CodeBlockElement struct {
	Type     string `json:"type"`
	Language string `json:"language"`
	Code     string `json:"code"`
	Lines    int    `json:"lines"`
}

// ListElement represents a list (unordered, ordered, or task)
type ListElement struct {
	Type     string     `json:"type"`
	ListType string     `json:"list_type"` // "unordered", "ordered", "task"
	Start    int        `json:"start,omitempty"` // Starting number for ordered lists
	Items    []ListItem `json:"items"`
}

// ListItem represents a single list item (can be nested)
type ListItem struct {
	Content string     `json:"content"`
	Checked *bool      `json:"checked,omitempty"` // For task lists (nil if not a task)
	Items   []ListItem `json:"items"` // Nested sub-items
}

// TableElement represents a table
type TableElement struct {
	Type   string      `json:"type"`
	Header []TableCell `json:"header"`
	Rows   [][]TableCell `json:"rows"`
}

// TableCell represents a single table cell
type TableCell struct {
	Content   string `json:"content"`
	Alignment string `json:"alignment,omitempty"` // "left", "center", "right", or omitted
}

// BlockquoteElement represents a blockquote
type BlockquoteElement struct {
	Type     string    `json:"type"`
	Content  string    `json:"content"`
	Elements []Element `json:"elements,omitempty"` // For complex nested blockquotes
}

// ThematicBreakElement represents a horizontal rule
type ThematicBreakElement struct {
	Type string `json:"type"`
}

// LinkElement represents a link reference
type LinkElement struct {
	Type  string `json:"type"`
	URL   string `json:"url"`
	Title string `json:"title,omitempty"`
	Text  string `json:"text"`
}

// ImageElement represents an image
type ImageElement struct {
	Type  string `json:"type"`
	URL   string `json:"url"`
	Title string `json:"title,omitempty"`
	Alt   string `json:"alt"`
}

// JSONExporter converts Markdown to structured JSON documents
type JSONExporter struct {
	markdown goldmark.Markdown
}

// NewJSONExporter creates a new JSON exporter with Goldmark configured
func NewJSONExporter() (*JSONExporter, error) {
	// Configure Goldmark with GitHub Flavored Markdown and syntax highlighting
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Table,
			extension.Strikethrough,
			extension.TaskList,
			highlighting.NewHighlighting(
				highlighting.WithStyle("monokai"),
			),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
			html.WithUnsafe(), // Allow raw HTML in Markdown
		),
		goldmark.WithParserOptions(
			parser.WithASTAttribute(), // Enable AST parsing
		),
	)

	return &JSONExporter{
		markdown: md,
	}, nil
}

// ExportToJSON converts a Markdown file to a structured JSON file
func (e *JSONExporter) ExportToJSON(markdownPath, outputPath string) error {
	// Read Markdown file
	mdContent, err := os.ReadFile(markdownPath)
	if err != nil {
		return fmt.Errorf("failed to read markdown: %w", err)
	}

	// Parse markdown to AST
	context := parser.NewContext()
	doc := e.markdown.Parser().Parse(
		textBytes(mdContent),
		parser.WithContext(context),
	)

	// Extract metadata
	title := extractTitle(string(mdContent))
	wordCount, charCount := countText(mdContent)

	// Build JSON document
	jsonDoc, err := e.buildJSONDocument(
		doc,
		string(mdContent),
		markdownPath,
		title,
		wordCount,
		charCount,
	)
	if err != nil {
		return fmt.Errorf("failed to build JSON document: %w", err)
	}

	// Marshal to JSON with indentation
	jsonData, err := marshalJSON(jsonDoc)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write output file
	if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write JSON: %w", err)
	}

	return nil
}

// buildJSONDocument constructs the JSON document structure from the AST
// This is a placeholder - full implementation will be in subtask 2.2
func (e *JSONExporter) buildJSONDocument(
	doc ast.Node,
	markdownContent string,
	sourceFile string,
	title string,
	wordCount int,
	charCount int,
) (*JSONDocument, error) {
	// For now, return a minimal structure
	// Full AST traversal implementation will be in subtask 2.2

	metadata := Metadata{
		Title:      title,
		GeneratedAt: time.Now(),
		Generator: Generator{
			Name:    "Gendocs",
			Version: "1.0.0",
			URL:     "https://github.com/user/gendocs",
		},
		SourceFile: sourceFile,
		WordCount:  wordCount,
		CharCount:  charCount,
	}

	content := ContentSection{
		Headings: []Heading{},
		Elements: []Element{},
	}

	return &JSONDocument{
		Metadata: metadata,
		Content:  content,
	}, nil
}

// marshalJSON converts the JSONDocument to JSON bytes with indentation
func marshalJSON(doc *JSONDocument) ([]byte, error) {
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return data, nil
}

// extractTitle extracts the first H1 heading from Markdown content
func extractTitle(markdown string) string {
	lines := strings.Split(markdown, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimPrefix(trimmed, "# ")
		}
	}
	return "Documentation"
}

// countText counts words and characters in markdown content
func countText(content []byte) (wordCount, charCount int) {
	text := string(content)
	charCount = len(text)

	// Simple word count - split by whitespace
	words := strings.Fields(text)
	wordCount = len(words)

	return wordCount, charCount
}

// Helper conversion function
func textBytes(b []byte) []byte {
	return b
}
