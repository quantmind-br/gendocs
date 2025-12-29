# JSON Export Format Guide

## Overview

The JSON export format converts Markdown documentation into structured JSON data, enabling programmatic access to documentation content for further processing, indexing, or integration with other tools.

## When to Use JSON Export

JSON export is ideal for:

- **API Documentation**: Generate API reference documentation
- **Search Indexing**: Feed documentation into search engines (Elasticsearch, Algolia, etc.)
- **Static Site Generators**: Process documentation with custom templates
- **Documentation Portals**: Integrate into existing documentation platforms
- **Data Analysis**: Analyze documentation structure and content
- **Content Migration**: Convert documentation between systems
- **Custom Processing**: Apply custom transformations to documentation

## Quick Start

```bash
# Export to JSON
gendocs generate export --input README.md --output docs.json --format json

# Export with progress indicator
gendocs generate export --input README.md --output docs.json --format json --progress
```

## JSON Structure Overview

The JSON output consists of two main sections:

```json
{
  "metadata": {
    // Document metadata (title, timestamps, generator info)
  },
  "content": {
    "headings": [
      // Hierarchical heading tree for navigation
    ],
    "elements": [
      // Flat list of all document elements in order
    ]
  }
}
```

## Metadata Section

Contains information about the document and generation process.

### Fields

| Field | Type | Description |
|-------|------|-------------|
| `title` | string | Document title (from first H1 or "Documentation") |
| `generated_at` | string | ISO 8601 timestamp when JSON was generated |
| `generator.name` | string | Generator name ("Gendocs") |
| `generator.version` | string | Generator version |
| `generator.url` | string | Generator repository URL |
| `source_file` | string | Original markdown filename |
| `word_count` | number | Total word count (optional) |
| `char_count` | number | Total character count (optional) |

### Example

```json
{
  "metadata": {
    "title": "Getting Started with Gendocs",
    "generated_at": "2025-12-29T10:30:00Z",
    "generator": {
      "name": "Gendocs",
      "version": "1.0.0",
      "url": "https://github.com/user/gendocs"
    },
    "source_file": "README.md",
    "word_count": 1234,
    "char_count": 5678
  }
}
```

## Content Structure

### Headings Array

Hierarchical tree of document headings for navigation and table of contents generation.

**Structure:**

```json
{
  "headings": [
    {
      "id": "getting-started",
      "level": 1,
      "text": "Getting Started",
      "children": [
        {
          "id": "installation",
          "level": 2,
          "text": "Installation",
          "children": []
        }
      ]
    }
  ]
}
```

**Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier (URL-safe slug) |
| `level` | number | Heading level (1-6) |
| `text` | string | Heading text (plain text, no markdown) |
| `children` | array | Nested child headings |

**Use Cases:**

- Generate table of contents
- Create navigation menus
- Build documentation site structure
- Enable anchor link navigation

### Elements Array

Flat list of all document elements in document order. Each element has a `type` field indicating its kind.

**Element Types:**

#### 1. Paragraph

```json
{
  "type": "paragraph",
  "content": "This is a paragraph with **bold** and *italic* text."
}
```

**Fields:**
- `type`: "paragraph"
- `content`: Text content with inline markdown formatting

#### 2. Heading

```json
{
  "type": "heading",
  "level": 2,
  "text": "Section Title"
}
```

**Fields:**
- `type`: "heading"
- `level`: Heading level (1-6)
- `text`: Heading text (plain)

#### 3. Code Block

```json
{
  "type": "code_block",
  "language": "go",
  "code": "func main() {\n    fmt.Println(\"Hello\")\n}",
  "lines": 3
}
```

**Fields:**
- `type`: "code_block"
- `language`: Language identifier (or empty if none)
- `code`: Code content
- `lines`: Number of lines

#### 4. List

```json
{
  "type": "list",
  "list_type": "unordered",
  "items": [
    {
      "content": "First item",
      "items": []
    }
  ]
}
```

**Fields:**
- `type`: "list"
- `list_type`: "unordered", "ordered", or "task"
- `start`: (optional) Starting number for ordered lists
- `items`: Array of list items (can be nested)

**List Item Fields:**
- `content`: Item text content
- `checked`: (optional) Boolean for task lists
- `items`: Nested sub-items

#### 5. Table

```json
{
  "type": "table",
  "header": [
    {"content": "Column 1", "alignment": "left"},
    {"content": "Column 2", "alignment": "center"}
  ],
  "rows": [
    [
      {"content": "Value 1"},
      {"content": "Value 2"}
    ]
  ]
}
```

**Fields:**
- `type`: "table"
- `header`: Array of header cells with alignment
- `rows`: Array of arrays (table rows)

**Cell Fields:**
- `content`: Cell text content
- `alignment`: "left", "center", "right", or null

#### 6. Blockquote

```json
{
  "type": "blockquote",
  "content": "This is a quoted text",
  "elements": []
}
```

