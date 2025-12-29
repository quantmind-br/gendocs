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

// headingNode is used for building the hierarchy during AST traversal
type headingNode struct {
	heading  *Heading
	parent   int // index in the heading slice
	children []int // indices of children
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
func (e *JSONExporter) buildJSONDocument(
	doc ast.Node,
	markdownContent string,
	sourceFile string,
	title string,
	wordCount int,
	charCount int,
) (*JSONDocument, error) {
	// Build metadata
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

	// Extract content from AST
	headings, elements := e.traverseAST(doc)

	content := ContentSection{
		Headings: headings,
		Elements: elements,
	}

	return &JSONDocument{
		Metadata: metadata,
		Content:  content,
	}, nil
}

// traverseAST walks through the AST and extracts headings and elements
func (e *JSONExporter) traverseAST(doc ast.Node) ([]Heading, []Element) {
	var elements []Element
	var headingNodes []headingNode
	var headingStack []int // Stack of heading indices

	// Walk the AST
	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		e.processNode(child, &headingNodes, &headingStack, &elements)
	}

	// Build the final heading hierarchy
	headings := e.buildHeadingHierarchy(headingNodes)

	return headings, elements
}

// processNode processes a single AST node
func (e *JSONExporter) processNode(
	node ast.Node,
	headingNodes *[]headingNode,
	headingStack *[]int,
	elements *[]Element,
) {
	switch n := node.(type) {
	case *ast.Heading:
		e.processHeading(n, headingNodes, headingStack, elements)
	case *ast.Paragraph:
		e.processParagraph(n, elements)
	case *ast.FencedCodeBlock:
		e.processFencedCodeBlock(n, elements)
	case *ast.CodeBlock:
		e.processCodeBlock(n, elements)
	case *ast.List:
		e.processList(n, elements)
	case *ast.Table:
		e.processTable(n, elements)
	case *ast.Blockquote:
		e.processBlockquote(n, elements)
	case *ast.ThematicBreak:
		e.processThematicBreak(elements)
	case *ast.Text:
		// Text nodes are handled within their parent containers
		return
	case *ast.String:
		// String nodes are handled within their parent containers
		return
	}

	// Extract inline elements (links, images) from paragraphs only
	// This avoids duplication when walking the entire tree
	if _, isParagraph := node.(*ast.Paragraph); isParagraph {
		e.extractInlineElements(node, elements)
	}
}

// processHeading processes a heading node
func (e *JSONExporter) processHeading(
	headingNode *ast.Heading,
	headingNodes *[]headingNode,
	headingStack *[]int,
	elements *[]Element,
) {
	// Extract heading text
	text := e.extractText(headingNode)

	// Create heading element for elements array
	headingElem := HeadingElement{
		Type:  "heading",
		Level: headingNode.Level,
		Text:  text,
	}
	*elements = append(*elements, map[string]interface{}{
		"type":  headingElem.Type,
		"level": headingElem.Level,
		"text":  headingElem.Text,
	})

	// Create heading for hierarchy
	newHeading := Heading{
		ID:    e.generateID(text),
		Level: headingNode.Level,
		Text:  text,
	}

	nodeIdx := len(*headingNodes)
	parentIdx := -1

	// Pop headings that are at the same level or higher (less nested)
	for len(*headingStack) > 0 {
		topIdx := (*headingStack)[len(*headingStack)-1]
		if (*headingNodes)[topIdx].heading.Level >= headingNode.Level {
			*headingStack = (*headingStack)[:len(*headingStack)-1]
		} else {
			break
		}
	}

	// Find parent
	if len(*headingStack) > 0 {
		parentIdx = (*headingStack)[len(*headingStack)-1]
	}

	newNode := headingNode{
		heading:  &newHeading,
		parent:   parentIdx,
		children: []int{},
	}

	*headingNodes = append(*headingNodes, newNode)
	*headingStack = append(*headingStack, nodeIdx)

	// Add this node as a child of its parent
	if parentIdx >= 0 {
		(*headingNodes)[parentIdx].children = append((*headingNodes)[parentIdx].children, nodeIdx)
	}
}

// processParagraph processes a paragraph node
func (e *JSONExporter) processParagraph(para *ast.Paragraph, elements *[]Element) {
	content := e.extractText(para)

	// Only add non-empty paragraphs
	if strings.TrimSpace(content) != "" {
		paraElem := ParagraphElement{
			Type:    "paragraph",
			Content: content,
		}
		*elements = append(*elements, map[string]interface{}{
			"type":    paraElem.Type,
			"content": paraElem.Content,
		})
	}
}

