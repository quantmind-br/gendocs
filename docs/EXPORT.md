# Documentation Export

This guide explains how to export Gendocs-generated documentation to various formats for easier sharing and publishing.

## Quick Start

```bash
# Export README.md to HTML (default format)
gendocs generate export

# Export README.md to JSON
gendocs generate export --format json --output docs.json

# Export specific file to HTML
gendocs generate export --input .ai/docs/code_structure.md --output structure.html

# Generate and export in one command
gendocs generate readme --export-html
```

## Supported Formats

### HTML

Generate standalone HTML files with embedded CSS and syntax highlighting.

**Features:**
- Single-file output (no external dependencies)
- Embedded CSS for GitHub-style rendering
- Syntax highlighting for code blocks (50+ languages)
- Responsive design (mobile-friendly)
- Dark theme syntax highlighting (Monokai)
- Valid HTML5 output

**Usage:**
```bash
gendocs generate export [flags]
```

**Flags:**
- `--input string`: Input markdown file (default: "README.md")
- `--output string`: Output file path (default: input.html or input.json based on format)
- `--format string`: Export format - "html" or "json" (default: "html")
- `--repo-path string`: Path to repository (default: ".")

### JSON

Generate structured JSON with metadata and hierarchical content.

**Features:**
- Structured data format for programmatic access
- Metadata section with title, timestamps, generator info, word/character counts
- Hierarchical headings tree for navigation and table of contents
- All document elements in flat array (paragraphs, code blocks, lists, tables, blockquotes, links, images)
- Language information for code blocks
- Table column alignment information
- Task list checkbox states
- URL-safe heading IDs

**Usage:**
```bash
gendocs generate export --format json [flags]
```

**When to use JSON export:**
- **Search indexing**: Feed documentation into search engines (Elasticsearch, Algolia, Meilisearch)
- **Static site generators**: Process with custom templates (Hugo, Jekyll, Eleventy)
- **API documentation**: Generate API references from markdown
- **Documentation portals**: Integrate into existing platforms
- **Content analysis**: Analyze documentation structure and metrics
- **Content migration**: Convert between documentation systems
- **Custom processing**: Apply transformations or extract specific data

**Example output structure:**
```json
{
  "metadata": {
    "title": "Document Title",
    "generated_at": "2025-12-29T10:30:00Z",
    "generator": {
      "name": "Gendocs",
      "version": "1.0.0"
    },
    "source_file": "README.md",
    "word_count": 1234,
    "char_count": 5678
  },
  "content": {
    "headings": [
      {
        "id": "section-title",
        "level": 2,
        "text": "Section Title",
        "children": []
      }
    ],
    "elements": [
      {"type": "heading", "level": 1, "text": "Title"},
      {"type": "paragraph", "content": "Content..."},
      {"type": "code_block", "language": "go", "code": "..."}
    ]
  }
}
```

For detailed JSON structure documentation, see [JSON_FORMAT.md](JSON_FORMAT.md).

## Common Use Cases

### Export README for GitHub Pages

```bash
# Generate README and export to docs/index.html for GitHub Pages
gendocs generate readme --export-html
mv README.html docs/index.html
```

### Export All Analysis Documents

```bash
# Export each analysis document to HTML
gendocs generate export --input .ai/docs/code_structure.md --output docs/structure.html
gendocs generate export --input .ai/docs/dependencies.md --output docs/dependencies.html
gendocs generate export --input .ai/docs/data_flow.md --output docs/data-flow.html
gendocs generate export --input .ai/docs/request_flow.md --output docs/request-flow.html
gendocs generate export --input .ai/docs/api_documentation.md --output docs/api.html
```

### Export Documentation for Search Indexing

```bash
# Export README to JSON for search indexing
gendocs generate export --input README.md --output search-index.json --format json

# Export all docs to JSON for indexing
gendocs generate export --input .ai/docs/code_structure.md --output search/structure.json --format json
gendocs generate export --input .ai/docs/dependencies.md --output search/dependencies.json --format json
gendocs generate export --input .ai/docs/api_documentation.md --output search/api.json --format json
```

