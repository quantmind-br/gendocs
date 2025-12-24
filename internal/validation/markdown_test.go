package validation

import (
	"strings"
	"testing"
)

func TestMarkdownValidator_Validate_Valid(t *testing.T) {
	validator := NewMarkdownValidator()

	validMarkdown := `# Test Document

This is a valid Markdown document with proper structure.

## Section 1

Content here with **bold** and *italic* text.

` + "```go\ncode here\n```" + `

## Section 2

More content.
`

	err := validator.Validate(validMarkdown)
	if err != nil {
		t.Errorf("Expected no error for valid markdown, got: %v", err)
	}
}

func TestMarkdownValidator_Validate_UnclosedCodeBlock(t *testing.T) {
	validator := NewMarkdownValidator()

	invalidMarkdown := `# Test

` + "```go\ncode here\n" + `

No closing backticks.
`

	err := validator.Validate(invalidMarkdown)
	if err == nil {
		t.Fatal("Expected error for unclosed code block, got nil")
	}

	if !strings.Contains(err.Error(), "unclosed code block") {
		t.Errorf("Expected 'unclosed code block' in error, got: %v", err)
	}
}

func TestMarkdownValidator_Validate_MalformedHeader(t *testing.T) {
	validator := NewMarkdownValidator()

	tests := []struct {
		name     string
		markdown string
		wantErr  bool
	}{
		{
			name:     "no space after hash",
			markdown: `#Header Without Space`,
			wantErr:  true,
		},
		{
			name:     "empty header",
			markdown: `#`,
			wantErr:  true,
		},
		{
			name:     "too many hashes",
			markdown: `####### Too Many`,
			wantErr:  true,
		},
		{
			name:     "valid header",
			markdown: `# Valid Header`,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Add minimum content
			content := tt.markdown + "\n\nSome content to meet minimum length requirement here."
			err := validator.Validate(content)

			if tt.wantErr && err == nil {
				t.Error("Expected error, got nil")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestMarkdownValidator_Validate_TooShort(t *testing.T) {
	validator := NewMarkdownValidator()

	shortMarkdown := `# Short`

	err := validator.Validate(shortMarkdown)
	if err == nil {
		t.Fatal("Expected error for too short content, got nil")
	}

	if !strings.Contains(err.Error(), "too short") {
		t.Errorf("Expected 'too short' in error, got: %v", err)
	}
}

func TestMarkdownValidator_Validate_NoHeaders(t *testing.T) {
	validator := NewMarkdownValidator()

	noHeadersMarkdown := `This is content without any headers at all.
It has multiple lines and enough content length.
But still no headers which makes it invalid.`

	err := validator.Validate(noHeadersMarkdown)
	if err == nil {
		t.Fatal("Expected error for no headers, got nil")
	}

	if !strings.Contains(err.Error(), "no headers") {
		t.Errorf("Expected 'no headers' in error, got: %v", err)
	}
}

func TestMarkdownValidator_Validate_UnmatchedBrackets(t *testing.T) {
	validator := NewMarkdownValidator()

	invalidMarkdown := `# Test

This is a [link with unmatched bracket.

More content here to meet minimum length.
`

	err := validator.Validate(invalidMarkdown)
	if err == nil {
		t.Fatal("Expected error for unmatched brackets, got nil")
	}

	if !strings.Contains(err.Error(), "bracket") {
		t.Errorf("Expected 'bracket' in error, got: %v", err)
	}
}

func TestMarkdownValidator_Validate_UnmatchedParentheses(t *testing.T) {
	validator := NewMarkdownValidator()

	invalidMarkdown := `# Test

This is a [link](https://example.com without closing paren.

More content here to meet minimum length requirement.
`

	err := validator.Validate(invalidMarkdown)
	if err == nil {
		t.Fatal("Expected error for unmatched parentheses, got nil")
	}

	if !strings.Contains(err.Error(), "parenthes") {
		t.Errorf("Expected 'parenthes' in error, got: %v", err)
	}
}

func TestMarkdownValidator_Validate_MultipleErrors(t *testing.T) {
	validator := NewMarkdownValidator()

	invalidMarkdown := `#NoSpace
` + "```go\nunclosed code block\n" + `
[unmatched bracket
`

	err := validator.Validate(invalidMarkdown)
	if err == nil {
		t.Fatal("Expected error for multiple issues, got nil")
	}

	// Should report multiple errors
	result, ok := err.(*ValidationResult)
	if !ok {
		t.Fatalf("Expected ValidationResult, got %T", err)
	}

	if len(result.Errors) < 2 {
		t.Errorf("Expected at least 2 errors, got %d", len(result.Errors))
	}
}

func TestMarkdownValidator_ValidateAndFix_AddNewline(t *testing.T) {
	validator := NewMarkdownValidator()

	markdown := `# Test Document

This is valid content but missing final newline.`

	fixed, err := validator.ValidateAndFix(markdown)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !strings.HasSuffix(fixed, "\n") {
		t.Error("Expected fixed markdown to end with newline")
	}
}

func TestMarkdownValidator_ValidateAndFix_TrimTrailingSpaces(t *testing.T) {
	validator := NewMarkdownValidator()

	markdown := `# Test Document

This is valid content with trailing spaces on line above.
And this line too.
`

	fixed, err := validator.ValidateAndFix(markdown)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	lines := strings.Split(fixed, "\n")
	for i, line := range lines {
		if strings.HasSuffix(line, " ") || strings.HasSuffix(line, "\t") {
			t.Errorf("Line %d still has trailing whitespace: %q", i+1, line)
		}
	}
}

func TestMarkdownValidator_ValidateAndFix_CannotFix(t *testing.T) {
	validator := NewMarkdownValidator()

	// Unclosed code block cannot be auto-fixed
	unfixableMarkdown := `# Test
` + "```go\ncode here\n" + `
More content.
`

	_, err := validator.ValidateAndFix(unfixableMarkdown)
	if err == nil {
		t.Fatal("Expected error for unfixable markdown, got nil")
	}

	if !strings.Contains(err.Error(), "could not auto-fix") {
		t.Errorf("Expected 'could not auto-fix' in error, got: %v", err)
	}
}

func TestMarkdownValidator_QuickCheck_Valid(t *testing.T) {
	validator := NewMarkdownValidator()

	validMarkdown := `# Test

Valid content with enough length and proper structure.
`

	if !validator.QuickCheck(validMarkdown) {
		t.Error("Expected QuickCheck to return true for valid markdown")
	}
}

func TestMarkdownValidator_QuickCheck_TooShort(t *testing.T) {
	validator := NewMarkdownValidator()

	shortMarkdown := `# Test`

	if validator.QuickCheck(shortMarkdown) {
		t.Error("Expected QuickCheck to return false for too short markdown")
	}
}

func TestMarkdownValidator_QuickCheck_NoHeaders(t *testing.T) {
	validator := NewMarkdownValidator()

	noHeaders := `This is content without headers but with enough length to pass length check.`

	if validator.QuickCheck(noHeaders) {
		t.Error("Expected QuickCheck to return false for markdown without headers")
	}
}

func TestMarkdownValidator_QuickCheck_UnclosedCodeBlock(t *testing.T) {
	validator := NewMarkdownValidator()

	unclosed := `# Test
` + "```go\ncode\n" + `
More content here.
`

	if validator.QuickCheck(unclosed) {
		t.Error("Expected QuickCheck to return false for unclosed code block")
	}
}

func TestValidationResult_IsValid(t *testing.T) {
	result := &ValidationResult{Errors: []ValidationError{}}
	if !result.IsValid() {
		t.Error("Expected IsValid to return true for empty errors")
	}

	result.Errors = append(result.Errors, ValidationError{Message: "error"})
	if result.IsValid() {
		t.Error("Expected IsValid to return false when errors exist")
	}
}

func TestValidationResult_Error(t *testing.T) {
	result := &ValidationResult{
		Errors: []ValidationError{
			{Line: 5, Message: "error on line 5"},
			{Message: "general error"},
		},
	}

	errMsg := result.Error()
	if !strings.Contains(errMsg, "line 5") {
		t.Error("Expected error message to contain 'line 5'")
	}

	if !strings.Contains(errMsg, "general error") {
		t.Error("Expected error message to contain 'general error'")
	}
}

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      ValidationError
		expected string
	}{
		{
			name:     "with line number",
			err:      ValidationError{Line: 10, Message: "test error"},
			expected: "line 10: test error",
		},
		{
			name:     "without line number",
			err:      ValidationError{Line: 0, Message: "general error"},
			expected: "general error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, tt.err.Error())
			}
		})
	}
}

