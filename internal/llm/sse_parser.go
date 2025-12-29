package llm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// SSEEvent represents a single Server-Sent Event
type SSEEvent struct {
	Event string // Event type (optional, empty if not specified)
	Data  []byte // Concatenated data lines
	ID    string // Event ID (optional)
}

// SSEParser parses Server-Sent Events (SSE) streams
type SSEParser struct {
	reader    *bufio.Reader
	buffer    *bytes.Buffer // Accumulates data for the current event
	eventType string        // Current event type
	eventID   string        // Current event ID
}

// NewSSEParser creates a new SSE parser
func NewSSEParser(reader io.Reader) *SSEParser {
	return &SSEParser{
		reader: bufio.NewReader(reader),
		buffer: &bytes.Buffer{},
	}
}

// NextEvent reads the next SSE event from the stream
// Returns io.EOF when the stream is complete
// Returns io.ErrUnexpectedEOF if the stream ends mid-event
func (p *SSEParser) NextEvent() (SSEEvent, error) {
	debug := false // Set to true to enable debug logging
	for {
		line, err := p.reader.ReadBytes('\n')
		if err != nil {
			// If we have buffered data, the stream ended unexpectedly
			if p.buffer.Len() > 0 || p.eventType != "" {
				return SSEEvent{}, fmt.Errorf("stream ended mid-event: %w", io.ErrUnexpectedEOF)
			}
			return SSEEvent{}, err
		}

		if debug {
			fmt.Printf("DEBUG: ReadBytes returned: %v\n", line)
		}

		// Remove trailing \n first (from ReadBytes delimiter)
		line = bytes.TrimSuffix(line, []byte{'\n'})
		// Then remove trailing \r if present (Windows line endings CRLF)
		line = bytes.TrimSuffix(line, []byte{'\r'})

		if debug {
			fmt.Printf("DEBUG: After trimming: %v (len=%d)\n", line, len(line))
		}

		// Empty line marks the end of an event
		if len(line) == 0 {
			// If we have accumulated data, an event type, or an ID, return the event
			if p.buffer.Len() > 0 || p.eventType != "" || p.eventID != "" {
				event := SSEEvent{
					Event: p.eventType,
					Data:  p.buffer.Bytes(),
					ID:    p.eventID,
				}
				// Reset for next event
				p.reset()
				return event, nil
			}
			// Otherwise, continue to next line
			continue
		}

		// Skip comments (lines starting with ':')
		if len(line) > 0 && line[0] == ':' {
			continue
		}

		// Parse field
		if idx := bytes.IndexByte(line, ':'); idx != -1 {
			field := string(line[:idx])
			value := string(line[idx+1:])

			// Remove leading space from value if present (SSE spec)
			if len(value) > 0 && value[0] == ' ' {
				value = value[1:]
			}

			switch field {
			case "event":
				p.eventType = value
			case "data":
				if p.buffer.Len() > 0 {
					p.buffer.WriteByte('\n')
				}
				p.buffer.WriteString(value)
			case "id":
				p.eventID = value
			case "retry":
				// Ignore retry field for now
				// Could be used to configure reconnection time
			}
		}
		// If no colon found, ignore the line (per SSE spec)
	}
}

// reset clears the parser state for the next event
func (p *SSEParser) reset() {
	p.buffer.Reset()
	p.eventType = ""
	p.eventID = ""
}

// ParseSSEData parses JSON data from an SSE event
// This is a helper to avoid importing json in the parser package
func ParseSSEData(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// IsSSEDone checks if the SSE data is the OpenAI [DONE] marker
func IsSSEDone(data []byte) bool {
	return bytes.Equal(data, []byte("[DONE]"))
}