### Export for Static Site Generator

```bash
# Export to JSON for processing with Hugo/Jekyll/Eleventy
gendocs generate export --input README.md --output content/docs/index.json --format json

# Use with jq to extract specific data
jq '{title: .metadata.title, sections: [.content.headings[] | {id, text, level}]}' search-index.json
```

### Batch Export Script

Create a script to export all documentation:

```bash
#!/bin/bash
# export-docs.sh

FORMAT=${1:-html}  # Default to HTML if no format specified
OUTPUT_DIR="docs/$FORMAT"

echo "Exporting documentation to $FORMAT..."

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Export README
gendocs generate export --input README.md --output "$OUTPUT_DIR/index.$FORMAT" --format "$FORMAT"

# Export analysis documents
for file in .ai/docs/*.md; do
    if [ -f "$file" ]; then
        basename=$(basename "$file" .md)
        gendocs generate export --input "$file" --output "$OUTPUT_DIR/${basename}.$FORMAT" --format "$FORMAT"
        echo "  ✓ Exported $basename to ${basename}.$FORMAT"
    fi
done

echo "All documentation exported to $OUTPUT_DIR/"
```

Usage:
```bash
# Export to HTML (default)
./export-docs.sh
# or
./export-docs.sh html

# Export to JSON
./export-docs.sh json
```

### Integrate with CI/CD

#### GitHub Actions

```yaml
name: Generate Documentation

on:
  push:
    branches: [main]
  workflow_dispatch:

jobs:
  docs:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.22'

      - name: Install Gendocs
        run: |
          git clone https://github.com/user/gendocs
          cd gendocs
          go build -o gendocs .
          sudo mv gendocs /usr/local/bin/

      - name: Export Documentation
        run: |
          # Export to HTML for GitHub Pages
          gendocs generate export --output docs/index.html

          # Optionally export to JSON for search indexing
          gendocs generate export --output docs/search-index.json --format json
        env:
          DOCUMENTER_LLM_API_KEY: ${{ secrets.OPENAI_API_KEY }}
          DOCUMENTER_LLM_PROVIDER: openai
          DOCUMENTER_LLM_MODEL: gpt-4o

      - name: Deploy to GitHub Pages
        uses: peaceiris/actions-gh-pages@v3
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          publish_dir: ./docs
```

#### GitLab CI

```yaml
pages:
  stage: deploy
  script:
    - go install github.com/user/gendocs@latest
    - gendocs generate export --output public/index.html
  artifacts:
    paths:
      - public
  only:
    - main
```

## Output Customization

Currently, the HTML exporter uses a fixed GitHub-style theme. Future versions may support:

- Custom CSS themes
- Light/dark mode toggle
- Configurable syntax highlighting themes
- Custom header/footer templates

## Technical Details

### HTML Structure

The exported HTML follows this structure:

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="generator" content="Gendocs">
    <title><!-- Extracted from first H1 --></title>
    <style>
        <!-- Embedded CSS -->
    </style>
</head>
<body>
    <div class="container">
        <header>
            <div class="generator-badge">Generated with Gendocs</div>
        </header>
        <main>
            <!-- Converted Markdown content -->
        </main>
        <footer>
            <p>Generated on YYYY-MM-DD HH:MM:SS by Gendocs</p>
        </footer>
    </div>