func TestMarkdownValidator_Validate_ComplexDocument(t *testing.T) {
	validator := NewMarkdownValidator()

	complexMarkdown := `# Main Title

This is the introduction paragraph.

## Section 1

Content with **bold**, *italic*, and ` + "`code`" + `.

### Subsection 1.1

` + "```go\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```" + `

## Section 2

A list:
- Item 1
- Item 2
- Item 3

A link: [Example](https://example.com)

### Subsection 2.1

More content with proper formatting.

## Section 3

Final section with a table:

| Column 1 | Column 2 |
|----------|----------|
| Data 1   | Data 2   |

And that's the end.
`

	err := validator.Validate(complexMarkdown)
	if err != nil {
		t.Errorf("Expected no error for complex valid markdown, got: %v", err)
	}
}

func TestMarkdownValidator_Validate_EmptyContent(t *testing.T) {
	validator := NewMarkdownValidator()

	err := validator.Validate("")
	if err == nil {
		t.Fatal("Expected error for empty content, got nil")
	}

	if !strings.Contains(err.Error(), "too short") {
		t.Errorf("Expected 'too short' in error, got: %v", err)
	}
}

func TestMarkdownValidator_Validate_WhitespaceOnly(t *testing.T) {
	validator := NewMarkdownValidator()

	whitespace := "   \n\n\t\n   "

	err := validator.Validate(whitespace)
	if err == nil {
		t.Fatal("Expected error for whitespace-only content, got nil")
	}
}
