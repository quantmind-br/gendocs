package llm

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestSSEParser_SimpleEvent(t *testing.T) {
	input := "data: hello world\n\n"
	parser := NewSSEParser(strings.NewReader(input))

	event, err := parser.NextEvent()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if event.Event != "" {
		t.Errorf("Expected empty event type, got '%s'", event.Event)
	}

	if string(event.Data) != "hello world" {
		t.Errorf("Expected data 'hello world', got '%s'", string(event.Data))
	}

	// Should return EOF on next call
	_, err = parser.NextEvent()
	if err != io.EOF {
		t.Errorf("Expected EOF, got %v", err)
	}
}

func TestSSEParser_WithEventType(t *testing.T) {
	input := "event: message_start\ndata: {\"type\":\"message_start\"}\n\n"
	parser := NewSSEParser(strings.NewReader(input))

	event, err := parser.NextEvent()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if event.Event != "message_start" {
		t.Errorf("Expected event type 'message_start', got '%s'", event.Event)
	}

	if string(event.Data) != `{"type":"message_start"}` {
		t.Errorf("Expected data '{\"type\":\"message_start\"}', got '%s'", string(event.Data))
	}
}

func TestSSEParser_MultipleEvents(t *testing.T) {
	input := "event: message_start\ndata: start\n\nevent: content_block\ndata: chunk1\n\ndata: chunk2\n\n"
	parser := NewSSEParser(strings.NewReader(input))

	// First event
	event, err := parser.NextEvent()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if event.Event != "message_start" {
		t.Errorf("Expected event type 'message_start', got '%s'", event.Event)
	}
	if string(event.Data) != "start" {
		t.Errorf("Expected data 'start', got '%s'", string(event.Data))
	}

	// Second event
	event, err = parser.NextEvent()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if event.Event != "content_block" {
		t.Errorf("Expected event type 'content_block', got '%s'", event.Event)
	}
	if string(event.Data) != "chunk1" {
		t.Errorf("Expected data 'chunk1', got '%s'", string(event.Data))
	}

	// Third event
	event, err = parser.NextEvent()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if event.Event != "" {
		t.Errorf("Expected empty event type, got '%s'", event.Event)
	}
	if string(event.Data) != "chunk2" {
		t.Errorf("Expected data 'chunk2', got '%s'", string(event.Data))
	}

	// Should return EOF on next call
	_, err = parser.NextEvent()
	if err != io.EOF {
		t.Errorf("Expected EOF, got %v", err)
	}
}

func TestSSEParser_MultipleDataLines(t *testing.T) {
	input := "data: line1\ndata: line2\ndata: line3\n\n"
	parser := NewSSEParser(strings.NewReader(input))

	event, err := parser.NextEvent()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := "line1\nline2\nline3"
	if string(event.Data) != expected {
		t.Errorf("Expected data '%s', got '%s'", expected, string(event.Data))
	}
}

func TestSSEParser_WithEventID(t *testing.T) {
	input := "id: 123\nevent: message\ndata: test\n\n"
	parser := NewSSEParser(strings.NewReader(input))

	event, err := parser.NextEvent()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if event.ID != "123" {
		t.Errorf("Expected ID '123', got '%s'", event.ID)
	}

	if event.Event != "message" {
		t.Errorf("Expected event type 'message', got '%s'", event.Event)
	}

	if string(event.Data) != "test" {
		t.Errorf("Expected data 'test', got '%s'", string(event.Data))
	}
}

func TestSSEParser_CommentLines(t *testing.T) {
	input := ": this is a comment\ndata: actual data\n\n"
	parser := NewSSEParser(strings.NewReader(input))

	event, err := parser.NextEvent()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if string(event.Data) != "actual data" {
		t.Errorf("Expected data 'actual data', got '%s'", string(event.Data))
	}
}

func TestSSEParser_EmptyLinesBetweenEvents(t *testing.T) {
	input := "data: event1\n\n\ndata: event2\n\n"
	parser := NewSSEParser(strings.NewReader(input))

	// First event
	event, err := parser.NextEvent()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if string(event.Data) != "event1" {
		t.Errorf("Expected data 'event1', got '%s'", string(event.Data))
	}

	// Second event
	event, err = parser.NextEvent()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if string(event.Data) != "event2" {
		t.Errorf("Expected data 'event2', got '%s'", string(event.Data))
	}
}

func TestSSEParser_LeadingSpaceInValue(t *testing.T) {
	input := "data:  hello\n\n"
	parser := NewSSEParser(strings.NewReader(input))

	event, err := parser.NextEvent()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Per SSE spec: "If the line starts with a U+003E COLON character (':'), ignore the line."
	// "If the line contains a U+003A COLON character character (':'), collect the field name and field value..."
	// "If value starts with a U+0020 SPACE character, remove it from the value."
	// So for "data:  hello" (two spaces after colon), we remove ONE space, leaving " hello"
	if string(event.Data) != " hello" {
		t.Errorf("Expected data ' hello', got '%s'", string(event.Data))
	}
}

