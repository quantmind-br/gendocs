package export

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
)

// JSONExporter converts Markdown to structured JSON documents
type JSONExporter struct {
	source   []byte
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
			parser.WithAutoHeadingID(),
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
	e.source = mdContent

	// Parse markdown to AST
	context := parser.NewContext()
	doc := e.markdown.Parser().Parse(
		text.NewReader(mdContent),
		parser.WithContext(context),
	)

	// Extract metadata
	title := extractJSONTitle(string(mdContent))
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
		Title:       title,
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
	var headingNodes []headingTreeNode
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
	headingNodes *[]headingTreeNode,
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
	case *east.Table:
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
	headingNodes *[]headingTreeNode,
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

	newNode := headingTreeNode{
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
	language := string(code.Language(e.source))

	// Extract code content using the Text() method
	codeStr := string(code.Text(e.source)) //nolint:staticcheck // TODO: migrate to code.Lines

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
	// Extract code content using the Text() method
	codeStr := string(code.Text(e.source)) //nolint:staticcheck // TODO: migrate to code.Lines

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

	// Check if it's a task list by checking ALL items
	// Task lists have a TaskCheckBox in their items
	hasTaskCheckBox := false
	for itemChild := list.FirstChild(); itemChild != nil; itemChild = itemChild.NextSibling() {
		if item, ok := itemChild.(*ast.ListItem); ok {
			// Check all children for TaskCheckBox (could be direct child or in TextBlock)
			for child := item.FirstChild(); child != nil; child = child.NextSibling() {
				if _, ok := child.(*east.TaskCheckBox); ok {
					hasTaskCheckBox = true
					break
				}
				// Also check inside TextBlock
				if textBlock, ok := child.(*ast.TextBlock); ok {
					for gc := textBlock.FirstChild(); gc != nil; gc = gc.NextSibling() {
						if _, ok := gc.(*east.TaskCheckBox); ok {
							hasTaskCheckBox = true
							break
						}
					}
				}
			}
			if hasTaskCheckBox {
				break
			}
		}
	}
	if hasTaskCheckBox {
		listType = "task"
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

		// Check for task checkbox (may be inside TextBlock)
		var checked *bool
		if listItem.FirstChild() != nil {
			// TaskCheckBox might be a direct child (old behavior)
			if checkbox, ok := listItem.FirstChild().(*east.TaskCheckBox); ok {
				isChecked := checkbox.IsChecked
				checked = &isChecked
			} else if textBlock, ok := listItem.FirstChild().(*ast.TextBlock); ok {
				// Or it might be inside a TextBlock
				if textBlock.FirstChild() != nil {
					if checkbox, ok := textBlock.FirstChild().(*east.TaskCheckBox); ok {
						isChecked := checkbox.IsChecked
						checked = &isChecked
					}
				}
			}
		}

		// Extract nested lists
		var nestedItems []ListItem
		for child := listItem.FirstChild(); child != nil; child = child.NextSibling() {
			if nestedList, ok := child.(*ast.List); ok {
				nestedItems = e.extractListItems(nestedList)
				break
			}
			// Also check inside TextBlock for nested lists
			if textBlock, ok := child.(*ast.TextBlock); ok {
				for gc := textBlock.FirstChild(); gc != nil; gc = gc.NextSibling() {
					if nestedList, ok := gc.(*ast.List); ok {
						nestedItems = e.extractListItems(nestedList)
						break
					}
				}
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
		if _, ok := child.(*east.TaskCheckBox); ok {
			continue
		}
		if _, ok := child.(*ast.List); ok {
			break
		}
		// Handle TextBlocks (which contain the actual text content)
		if textBlock, ok := child.(*ast.TextBlock); ok {
			// Extract text from all children of TextBlock, skipping TaskCheckBox
			for gc := textBlock.FirstChild(); gc != nil; gc = gc.NextSibling() {
				if _, ok := gc.(*east.TaskCheckBox); ok {
					continue
				}
				// Handle Text nodes directly
				if textNode, ok := gc.(*ast.Text); ok {
					content.WriteString(string(textNode.Segment.Value(e.source)))
				} else if strNode, ok := gc.(*ast.String); ok {
					content.WriteString(string(strNode.Value))
				} else {
					content.WriteString(e.extractNodeText(gc))
				}
			}
		} else {
			// For other node types, use the standard extraction
			content.WriteString(e.extractNodeText(child))
		}
	}

	return strings.TrimSpace(content.String())
}

// processTable processes a table node
func (e *JSONExporter) processTable(table *east.Table, elements *[]Element) {
	var header []TableCell
	var rows [][]TableCell

	// Process table children - TableHeader comes first, then TableRows
	for child := table.FirstChild(); child != nil; child = child.NextSibling() {
		switch n := child.(type) {
		case *east.TableHeader:
			// Extract header row from TableHeader
			header = e.extractTableHeader(n, table)
		case *east.TableRow:
			// Extract data row from TableRow
			rowData := e.extractTableRow(n, table)
			rows = append(rows, rowData)
		}
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

// extractTableHeader extracts the header row from a TableHeader node
func (e *JSONExporter) extractTableHeader(header *east.TableHeader, table *east.Table) []TableCell {
	var cells []TableCell
	colIndex := 0

	for cell := header.FirstChild(); cell != nil; cell = cell.NextSibling() {
		tableCell, ok := cell.(*east.TableCell)
		if !ok {
			continue
		}

		content := e.extractText(tableCell)

		// Determine alignment
		var alignment string
		if table.Alignments != nil && colIndex < len(table.Alignments) {
			switch table.Alignments[colIndex] {
			case east.AlignLeft:
				alignment = "left"
			case east.AlignCenter:
				alignment = "center"
			case east.AlignRight:
				alignment = "right"
			default:
				alignment = "default"
			}
		}

		cells = append(cells, TableCell{
			Content:   strings.TrimSpace(content),
			Alignment: alignment,
		})
		colIndex++
	}

	return cells
}

// extractTableRow extracts a single table row
func (e *JSONExporter) extractTableRow(row *east.TableRow, table *east.Table) []TableCell {
	var cells []TableCell
	colIndex := 0

	for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
		tableCell, ok := cell.(*east.TableCell)
		if !ok {
			continue
		}

		content := e.extractText(tableCell)

		// Determine alignment
		var alignment string
		if table.Alignments != nil && colIndex < len(table.Alignments) {
			switch table.Alignments[colIndex] {
			case east.AlignLeft:
				alignment = "left"
			case east.AlignCenter:
				alignment = "center"
			case east.AlignRight:
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
			text.WriteString(string(n.Segment.Value(e.source)))
		case *ast.String:
			text.WriteString(string(n.Value))
		case *ast.Emphasis, *ast.CodeSpan, *ast.Link, *ast.Image:
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

	// Collapse multiple consecutive hyphens into one
	for strings.Contains(id, "--") {
		id = strings.ReplaceAll(id, "--", "-")
	}

	// Trim leading/trailing hyphens
	id = strings.Trim(id, "-")

	if id == "" {
		return "heading"
	}

	return id
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
func extractJSONTitle(markdown string) string {
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

// buildHeadingHierarchy constructs the hierarchical heading structure
func (e *JSONExporter) buildHeadingHierarchy(nodes []headingTreeNode) []Heading {
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
func (e *JSONExporter) buildHeadingNode(node *headingTreeNode, allNodes []headingTreeNode) *Heading {
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
