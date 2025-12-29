# JSON Export Examples

This directory contains example markdown files and their corresponding JSON output to demonstrate the JSON export feature.

## Files

### complete-example.md / complete-example.json

A comprehensive example demonstrating all features supported by the JSON exporter:

**Features demonstrated:**
- All heading levels (H1-H6) with hierarchical structure
- Text formatting (bold, italic, inline code)
- Unordered, ordered, and task lists with nesting
- Code blocks with different language syntax highlighting
- Tables with column alignment
- Blockquotes (single and nested)
- Links (inline, reference, autolinks)
- Images with alt text and titles
- Horizontal rules (thematic breaks)
- Special characters (Unicode, math symbols, emojis)
- Deep nesting in lists
- Complex inline formatting combinations

## How to Use These Examples

### Generate JSON from Markdown

```bash
# Navigate to the project root
cd /path/to/gendocs

# Export the example markdown to JSON
gendocs generate export \
  --input examples/json-export/complete-example.md \
  --output examples/json-export/output.json \
  --format json
```

### Compare Output

You can compare your generated JSON with the provided example:

```bash
# Using jq for pretty printing
jq . examples/json-export/complete-example.json

# Or use a diff tool
diff <(jq . examples/json-export/complete-example.json) <(jq . your-output.json)
```

## Understanding the JSON Structure

The JSON output consists of two main sections:

### 1. Metadata

Contains document information:
- `title`: Document title (from first H1)
- `generated_at`: ISO 8601 timestamp
- `generator`: Name, version, and URL of the generator
- `source_file`: Original markdown filename
- `word_count`: Total word count (optional)
- `char_count`: Total character count (optional)

### 2. Content

Contains the document content in two forms:

#### Headings Array
Hierarchical tree structure for navigation:
- `id`: URL-safe unique identifier
- `level`: Heading level (1-6)
- `text`: Heading text (plain)
- `children`: Nested child headings

#### Elements Array
Flat list of all document elements in order:
- Each element has a `type` field
- Types include: `paragraph`, `heading`, `code_block`, `list`, `table`, `blockquote`, `thematic_break`, `link`, `image`
- Type-specific fields for each element

## Practical Usage Examples

### Extract All Code Blocks

```bash
# Extract Go code blocks
jq '.content.elements[] | select(.type == "code_block" and .language == "go")' \
  examples/json-export/complete-example.json
```

### Generate Table of Contents

```javascript
const fs = require('fs');
const doc = JSON.parse(fs.readFileSync('examples/json-export/complete-example.json', 'utf8'));

function generateTOC(headings, level = 0) {
  const indent = '  '.repeat(level);
  return headings.map(h => {
    const link = `${indent}- [${h.text}](#${h.id})`;
    const children = h.children.length > 0
      ? '\n' + generateTOC(h.children, level + 1)
      : '';
    return link + children;
  }).join('\n');
}

console.log(generateTOC(doc.content.headings));
```

### Count Words by Section

```python
import json

with open('examples/json-export/complete-example.json') as f:
    doc = json.load(f)

def count_words(elements, start, end):
    count = 0
    for el in elements[start:end]:
        if el['type'] == 'paragraph':
            count += len(el['content'].split())
    return count

# Count words in each section
elements = doc['content']['elements']
heading_indices = [i for i, el in enumerate(elements) if el['type'] == 'heading']

for i, idx in enumerate(heading_indices):
    end_idx = heading_indices[i + 1] if i + 1 < len(heading_indices) else len(elements)
    heading = elements[idx]
    words = count_words(elements, idx + 1, end_idx)
    print(f"{heading['text']}: {words} words")
```

### Convert to HTML

```python
import json

def element_to_html(element):
    type_map = {
        'paragraph': lambda e: f"<p>{e['content']}</p>",
        'heading': lambda e: f"<h{e['level']}>{e['text']}</h{e['level']}>",
        'code_block': lambda e: f'<pre><code class="language-{e["language"]}">{e["code"]}</code></pre>',
        'thematic_break': lambda e: '<hr>',
    }
    return type_map.get(element['type'], lambda e: '')(element)

with open('examples/json-export/complete-example.json') as f:
    doc = json.load(f)

html = '\n'.join(element_to_html(el) for el in doc['content']['elements'])
print(html)
```

## Validation

You can validate the JSON structure:

```bash
# Check if JSON is valid
jq empty examples/json-export/complete-example.json

# Verify required fields
jq '.metadata.title' examples/json-export/complete-example.json
jq '.metadata.generated_at' examples/json-export/complete-example.json
jq '.content.headings | length' examples/json-export/complete-example.json
jq '.content.elements | length' examples/json-export/complete-example.json
```

## More Information

- [JSON Format Guide](../../docs/JSON_FORMAT.md) - Complete JSON structure documentation
- [Export Guide](../../docs/EXPORT.md) - General export documentation
- [Implementation Plan](../../.auto-claude/specs/008-add-json-exporter-for-structured-documentation-dat/implementation_plan.json) - Technical implementation details

## Tips for Working with JSON Export

1. **Use jq for CLI Operations**: jq is powerful for filtering and transforming JSON
2. **Validate First**: Always validate JSON before processing
3. **Handle Optional Fields**: Some fields like `word_count` are optional
4. **Preserve IDs**: Heading IDs are URL-safe and unique - use them for anchor links
5. **Consider File Size**: JSON files are typically 2-3x larger than source markdown
6. **Use Version Info**: Check `generator.version` for compatibility

## Troubleshooting

### JSON Parsing Errors

```bash
# Validate JSON syntax
jq . examples/json-export/complete-example.json
```

### Missing Fields

Some fields are optional. Always check before accessing:

```javascript
// Good
const wordCount = doc.metadata.word_count || 0;

// Bad - may error if undefined
const wordCount = doc.metadata.word_count;
```

### Empty Headings Array

This is normal for documents without markdown headings (using `#`). Ensure your markdown uses `#` for headings, not underlines.

## Contributing

If you find issues with the JSON export or have suggestions for improvements, please:
1. Check the [implementation plan](../../.auto-claude/specs/008-add-json-exporter-for-structured-documentation-dat/implementation_plan.json)
2. Review the [JSON format documentation](../../docs/JSON_FORMAT.md)
3. Open an issue with example input/output that demonstrates the problem