func TestSSEParser_RetryField(t *testing.T) {
	input := "retry: 10000\ndata: test\n\n"
	parser := NewSSEParser(strings.NewReader(input))

	event, err := parser.NextEvent()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Retry field should be ignored (just verify parsing doesn't break)
	if string(event.Data) != "test" {
		t.Errorf("Expected data 'test', got '%s'", string(event.Data))
	}
}

func TestSSEParser_CRLFLineEndings(t *testing.T) {
	input := "data: line1\r\ndata: line2\r\n\r\n"
	t.Logf("Input bytes: %v", []byte(input))
	parser := NewSSEParser(strings.NewReader(input))

	event, err := parser.NextEvent()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := "line1\nline2"
	if string(event.Data) != expected {
		t.Errorf("Expected data '%s', got '%s'", expected, string(event.Data))
	}
}

func TestSSEParser_UnexpectedEOF(t *testing.T) {
	input := "data: incomplete event"
	parser := NewSSEParser(strings.NewReader(input))

	_, err := parser.NextEvent()
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	// Since we haven't successfully parsed any data yet, we get EOF, not wrapped UnexpectedEOF
	// The data "incomplete event" was never added to the buffer because there was no newline
	if err != io.EOF {
		t.Errorf("Expected EOF, got %v", err)
	}
}

func TestSSEParser_UnexpectedEOFWithData(t *testing.T) {
	input := "event: test\ndata: partial data"
	parser := NewSSEParser(strings.NewReader(input))

	_, err := parser.NextEvent()
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	// The error is wrapped, so we need to use errors.Is or check the message
	if !errors.Is(err, io.ErrUnexpectedEOF) && err.Error() != "stream ended mid-event: unexpected EOF" {
		t.Errorf("Expected ErrUnexpectedEOF (or wrapped), got %v", err)
	}
}

func TestSSEParser_EmptyStream(t *testing.T) {
	input := ""
	parser := NewSSEParser(strings.NewReader(input))

	_, err := parser.NextEvent()
	if err != io.EOF {
		t.Errorf("Expected EOF, got %v", err)
	}
}

func TestSSEParser_OnlyEmptyLines(t *testing.T) {
	input := "\n\n\n"
	parser := NewSSEParser(strings.NewReader(input))

	_, err := parser.NextEvent()
	if err != io.EOF {
		t.Errorf("Expected EOF, got %v", err)
	}
}

func TestSSEParser_OnlyComments(t *testing.T) {
	input := ": comment1\n: comment2\n\n"
	parser := NewSSEParser(strings.NewReader(input))

	_, err := parser.NextEvent()
	if err != io.EOF {
		t.Errorf("Expected EOF, got %v", err)
	}
}

func TestSSEParser_LineWithoutColon(t *testing.T) {
	input := "invalid line without colon\ndata: valid\n\n"
	parser := NewSSEParser(strings.NewReader(input))

	event, err := parser.NextEvent()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Invalid line should be ignored, only valid data should be returned
	if string(event.Data) != "valid" {
		t.Errorf("Expected data 'valid', got '%s'", string(event.Data))
	}
}

func TestSSEParser_ComplexEvent(t *testing.T) {
	input := `id: msg-123
event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}
data:  world
retry: 1000

`
	parser := NewSSEParser(strings.NewReader(input))

	event, err := parser.NextEvent()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if event.ID != "msg-123" {
		t.Errorf("Expected ID 'msg-123', got '%s'", event.ID)
	}

	if event.Event != "content_block_delta" {
		t.Errorf("Expected event type 'content_block_delta', got '%s'", event.Event)
	}

	expectedData := `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}
 world`
	if string(event.Data) != expectedData {
		t.Errorf("Expected data '%s', got '%s'", expectedData, string(event.Data))
	}
}

func TestIsSSEDone(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{
			name:     "OpenAI DONE marker",
			data:     []byte("[DONE]"),
			expected: true,
		},
		{
			name:     "Regular data",
			data:     []byte(`{"type":"message"}`),
			expected: false,
		},
		{
			name:     "Empty data",
			data:     []byte(""),
			expected: false,
		},
		{
			name:     "Almost DONE",
			data:     []byte("[DONE]extra"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSSEDone(tt.data)
			if result != tt.expected {
				t.Errorf("IsSSEDone(%q) = %v, want %v", string(tt.data), result, tt.expected)
			}
		})
	}
}

func TestSSEParser_LargeEvent(t *testing.T) {
	// Build a large event with multiple data lines
	var buf bytes.Buffer
	for i := 0; i < 100; i++ {
		buf.WriteString("data: ")
		buf.WriteString(strings.Repeat("text", 100))
		buf.WriteString("\n")
	}
	buf.WriteString("\n")

	parser := NewSSEParser(&buf)

	event, err := parser.NextEvent()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify we got all the data
	expectedLines := 100
	actualLines := strings.Count(string(event.Data), "\n") + 1
	if actualLines != expectedLines {
		t.Errorf("Expected %d lines, got %d", expectedLines, actualLines)
	}

	// Should return EOF on next call
	_, err = parser.NextEvent()
	if err != io.EOF {
		t.Errorf("Expected EOF, got %v", err)
	}
}

func TestSSEParser_EventWithOnlyID(t *testing.T) {
	input := "id: 456\n\n"
	parser := NewSSEParser(strings.NewReader(input))

	event, err := parser.NextEvent()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if event.ID != "456" {
		t.Errorf("Expected ID '456', got '%s'", event.ID)
	}

	if len(event.Data) != 0 {
		t.Errorf("Expected empty data, got '%s'", string(event.Data))
	}
}

func TestSSEParser_EventWithOnlyEvent(t *testing.T) {
	input := "event: custom_event\n\n"
	parser := NewSSEParser(strings.NewReader(input))

	event, err := parser.NextEvent()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if event.Event != "custom_event" {
		t.Errorf("Expected event type 'custom_event', got '%s'", event.Event)
	}

	if len(event.Data) != 0 {
		t.Errorf("Expected empty data, got '%s'", string(event.Data))
	}
}