// processFencedCodeBlock processes a fenced code block
func (e *JSONExporter) processFencedCodeBlock(code *ast.FencedCodeBlock, elements *[]Element) {
	// Extract language
	language := string(code.Language(sourceTextProvider{}))

	// Extract code content
	var codeBuilder strings.Builder
	for child := code.FirstChild(); child != nil; child = child.NextSibling() {
		if text, ok := child.(*ast.Text); ok {
			codeBuilder.WriteString(string(text.Segment.Value))
		}
	}

	codeStr := codeBuilder.String()
	lines := strings.Count(codeStr, "\n") + 1
	if strings.TrimSpace(codeStr) == "" {
		lines = 0
	}

	codeElem := CodeBlockElement{
		Type:     "code_block",
		Language: language,
		Code:     codeStr,
		Lines:    lines,
	}

	*elements = append(*elements, map[string]interface{}{
		"type":     codeElem.Type,
		"language": codeElem.Language,
		"code":     codeElem.Code,
		"lines":    codeElem.Lines,
	})
}

// processCodeBlock processes an indented code block
func (e *JSONExporter) processCodeBlock(code *ast.CodeBlock, elements *[]Element) {
	// Extract code content
	var codeBuilder strings.Builder
	for child := code.FirstChild(); child != nil; child = child.NextSibling() {
		if text, ok := child.(*ast.Text); ok {
			codeBuilder.WriteString(string(text.Segment.Value))
		}
	}

	codeStr := codeBuilder.String()
	lines := strings.Count(codeStr, "\n") + 1
	if strings.TrimSpace(codeStr) == "" {
		lines = 0
	}

	codeElem := CodeBlockElement{
		Type:     "code_block",
		Language: "", // Indented code blocks have no language
		Code:     codeStr,
		Lines:    lines,
	}

	*elements = append(*elements, map[string]interface{}{
		"type":     codeElem.Type,
		"language": codeElem.Language,
		"code":     codeElem.Code,
		"lines":    codeElem.Lines,
	})
}

// processList processes a list node
func (e *JSONExporter) processList(list *ast.List, elements *[]Element) {
	// Determine list type
	listType := "unordered"
	start := 0

	if list.IsOrdered() {
		listType = "ordered"
		start = list.Start
	}

	// Check if it's a task list
	isTaskList := false
	if list.FirstChild() != nil {
		if item, ok := list.FirstChild().(*ast.ListItem); ok {
			if item.FirstChild() != nil {
				if _, ok := item.FirstChild().(*ast.TaskCheckBox); ok {
					isTaskList = true
					listType = "task"
				}
			}
		}
	}

	// Extract list items
	items := e.extractListItems(list)

	listElem := ListElement{
		Type:     "list",
		ListType: listType,
		Start:    start,
		Items:    items,
	}

	elemData := map[string]interface{}{
		"type":      listElem.Type,
		"list_type": listElem.ListType,
		"items":     listElem.Items,
	}
	if start > 0 {
		elemData["start"] = start
	}

	*elements = append(*elements, elemData)
}

// extractListItems recursively extracts list items
func (e *JSONExporter) extractListItems(list *ast.List) []ListItem {
	var items []ListItem

	for item := list.FirstChild(); item != nil; item = item.NextSibling() {
		listItem, ok := item.(*ast.ListItem)
		if !ok {
			continue
		}

		// Extract content (text before any nested list)
		content := e.extractListItemContent(listItem)

		// Check for task checkbox
		var checked *bool
		if listItem.FirstChild() != nil {
			if checkbox, ok := listItem.FirstChild().(*ast.TaskCheckBox); ok {
				isChecked := checkbox.IsChecked
				checked = &isChecked
			}
		}

		// Extract nested lists
		var nestedItems []ListItem
		for child := listItem.FirstChild(); child != nil; child = child.NextSibling() {
			if nestedList, ok := child.(*ast.List); ok {
				nestedItems = e.extractListItems(nestedList)
				break
			}
		}

		items = append(items, ListItem{
			Content: content,
			Checked: checked,
			Items:   nestedItems,
		})
	}

	return items
}

// extractListItemContent extracts text content from a list item
func (e *JSONExporter) extractListItemContent(item *ast.ListItem) string {
	var content strings.Builder

	for child := item.FirstChild(); child != nil; child = child.NextSibling() {
		// Skip task checkboxes and nested lists
		if _, ok := child.(*ast.TaskCheckBox); ok {
			continue
		}
		if _, ok := child.(*ast.List); ok {
			break
		}
		content.WriteString(e.extractNodeText(child))
	}

	return strings.TrimSpace(content.String())
}

// processTable processes a table node
func (e *JSONExporter) processTable(table *ast.Table, elements *[]Element) {
	// Extract header row
	var header []TableCell
	if row := table.FirstChild(); row != nil {
		if tableRow, ok := row.(*ast.TableRow); ok {
			header = e.extractTableRow(tableRow, table)
		}
	}

	// Extract data rows
	var rows [][]TableCell
	rowIndex := 0
	for row := table.FirstChild(); row != nil; row = row.NextSibling() {
		tableRow, ok := row.(*ast.TableRow)
		if !ok {
			continue
		}

		// Skip first row (header)
		if rowIndex == 0 {
			rowIndex++
			continue
		}

		rowData := e.extractTableRow(tableRow, table)
		rows = append(rows, rowData)
		rowIndex++
	}

	tableElem := TableElement{
		Type:   "table",
		Header: header,
		Rows:   rows,
	}

	*elements = append(*elements, map[string]interface{}{
		"type":   tableElem.Type,
		"header": tableElem.Header,
		"rows":   tableElem.Rows,
	})
}

