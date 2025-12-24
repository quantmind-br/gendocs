package validation

import (
	"fmt"
	"strings"
)

// MarkdownValidator validates Markdown content for common issues
type MarkdownValidator struct{}

// NewMarkdownValidator creates a new Markdown validator
func NewMarkdownValidator() *MarkdownValidator {
	return &MarkdownValidator{}
}

// ValidationError represents a Markdown validation error
type ValidationError struct {
	Line    int
	Message string
}

func (e *ValidationError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("line %d: %s", e.Line, e.Message)
	}
	return e.Message
}

// ValidationResult contains all validation errors
type ValidationResult struct {
	Errors []ValidationError
}

// IsValid returns true if there are no validation errors
func (vr *ValidationResult) IsValid() bool {
	return len(vr.Errors) == 0
}

// Error returns a combined error message
func (vr *ValidationResult) Error() string {
	if vr.IsValid() {
		return ""
	}

	var messages []string
	for _, err := range vr.Errors {
		messages = append(messages, err.Error())
	}
	return fmt.Sprintf("markdown validation failed:\n  - %s", strings.Join(messages, "\n  - "))
}

// Validate checks Markdown for common issues
func (v *MarkdownValidator) Validate(content string) error {
	result := &ValidationResult{
		Errors: []ValidationError{},
	}

	// Check for unclosed code blocks
	if err := v.checkCodeBlocks(content); err != nil {
		result.Errors = append(result.Errors, ValidationError{Message: err.Error()})
	}

	// Check for malformed headers
	if errs := v.checkHeaders(content); len(errs) > 0 {
		result.Errors = append(result.Errors, errs...)
	}

	// Check for minimum structure
	if err := v.checkMinimumStructure(content); err != nil {
		result.Errors = append(result.Errors, ValidationError{Message: err.Error()})
	}

	// Check for unclosed brackets/parentheses in links
	if errs := v.checkLinks(content); len(errs) > 0 {
		result.Errors = append(result.Errors, errs...)
	}

	if !result.IsValid() {
		return result
	}

	return nil
}

// checkCodeBlocks validates code block markers
func (v *MarkdownValidator) checkCodeBlocks(content string) error {
	// Count triple backticks
	openCount := strings.Count(content, "```")
	if openCount%2 != 0 {
		return fmt.Errorf("unclosed code block detected (%d ``` markers, expected even number)", openCount)
	}

	// Check for mixed code block styles (shouldn't mix ``` and ~~~)
	tripleBacktick := strings.Count(content, "```")
	tripleTilde := strings.Count(content, "~~~")

	if tripleBacktick > 0 && tripleTilde > 0 {
		// This is valid but could be confusing
		// Just a warning, not an error
	}

	return nil
}

// checkHeaders validates header formatting
func (v *MarkdownValidator) checkHeaders(content string) []ValidationError {
	var errors []ValidationError
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Extract hash marks
		hashCount := 0
		for _, ch := range trimmed {
			if ch == '#' {
				hashCount++
			} else {
				break
			}
		}

		// Valid headers: # to ######
		if hashCount > 6 {
			errors = append(errors, ValidationError{
				Line:    i + 1,
				Message: fmt.Sprintf("too many # symbols (%d, max 6)", hashCount),
			})
			continue
		}

		// Check for space after hash marks
		if len(trimmed) > hashCount {
			nextChar := trimmed[hashCount]
			if nextChar != ' ' && nextChar != '#' {
				errors = append(errors, ValidationError{
					Line:    i + 1,
					Message: "missing space after # in header",
				})
			}
		}

		// Check for empty header
		headerText := strings.TrimSpace(trimmed[hashCount:])
		if headerText == "" {
			errors = append(errors, ValidationError{
				Line:    i + 1,
				Message: "empty header (no text after #)",
			})
		}
	}

	return errors
}

// checkMinimumStructure ensures basic Markdown requirements
func (v *MarkdownValidator) checkMinimumStructure(content string) error {
	trimmed := strings.TrimSpace(content)

	// Check length
	if len(trimmed) < 50 {
		return fmt.Errorf("content too short (%d characters, minimum 50)", len(trimmed))
	}

	// Check for at least one header
	if !strings.Contains(content, "#") {
		return fmt.Errorf("no headers found (expected at least one # header)")
	}

	return nil
}

// checkLinks validates link formatting
func (v *MarkdownValidator) checkLinks(content string) []ValidationError {
	var errors []ValidationError
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		// Check for unmatched brackets in links
		openBracket := strings.Count(line, "[")
		closeBracket := strings.Count(line, "]")
		openParen := strings.Count(line, "(")
		closeParen := strings.Count(line, ")")

		// Check for any unmatched brackets
		if openBracket != closeBracket {
			errors = append(errors, ValidationError{
				Line:    i + 1,
				Message: fmt.Sprintf("unmatched brackets ([ count: %d, ] count: %d)", openBracket, closeBracket),
			})
		}

		// Check for markdown link pattern [text](url)
		if strings.Contains(line, "](") {
			// This looks like a link, check parens
			if openParen != closeParen {
				errors = append(errors, ValidationError{
					Line:    i + 1,
					Message: fmt.Sprintf("unmatched parentheses in link (( count: %d, ) count: %d)", openParen, closeParen),
				})
			}
		}
	}

	return errors
}

// ValidateAndFix attempts to fix common issues
func (v *MarkdownValidator) ValidateAndFix(content string) (string, error) {
	// Apply common fixes first
	fixed := content

	// Fix: Add newline at end if missing
	if !strings.HasSuffix(fixed, "\n") {
		fixed += "\n"
	}

	// Fix: Remove trailing whitespace
	lines := strings.Split(fixed, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	fixed = strings.Join(lines, "\n")

	// Validate after fixes
	if err := v.Validate(fixed); err != nil {
		return fixed, fmt.Errorf("could not auto-fix: %w", err)
	}

	return fixed, nil
}

// QuickCheck performs a fast validation with minimal checks
func (v *MarkdownValidator) QuickCheck(content string) bool {
	// Quick checks only
	if len(strings.TrimSpace(content)) < 50 {
		return false
	}

	if !strings.Contains(content, "#") {
		return false
	}

	// Check code blocks
	if strings.Count(content, "```")%2 != 0 {
		return false
	}

	return true
}
