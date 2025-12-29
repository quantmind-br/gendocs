# Complete Example: JSON Export Features

This document demonstrates all the features supported by the JSON exporter.

## Introduction

The JSON exporter converts **Markdown** documents into *structured* JSON format for programmatic processing.

### Why Use JSON Export?

- **Search indexing**: Feed documentation into search engines
- **API integration**: Process docs with custom tools
- **Data analysis**: Analyze documentation structure
- **Content migration**: Convert between documentation systems

## Text Formatting

This paragraph shows **bold text**, *italic text*, `inline code`, and [links](https://example.com).

You can also combine **bold and *italic* together**, or use `code` in the middle of sentences.

## Lists

### Unordered List

- First item
- Second item
  - Nested item
  - Another nested item
- Third item

### Ordered List

1. First step
2. Second step
   1. Nested step 1
   2. Nested step 2
3. Third step

### Task List

- [x] Completed task
- [ ] Incomplete task
- [x] Another completed task
  - [x] Nested completed task
  - [ ] Nested incomplete task

## Code Blocks

### Go Example

```go
func main() {
    fmt.Println("Hello, World!")
}
```

### Python Example

```python
def hello():
    print("Hello, World!")
```

### Bash Example

```bash
echo "Hello, World!"
```

### Without Language

```
This is a code block without language specification.
```

## Tables

| Feature | Status | Priority | Notes |
|:--------|:------:|--------:|-------|
| JSON Export | ✅ Done | High | Initial implementation |
| HTML Export | ✅ Done | High | Production ready |
| PDF Export | 🚧 WIP | Medium | Planned for v2.0 |

## Blockquotes

> This is a simple blockquote.
>
> It can span multiple lines.

> **Note:** Blockquotes can contain formatting like **bold** and `code`.

> Nested blockquotes are supported:
>
>> This is nested inside another blockquote.

## Links and Images

### Different Link Types

- Inline link: [GitHub](https://github.com)
- Reference link: [GitLab][1]
- Autolink: https://example.com

[1]: https://gitlab.com

### Images

Inline image with alt text:

![Example Image](https://example.com/image.png)

Image with title:

![Code Screenshot](https://example.com/screenshot.png "Screenshot of the code")

## Horizontal Rules

---

Above is a thematic break (horizontal rule).

## Advanced Features

### Inline Formatting Combinations

You can use **bold with `code` inside**, or *italic with [links](https://example.com) embedded*, or even ***all three together***.

### Special Characters

The exporter handles special characters:
- Unicode: café, naïve, 日本語
- Math: E = mc², ∑, ∫
- Currency: $100, €50, £30
- Symbols: ©, ®, ™, ❤️

## Deep Nesting

Lists can be deeply nested:

- Level 1
  - Level 2
    - Level 3
      - Level 4
    - Back to Level 3
  - Back to Level 2

## Code Examples with Special Cases

### String with quotes

```javascript
const message = "He said, \"Hello!\"";
```

### Multiline string

```python
text = """
This is a
multiline string
in Python
"""
```

## Final Notes

The JSON export preserves:
- Document structure and hierarchy
- All markdown element types
- Metadata (title, timestamps, word counts)
- Heading hierarchy for navigation

For more information, see the [JSON Format Documentation](./JSON_FORMAT.md).