</body>
</html>
```

### Markdown Processing

The exporter uses [Goldmark](https://github.com/yuin/goldmark) with these extensions:

- **GitHub Flavored Markdown (GFM)**: Tables, strikethrough, task lists
- **Syntax Highlighting**: [Chroma](https://github.com/alecthomas/chroma) with Monokai theme
- **Raw HTML**: Preserves raw HTML in Markdown source

Supported GFM features:
- Tables with alignment
- Strikethrough (`~~text~~`)
- Task lists (`- [x] Done`, `- [ ] Todo`)
- Autolinks
- Hard line breaks

### Syntax Highlighting

Code blocks are highlighted using Chroma with support for 50+ languages:

- Go, Python, JavaScript, TypeScript, Rust, Java, C, C++, C#
- Ruby, PHP, Swift, Kotlin, Scala, Elixir, Erlang
- Shell (bash, zsh), PowerShell, SQL, YAML, JSON, XML
- HTML, CSS, SCSS, Markdown, Dockerfile
- And many more...

Example:

````markdown
```go
func main() {
    fmt.Println("Hello, World!")
}
```
````

Renders with syntax-aware coloring for keywords, strings, comments, etc.

## Troubleshooting

### Input file not found

**Error:** `input file not found: README.md`

**Solution:** Ensure the input file exists or provide the correct path:

```bash
# Use absolute path
gendocs generate export --input /full/path/to/file.md

# Or relative to repo-path
gendocs generate export --repo-path /path/to/repo --input README.md
```

### Invalid Markdown

**Issue:** Exported HTML looks broken or incomplete

**Solution:** Validate your Markdown syntax:

```bash
# Check for common issues:
# - Unclosed code blocks (```)
# - Malformed tables
# - Invalid HTML in Markdown

# Use a Markdown linter
npm install -g markdownlint-cli
markdownlint README.md
```

### Large file export slow

**Issue:** Exporting very large Markdown files (>1MB) is slow

**Solution:**
1. Split large documents into smaller sections
2. Use table of contents with links instead of one huge file
3. Consider pagination for very long documents

### Missing syntax highlighting

**Issue:** Code blocks appear without syntax highlighting

**Solution:** Ensure language is specified in code fence:

````markdown
<!-- Wrong: no language specified -->
```
func main() {}
```

<!-- Correct: language specified -->
```go
func main() {}
```
````

### JSON Export Issues

#### Empty JSON output

**Issue:** JSON file is generated but content is empty or missing elements

**Solution:** Ensure your markdown uses proper formatting:
```bash
# Check that headings use # syntax, not underlines
# Proper: ## Heading
# Improper: Heading\n=======

# Validate JSON structure
jq . docs.json
```

#### Large JSON file size

**Issue:** JSON output is much larger than source markdown

**Explanation:** This is expected. JSON files are typically 2-3x larger due to:
- Structural overhead (quotes, brackets, field names)
- Metadata fields
- Hierarchical heading structure
- Detailed element information

**Solution:** For very large documents, consider:
```bash
# Minify JSON for production
jq -c . docs.json > docs.min.json

# Extract only needed fields
jq '{metadata, headings: .content.headings}' docs.json
```

#### Missing fields in JSON

**Issue:** Expected fields like `word_count` are not present

**Solution:** Some fields are optional. Always check before accessing:
```javascript
// Good
const wordCount = doc.metadata.word_count || 0;

// Bad - may error if undefined
const wordCount = doc.metadata.word_count;
```

#### Processing JSON with special characters

**Issue:** JSON parsing fails with Unicode or special characters

**Solution:** Ensure proper encoding:
```bash
# Use jq for safe parsing
jq . docs.json

# In Python, use encoding parameter
with open('docs.json', 'r', encoding='utf-8') as f:
    doc = json.load(f)
```

## Future Enhancements

Planned features for future versions:

- **PDF Export**: Generate PDF from HTML using headless browser
- **Multi-page Sites**: Generate full documentation sites with navigation
- **Custom Themes**: Support for custom CSS themes
- **Table of Contents**: Automatic ToC generation for long documents
- **Search**: Client-side search functionality
- **Version History**: Compare documentation across versions

## See Also

- [README.md](../README.md) - Main project documentation
- [JSON Format Guide](JSON_FORMAT.md) - Detailed JSON export structure and usage examples
- [JSON Export Examples](../examples/json-export/) - Comprehensive JSON export examples with code samples
- [Custom Prompts](../examples/custom-prompts/) - Customize analysis behavior
- [PLAN.md](../PLAN.md) - Development roadmap

## Support

For issues or questions:
- Report bugs: https://github.com/user/gendocs/issues
- Feature requests: https://github.com/user/gendocs/discussions