**Fields:**
- `type`: "blockquote"
- `content`: Quote text content
- `elements`: Optional nested elements

#### 7. Thematic Break (Horizontal Rule)

```json
{
  "type": "thematic_break"
}
```

#### 8. Link

```json
{
  "type": "link",
  "url": "https://example.com",
  "title": "Link Title",
  "text": "Click here"
}
```

**Fields:**
- `type`: "link"
- `url`: Target URL
- `title`: Optional link title
- `text`: Link text

#### 9. Image

```json
{
  "type": "image",
  "url": "https://example.com/image.png",
  "title": "Image Title",
  "alt": "Alternative text"
}
```

**Fields:**
- `type`: "image"
- `url`: Image URL
- `title`: Optional image title
- `alt`: Alt text

## Complete Example

### Input Markdown

```markdown
# Getting Started

Welcome to **Gendocs**! This tool helps you generate documentation.

## Features

- Easy to use
- Powerful features
- Flexible configuration

### Installation

```bash
go install github.com/user/gendocs@latest
```

## Quick Start

1. Install Gendocs
2. Run `gendocs init`
3. Generate docs

| Option | Default | Description |
|:-------|:-------:|------------:|
| output | docs   | Output directory |
| format | html   | Export format |

> **Note:** This is important information.

[Visit GitHub](https://github.com/user/gendocs)
```

### Output JSON (simplified)

```json
{
  "metadata": {
    "title": "Getting Started",
    "generated_at": "2025-12-29T10:30:00Z",
    "generator": {
      "name": "Gendocs",
      "version": "1.0.0",
      "url": "https://github.com/user/gendocs"
    },
    "source_file": "README.md"
  },
  "content": {
    "headings": [
      {
        "id": "getting-started",
        "level": 1,
        "text": "Getting Started",
        "children": [
          {
            "id": "features",
            "level": 2,
            "text": "Features",
            "children": [
              {
                "id": "installation",
                "level": 3,
                "text": "Installation",
                "children": []
              }
            ]
          },
          {
            "id": "quick-start",
            "level": 2,
            "text": "Quick Start",
            "children": []
          }
        ]
      }
    ],
    "elements": [
      {"type": "heading", "level": 1, "text": "Getting Started"},
      {"type": "paragraph", "content": "Welcome to **Gendocs**! This tool helps you generate documentation."},
      {"type": "heading", "level": 2, "text": "Features"},
      {
        "type": "list",
        "list_type": "unordered",
        "items": [
          {"content": "Easy to use", "items": []},
          {"content": "Powerful features", "items": []},
          {"content": "Flexible configuration", "items": []}
        ]
      },
      {"type": "heading", "level": 3, "text": "Installation"},
      {
        "type": "code_block",
        "language": "bash",
        "code": "go install github.com/user/gendocs@latest\n",
        "lines": 1
      },
      {"type": "heading", "level": 2, "text": "Quick Start"},
      {
        "type": "list",
        "list_type": "ordered",
        "start": 1,
        "items": [
          {"content": "Install Gendocs", "items": []},
          {"content": "Run `gendocs init`", "items": []},
          {"content": "Generate docs", "items": []}
        ]
      },
      {
        "type": "table",
        "header": [
          {"content": "Option", "alignment": "left"},
          {"content": "Default", "alignment": "center"},
          {"content": "Description", "alignment": "right"}
        ],
        "rows": [
          [
            {"content": "output"},
            {"content": "docs"},
            {"content": "Output directory"}
          ],
          [
            {"content": "format"},
            {"content": "html"},
            {"content": "Export format"}
          ]
        ]
      },
      {
        "type": "blockquote",
        "content": "**Note:** This is important information.",
        "elements": []
      },
      {
        "type": "link",
        "url": "https://github.com/user/gendocs",
        "title": "",
        "text": "Visit GitHub"
      }
    ]
  }
}
```

## Usage Examples

### Generate Table of Contents

```javascript
const fs = require('fs');
const doc = JSON.parse(fs.readFileSync('docs.json', 'utf8'));

function generateTOC(headings, level = 0) {
  const indent = '  '.repeat(level);
  return headings.map(h => {
    const children = h.children.length > 0
      ? '\n' + generateTOC(h.children, level + 1)
      : '';
    return `${indent}- [${h.text}](#${h.id})${children}`;
  }).join('\n');
}

console.log(generateTOC(doc.content.headings));
```

### Extract All Code Blocks

```bash
# Extract all Go code blocks
jq '.content.elements[] | select(.type == "code_block" and .language == "go")' docs.json

# Extract code with language info
jq '.content.elements[] | select(.type == "code_block") | {language, lines: .lines}' docs.json
```

### Convert to HTML

```python
import json

def element_to_html(element):
    if element['type'] == 'paragraph':
        return f"<p>{element['content']}</p>"
    elif element['type'] == 'heading':
        return f"<h{element['level']}>{element['text']}</h{element['level']}>"
    elif element['type'] == 'code_block':
        return f"<pre><code>{element['code']}</code></pre>"
    # ... handle other types
    return ''

