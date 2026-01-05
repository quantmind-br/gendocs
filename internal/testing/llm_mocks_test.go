package testing

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestOpenAIStreamHandler(t *testing.T) {
	server := NewMockServer(t, OpenAIStreamHandler("Hello, world!"))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	content := string(body)

	if !strings.Contains(content, "Hello, world!") {
		t.Errorf("Expected response to contain 'Hello, world!', got: %s", content)
	}
	if !strings.Contains(content, "[DONE]") {
		t.Errorf("Expected response to contain '[DONE]', got: %s", content)
	}
}

func TestAnthropicStreamHandler(t *testing.T) {
	server := NewMockServer(t, AnthropicStreamHandler("Test response"))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	content := string(body)

	if !strings.Contains(content, "Test response") {
		t.Errorf("Expected response to contain 'Test response', got: %s", content)
	}
	if !strings.Contains(content, "message_stop") {
		t.Errorf("Expected response to contain 'message_stop', got: %s", content)
	}
}

func TestGeminiStreamHandler(t *testing.T) {
	server := NewMockServer(t, GeminiStreamHandler("Gemini response"))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	content := string(body)

	if !strings.Contains(content, "Gemini response") {
		t.Errorf("Expected response to contain 'Gemini response', got: %s", content)
	}
	if !strings.Contains(content, "STOP") {
		t.Errorf("Expected response to contain 'STOP', got: %s", content)
	}
}

func TestRetryHandler(t *testing.T) {
	handler := NewRetryHandler(2, http.StatusTooManyRequests, `{"error":"rate limit"}`,
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		})

	server := NewMockServer(t, handler.ServeHTTP)
	defer server.Close()

	for i := 1; i <= 3; i++ {
		resp, err := http.Get(server.URL)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}
		defer resp.Body.Close()

		if i <= 2 {
			if resp.StatusCode != http.StatusTooManyRequests {
				t.Errorf("Request %d: expected 429, got %d", i, resp.StatusCode)
			}
		} else {
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Request %d: expected 200, got %d", i, resp.StatusCode)
			}
		}
	}

	if handler.CallCount() != 3 {
		t.Errorf("Expected 3 calls, got %d", handler.CallCount())
	}
}

func TestWithAuthValidation(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	server := NewMockServer(t, handler, WithAuthValidation("Authorization", "Bearer test-key"))
	defer server.Close()

	req, _ := http.NewRequest("GET", server.URL, nil)
	req.Header.Set("Authorization", "Bearer test-key")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}