// extractTableRow extracts a single table row
func (e *JSONExporter) extractTableRow(row *ast.TableRow, table *ast.Table) []TableCell {
	var cells []TableCell
	colIndex := 0

	for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
		tableCell, ok := cell.(*ast.TableCell)
		if !ok {
			continue
		}

		content := e.extractText(tableCell)

		// Determine alignment
		var alignment string
		if table.Alignments != nil && colIndex < len(table.Alignments) {
			switch table.Alignments[colIndex] {
			case ast.AlignLeft:
				alignment = "left"
			case ast.AlignCenter:
				alignment = "center"
			case ast.AlignRight:
				alignment = "right"
			default:
				alignment = ""
			}
		}

		cellData := TableCell{
			Content: content,
		}
		if alignment != "" {
			cellData.Alignment = alignment
		}

		cells = append(cells, cellData)
		colIndex++
	}

	return cells
}

// processBlockquote processes a blockquote node
func (e *JSONExporter) processBlockquote(blockquote *ast.Blockquote, elements *[]Element) {
	content := e.extractText(blockquote)

	blockquoteElem := BlockquoteElement{
		Type:     "blockquote",
		Content:  content,
		Elements: []Element{}, // Could be extended to include nested elements
	}

	*elements = append(*elements, map[string]interface{}{
		"type":     blockquoteElem.Type,
		"content":  blockquoteElem.Content,
		"elements": blockquoteElem.Elements,
	})
}

// processThematicBreak processes a horizontal rule
func (e *JSONExporter) processThematicBreak(elements *[]Element) {
	breakElem := ThematicBreakElement{
		Type: "thematic_break",
	}

	*elements = append(*elements, map[string]interface{}{
		"type": breakElem.Type,
	})
}

// extractInlineElements extracts links and images as separate elements
func (e *JSONExporter) extractInlineElements(node ast.Node, elements *[]Element) {
	// Walk through children to find links and images
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		e.extractInlineElements(child, elements)

		switch n := child.(type) {
		case *ast.Link:
			linkElem := LinkElement{
				Type:  "link",
				URL:   string(n.Destination),
				Title: string(n.Title),
				Text:  e.extractText(n),
			}

			*elements = append(*elements, map[string]interface{}{
				"type":  linkElem.Type,
				"url":   linkElem.URL,
				"title": linkElem.Title,
				"text":  linkElem.Text,
			})

		case *ast.Image:
			imageElem := ImageElement{
				Type:  "image",
				URL:   string(n.Destination),
				Title: string(n.Title),
				Alt:   e.extractText(n),
			}

			*elements = append(*elements, map[string]interface{}{
				"type":  imageElem.Type,
				"url":   imageElem.URL,
				"title": imageElem.Title,
				"alt":   imageElem.Alt,
			})
		}
	}
}

// extractText extracts plain text from a node
func (e *JSONExporter) extractText(node ast.Node) string {
	return e.extractNodeText(node)
}

// extractNodeText recursively extracts text from a node
func (e *JSONExporter) extractNodeText(node ast.Node) string {
	var text strings.Builder

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		switch n := child.(type) {
		case *ast.Text:
			text.WriteString(string(n.Segment.Value))
		case *ast.String:
			text.WriteString(string(n.Value))
		case *ast.Emphasis, *ast.Strong, *ast.CodeSpan, *ast.Link, *ast.Image:
			// Recursively extract text from inline elements
			text.WriteString(e.extractNodeText(n))
		default:
			// Recursively process other node types
			text.WriteString(e.extractNodeText(n))
		}
	}

	return text.String()
}

// generateID generates a unique ID from text
func (e *JSONExporter) generateID(text string) string {
	// Simple slugification
	slug := strings.ToLower(strings.TrimSpace(text))
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")

	// Remove non-alphanumeric characters (except hyphens)
	var result strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}

	id := result.String()
	if id == "" {
		return "heading"
	}

	return id
}

// sourceTextProvider implements text source for code blocks
type sourceTextProvider struct{}

func (s sourceTextProvider) Text(source []byte) []byte {
	return source
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

// buildHeadingHierarchy constructs the hierarchical heading structure
func (e *JSONExporter) buildHeadingHierarchy(nodes []headingNode) []Heading {
	var roots []Heading

	// Build hierarchy from nodes
	for _, node := range nodes {
		if node.parent < 0 {
			// This is a root level heading
			roots = append(roots, *e.buildHeadingNode(&node, nodes))
		}
	}

	return roots
}

// buildHeadingNode recursively builds a heading with its children
func (e *JSONExporter) buildHeadingNode(node *headingNode, allNodes []headingNode) *Heading {
	result := node.heading

	// Build children
	for _, childIdx := range node.children {
		if childIdx < len(allNodes) {
			childNode := e.buildHeadingNode(&allNodes[childIdx], allNodes)
			result.Children = append(result.Children, *childNode)
		}
	}

	return result
}