with open('docs.json') as f:
    doc = json.load(f)

html = '\n'.join(element_to_html(el) for el in doc['content']['elements'])
print(html)
```

### Index for Search

```javascript
const doc = JSON.parse(fs.readFileSync('docs.json', 'utf8'));

const index = {
  title: doc.metadata.title,
  url: doc.metadata.source_file,
  sections: doc.content.headings.map(h => ({
    title: h.text,
    anchor: h.id,
    level: h.level
  })),
  content: doc.content.elements
    .filter(el => el.type === 'paragraph')
    .map(el => el.content)
    .join(' ')
};

// Send to search engine
indexInSearchEngine(index);
```

### Count Words by Section

```python
import json

def count_words_in_section(elements, start_idx, end_idx):
    word_count = 0
    for el in elements[start_idx:end_idx]:
        if el['type'] == 'paragraph':
            word_count += len(el['content'].split())
    return word_count

with open('docs.json') as f:
    doc = json.load(f)

# Simple section counting based on heading positions
elements = doc['content']['elements']
heading_positions = [i for i, el in enumerate(elements) if el['type'] == 'heading']

for i, pos in enumerate(heading_positions):
    end_pos = heading_positions[i + 1] if i + 1 < len(heading_positions) else len(elements)
    heading = elements[pos]
    words = count_words_in_section(elements, pos + 1, end_pos)
    print(f"{heading['text']}: {words} words")
```

## Best Practices

### 1. Validate JSON Structure

Always validate the JSON output before processing:

```bash
# Validate JSON syntax
jq empty docs.json

# Check for required fields
jq '.metadata.title' docs.json
```

### 2. Handle Missing Fields

Some fields are optional. Always check before accessing:

```javascript
// Good
const wordCount = doc.metadata.word_count || 0;

// Bad - will error if undefined
const wordCount = doc.metadata.word_count;
```

### 3. Use Version Information

Check the generator version to ensure compatibility:

```javascript
if (doc.metadata.generator.version !== '1.0.0') {
  console.warn('Version mismatch, expected 1.0.0');
}
```

### 4. Preserve IDs

Heading IDs are URL-safe and unique. Use them for:
- Anchor links in HTML
- Fragment identifiers
- Reference keys

### 5. Handle Nested Structures

Lists and headings can be deeply nested. Use recursive functions:

```javascript
function processList(items, depth = 0) {
  items.forEach(item => {
    console.log(`${'  '.repeat(depth)}- ${item.content}`);
    if (item.items) {
      processList(item.items, depth + 1);
    }
  });
}
```

## Validation Rules

The JSON output follows these rules:

1. **Required Fields**: `metadata.title`, `metadata.generated_at`, `metadata.generator`
2. **Heading Levels**: Values are 1-6
3. **List Types**: Values are "unordered", "ordered", or "task"
4. **Cell Alignment**: Values are "left", "center", "right", or null
5. **Timestamp Format**: ISO 8601 (RFC 3339)
6. **Unique IDs**: Heading IDs are unique within a document

## Troubleshooting

### Empty headings array

**Issue**: No headings in output

**Cause**: Document has no markdown headings

**Solution**: Ensure document uses `#` for headings, not underlines

### Missing word count

**Issue**: `word_count` field not present

**Cause**: Word counting is optional and may not be calculated

**Solution**: Use `doc.metadata.word_count || 0` for safe access

### Malformed JSON

**Issue**: JSON parsing fails

**Cause**: Export process error or file corruption

**Solution**:
```bash
# Validate JSON
jq . docs.json

# Re-export
gendocs generate export --input README.md --output docs.json --format json
```

## Performance Considerations

- **File Size**: JSON files can be 2-3x larger than source markdown
- **Parsing**: Modern JSON parsers are fast (<100ms for typical docs)
- **Memory**: For very large documents (>10MB), consider streaming parsers

## Advanced Features

### Future Extensibility

The JSON structure is designed for forward compatibility:

- New fields can be added without breaking existing parsers
- New element types will be added with clear `type` identifiers
- Unknown element types can be safely ignored

### Custom Processing

You can transform the JSON to suit your needs:

```javascript
// Flatten heading hierarchy
const flatHeadings = [];
function flatten(headings) {
  headings.forEach(h => {
    flatHeadings.push({id: h.id, text: h.text, level: h.level});
    if (h.children) flatten(h.children);
  });
}
flatten(doc.content.headings);
```

## See Also

- [Export Guide](./EXPORT.md) - General export documentation
- [JSON Structure Design](./.auto-claude/specs/008-add-json-exporter-for-structured-documentation-dat/json_structure_design.md) - Detailed technical design
- [Example Output](./.auto-claude/specs/008-add-json-exporter-for-structured-documentation-dat/example_output.json) - Complete example
