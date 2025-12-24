# Documentation Export

This guide explains how to export Gendocs-generated documentation to various formats for easier sharing and publishing.

## Quick Start

```bash
# Export README.md to HTML
gendocs generate export

# Export specific file
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
- `--output string`: Output file path (default: input.html)
- `--format string`: Export format, currently only "html" (default: "html")
- `--repo-path string`: Path to repository (default: ".")

## Common Use Cases

### Export README for GitHub Pages

```bash
# Generate README and export to docs/index.html for GitHub Pages
gendocs generate readme --export-html
mv README.html docs/index.html
```

### Export All Analysis Documents

```bash
# Export each analysis document
gendocs generate export --input .ai/docs/code_structure.md --output docs/structure.html
gendocs generate export --input .ai/docs/dependencies.md --output docs/dependencies.html
gendocs generate export --input .ai/docs/data_flow.md --output docs/data-flow.html
gendocs generate export --input .ai/docs/request_flow.md --output docs/request-flow.html
gendocs generate export --input .ai/docs/api_documentation.md --output docs/api.html
```

### Batch Export Script

Create a script to export all documentation:

```bash
#!/bin/bash
# export-docs.sh

echo "Exporting documentation to HTML..."

# Create output directory
mkdir -p docs/html

# Export README
gendocs generate export --input README.md --output docs/html/index.html

# Export analysis documents
for file in .ai/docs/*.md; do
    basename=$(basename "$file" .md)
    gendocs generate export --input "$file" --output "docs/html/${basename}.html"
    echo "  ✓ Exported $basename"
done

echo "All documentation exported to docs/html/"
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
          gendocs generate export --output docs/index.html
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
- [Custom Prompts](../examples/custom-prompts/) - Customize analysis behavior
- [PLAN.md](../PLAN.md) - Development roadmap

## Support

For issues or questions:
- Report bugs: https://github.com/user/gendocs/issues
- Feature requests: https://github.com/user/gendocs/discussions
