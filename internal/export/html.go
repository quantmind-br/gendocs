package export

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

// HTMLExporter converts Markdown to standalone HTML documents
type HTMLExporter struct {
	markdown     goldmark.Markdown
	htmlTemplate *template.Template
}

// HTMLDocument represents the data for HTML template rendering
type HTMLDocument struct {
	Title   string
	Content template.HTML
	CSS     template.CSS // Use template.CSS to mark as safe
}

// NewHTMLExporter creates a new HTML exporter with Goldmark configured
func NewHTMLExporter() (*HTMLExporter, error) {
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
	)

	// Load HTML template
	tmpl, err := loadHTMLTemplate()
	if err != nil {
		return nil, fmt.Errorf("failed to load HTML template: %w", err)
	}

	return &HTMLExporter{
		markdown:     md,
		htmlTemplate: tmpl,
	}, nil
}

// ExportToHTML converts a Markdown file to a standalone HTML file
func (e *HTMLExporter) ExportToHTML(markdownPath, outputPath string) error {
	// Read Markdown file
	mdContent, err := os.ReadFile(markdownPath)
	if err != nil {
		return fmt.Errorf("failed to read markdown: %w", err)
	}

	// Convert Markdown to HTML
	var buf bytes.Buffer
	if err := e.markdown.Convert(mdContent, &buf); err != nil {
		return fmt.Errorf("failed to convert markdown: %w", err)
	}

	// Extract title from first H1
	title := extractTitle(string(mdContent))

	// Render full HTML document
	doc := HTMLDocument{
		Title:   title,
		Content: template.HTML(buf.String()),
		CSS:     template.CSS(getDefaultCSS()), // Mark CSS as safe
	}

	var htmlBuf bytes.Buffer
	if err := e.htmlTemplate.Execute(&htmlBuf, doc); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	// Write output file
	if err := os.WriteFile(outputPath, htmlBuf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write HTML: %w", err)
	}

	return nil
}

// loadHTMLTemplate loads the HTML template with custom functions
func loadHTMLTemplate() (*template.Template, error) {
	const tmpl = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="generator" content="Gendocs">
    <title>{{.Title}}</title>
    <style>
        {{.CSS}}
    </style>
</head>
<body>
    <div class="container">
        <header>
            <div class="generator-badge">Generated with Gendocs</div>
        </header>
        <main>
            {{.Content}}
        </main>
        <footer>
            <p>Generated on {{now}} by <a href="https://github.com/user/gendocs">Gendocs</a></p>
        </footer>
    </div>
</body>
</html>`

	return template.New("html").Funcs(template.FuncMap{
		"now": func() string {
			return time.Now().Format("2006-01-02 15:04:05")
		},
	}).Parse(tmpl)
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

// getDefaultCSS returns GitHub-style CSS for the HTML document
func getDefaultCSS() string {
	return `
        * {
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, Arial, sans-serif;
            line-height: 1.6;
            color: #24292f;
            background-color: #ffffff;
            margin: 0;
            padding: 0;
        }

        .container {
            max-width: 980px;
            margin: 0 auto;
            padding: 45px;
        }

        header {
            border-bottom: 1px solid #d0d7de;
            margin-bottom: 30px;
            padding-bottom: 10px;
        }

        .generator-badge {
            font-size: 12px;
            color: #57606a;
            text-align: right;
        }

        main {
            margin-bottom: 60px;
        }

        h1, h2, h3, h4, h5, h6 {
            margin-top: 24px;
            margin-bottom: 16px;
            font-weight: 600;
            line-height: 1.25;
        }

        h1 {
            font-size: 2em;
            border-bottom: 1px solid #d0d7de;
            padding-bottom: 0.3em;
        }

        h2 {
            font-size: 1.5em;
            border-bottom: 1px solid #d0d7de;
            padding-bottom: 0.3em;
        }

        code {
            background-color: rgba(175, 184, 193, 0.2);
            border-radius: 6px;
            font-size: 85%;
            margin: 0;
            padding: 0.2em 0.4em;
            font-family: ui-monospace, SFMono-Regular, 'SF Mono', Menlo, Consolas, monospace;
        }

        pre {
            background-color: #f6f8fa;
            border-radius: 6px;
            font-size: 85%;
            line-height: 1.45;
            overflow: auto;
            padding: 16px;
        }

        pre code {
            background-color: transparent;
            border: 0;
            display: inline;
            line-height: inherit;
            margin: 0;
            overflow: visible;
            padding: 0;
            word-wrap: normal;
        }

        table {
            border-collapse: collapse;
            border-spacing: 0;
            width: 100%;
            margin-bottom: 16px;
        }

        table th {
            font-weight: 600;
            background-color: #f6f8fa;
        }

        table th, table td {
            padding: 6px 13px;
            border: 1px solid #d0d7de;
        }

        table tr:nth-child(2n) {
            background-color: #f6f8fa;
        }

        a {
            color: #0969da;
            text-decoration: none;
        }

        a:hover {
            text-decoration: underline;
        }

        blockquote {
            padding: 0 1em;
            color: #57606a;
            border-left: 0.25em solid #d0d7de;
            margin: 0 0 16px;
        }

        ul, ol {
            padding-left: 2em;
            margin-top: 0;
            margin-bottom: 16px;
        }

        li + li {
            margin-top: 0.25em;
        }

        footer {
            border-top: 1px solid #d0d7de;
            padding-top: 20px;
            text-align: center;
            font-size: 14px;
            color: #57606a;
        }

        @media (max-width: 768px) {
            .container {
                padding: 15px;
            }

            h1 {
                font-size: 1.6em;
            }

            h2 {
                font-size: 1.3em;
            }
        }
    `
}
