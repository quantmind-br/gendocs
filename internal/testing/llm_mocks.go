package testing

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func WriteSSE(w http.ResponseWriter, event, data string) {
	if event != "" {
		fmt.Fprintf(w, "event: %s\n", event)
	}
	fmt.Fprintf(w, "data: %s\n\n", data)
}

func WriteSSEDone(w http.ResponseWriter) {
	fmt.Fprintln(w, "data: [DONE]")
	fmt.Fprintln(w)
}

func SetSSEHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
}

func SetJSONHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
}

type MockServerOption func(*mockServerConfig)

type mockServerConfig struct {
	validateAuth bool
	authHeader   string
	authValue    string
}

func WithAuthValidation(header, value string) MockServerOption {
	return func(cfg *mockServerConfig) {
		cfg.validateAuth = true
		cfg.authHeader = header
		cfg.authValue = value
	}
}

func NewMockServer(t *testing.T, handler http.HandlerFunc, opts ...MockServerOption) *httptest.Server {
	t.Helper()
	cfg := &mockServerConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	wrappedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cfg.validateAuth {
			if r.Header.Get(cfg.authHeader) != cfg.authValue {
				t.Errorf("Expected %s header '%s', got '%s'", cfg.authHeader, cfg.authValue, r.Header.Get(cfg.authHeader))
			}
		}
		handler(w, r)
	})

	return httptest.NewServer(wrappedHandler)
}

func UnauthorizedHandler(errorBody string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(errorBody))
	}
}

func RateLimitHandler(errorBody string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(errorBody))
	}
}

func InternalErrorHandler(errorBody string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(errorBody))
	}
}

func OpenAIStreamChunk(content string, finishReason string) string {
	fr := "null"
	if finishReason != "" {
		fr = fmt.Sprintf(`"%s"`, finishReason)
	}
	deltaContent := ""
	if content != "" {
		deltaContent = fmt.Sprintf(`"content":"%s"`, content)
	}
	return fmt.Sprintf(`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{%s},"finish_reason":%s}]}`, deltaContent, fr)
}

func OpenAIToolCallChunk(index int, id, name, args string) string {
	idPart := ""
	if id != "" {
		idPart = fmt.Sprintf(`"id":"%s","type":"function",`, id)
	}
	namePart := ""
	if name != "" {
		namePart = fmt.Sprintf(`"name":"%s",`, name)
	}
	return fmt.Sprintf(`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"tool_calls":[{"index":%d,%s"function":{%s"arguments":"%s"}}]},"finish_reason":null}]}`, index, idPart, namePart, strings.ReplaceAll(args, `"`, `\"`))
}

func OpenAIStreamHandler(content string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		SetSSEHeaders(w)
		WriteSSE(w, "", OpenAIStreamChunk("", ""))
		WriteSSE(w, "", OpenAIStreamChunk(content, ""))
		WriteSSE(w, "", OpenAIStreamChunk("", "stop"))
		WriteSSEDone(w)
	}
}

func AnthropicMessageStart(inputTokens int) string {
	return fmt.Sprintf(`{"type":"message_start","message":{"id":"msg_123","type":"message","role":"assistant","content":[],"model":"claude-3-sonnet","stop_reason":null,"usage":{"input_tokens":%d,"output_tokens":0}}}`, inputTokens)
}

func AnthropicContentBlockStart(index int, blockType string) string {
	if blockType == "text" {
		return fmt.Sprintf(`{"type":"content_block_start","index":%d,"content_block":{"type":"text","text":""}}`, index)
	}
	return fmt.Sprintf(`{"type":"content_block_start","index":%d,"content_block":{"type":"tool_use","id":"toolu_123","name":"","input":null}}`, index)
}

func AnthropicTextDelta(index int, text string) string {
	return fmt.Sprintf(`{"type":"content_block_delta","index":%d,"delta":{"type":"text_delta","text":"%s"}}`, index, text)
}

func AnthropicContentBlockStop(index int) string {
	return fmt.Sprintf(`{"type":"content_block_stop","index":%d}`, index)
}

func AnthropicMessageDelta(stopReason string, outputTokens int) string {
	return fmt.Sprintf(`{"type":"message_delta","delta":{"stop_reason":"%s","stop_sequence":null},"usage":{"output_tokens":%d}}`, stopReason, outputTokens)
}

func AnthropicMessageStop() string {
	return `{"type":"message_stop"}`
}

func AnthropicStreamHandler(content string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		SetSSEHeaders(w)
		WriteSSE(w, "message_start", AnthropicMessageStart(10))
		WriteSSE(w, "content_block_start", AnthropicContentBlockStart(0, "text"))
		WriteSSE(w, "content_block_delta", AnthropicTextDelta(0, content))
		WriteSSE(w, "content_block_stop", AnthropicContentBlockStop(0))
		WriteSSE(w, "message_delta", AnthropicMessageDelta("end_turn", 5))
		WriteSSE(w, "message_stop", AnthropicMessageStop())
	}
}

func GeminiChunk(text string, finishReason string, inputTokens, outputTokens int) string {
	fr := "null"
	if finishReason != "" {
		fr = fmt.Sprintf(`"%s"`, finishReason)
	}
	textPart := ""
	if text != "" {
		textPart = fmt.Sprintf(`{"text":"%s"}`, text)
	}
	return fmt.Sprintf(`{"candidates":[{"content":{"parts":[%s],"role":"model"},"finishReason":%s,"index":0}],"usageMetadata":{"promptTokenCount":%d,"candidatesTokenCount":%d,"totalTokenCount":%d}}`, textPart, fr, inputTokens, outputTokens, inputTokens+outputTokens)
}

func GeminiStreamHandler(content string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		SetJSONHeaders(w)
		w.Write([]byte(fmt.Sprintf(`[%s,%s]`, GeminiChunk(content, "", 10, 5), GeminiChunk("", "STOP", 10, 6))))
	}
}

type RetryHandler struct {
	callCount      int
	failUntil      int
	failStatusCode int
	failBody       string
	successHandler http.HandlerFunc
}

func NewRetryHandler(failUntil, failStatusCode int, failBody string, successHandler http.HandlerFunc) *RetryHandler {
	return &RetryHandler{
		failUntil:      failUntil,
		failStatusCode: failStatusCode,
		failBody:       failBody,
		successHandler: successHandler,
	}
}

func (h *RetryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.callCount++
	if h.callCount <= h.failUntil {
		w.WriteHeader(h.failStatusCode)
		w.Write([]byte(h.failBody))
		return
	}
	h.successHandler(w, r)
}

func (h *RetryHandler) CallCount() int {
	return h.callCount
}
