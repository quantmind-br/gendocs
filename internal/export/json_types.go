package export

import "time"

// JSONDocument represents the complete JSON output structure
type JSONDocument struct {
	Metadata Metadata       `json:"metadata"`
	Content  ContentSection `json:"content"`
}

// Metadata contains document metadata
type Metadata struct {
	Title       string    `json:"title"`
	GeneratedAt time.Time `json:"generated_at"`
	Generator   Generator `json:"generator"`
	SourceFile  string    `json:"source_file"`
	WordCount   int       `json:"word_count,omitempty"`
	CharCount   int       `json:"char_count,omitempty"`
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

// headingTreeNode is used for building the hierarchy during AST traversal
type headingTreeNode struct {
	heading  *Heading
	parent   int
	children []int
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
	ListType string     `json:"list_type"`
	Start    int        `json:"start,omitempty"`
	Items    []ListItem `json:"items"`
}

// ListItem represents a single list item (can be nested)
type ListItem struct {
	Content string     `json:"content"`
	Checked *bool      `json:"checked,omitempty"`
	Items   []ListItem `json:"items"`
}

// TableElement represents a table
type TableElement struct {
	Type   string        `json:"type"`
	Header []TableCell   `json:"header"`
	Rows   [][]TableCell `json:"rows"`
}

// TableCell represents a single table cell
type TableCell struct {
	Content   string `json:"text"`
	Alignment string `json:"alignment,omitempty"`
}

// BlockquoteElement represents a blockquote
type BlockquoteElement struct {
	Type     string    `json:"type"`
	Content  string    `json:"content"`
	Elements []Element `json:"elements,omitempty"`
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
