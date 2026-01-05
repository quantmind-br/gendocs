package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/user/gendocs/internal/config"
	"github.com/user/gendocs/internal/llmcache"
)

func TestFactory_CreateClient_AllProviders(t *testing.T) {
	tests := []struct {
		name        string
		provider    string
		expectError bool
	}{
		{"openai", "openai", false},
		{"anthropic", "anthropic", false},
		{"gemini", "gemini", false},
		{"ollama", "ollama", false},
		{"lmstudio", "lmstudio", false},
		{"unsupported", "unsupported", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := NewFactory(nil, nil, nil, false, 0)
			cfg := config.LLMConfig{
				Provider: tt.provider,
				Model:    "test-model",
				APIKey:   "test-key",
			}

			client, err := factory.CreateClient(cfg)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if client == nil {
					t.Error("Expected client but got nil")
				}
			}
		})
	}
}

func TestFactory_CreateClient_Ollama(t *testing.T) {
	factory := NewFactory(nil, nil, nil, false, 0)

	cfg := config.LLMConfig{
		Provider: "ollama",
		Model:    "llama3",
		BaseURL:  "http://localhost:11434/v1",
	}

	client, err := factory.CreateClient(cfg)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if client == nil {
		t.Fatal("Expected client to be created")
	}

	_, ok := client.(*OpenAIClient)
	if !ok {
		t.Error("Expected OpenAI client type for ollama provider")
	}
}

func TestFactory_CreateClient_LMStudio(t *testing.T) {
	factory := NewFactory(nil, nil, nil, false, 0)

	cfg := config.LLMConfig{
		Provider: "lmstudio",
		Model:    "llama3",
		BaseURL:  "http://localhost:1234/v1",
	}

	client, err := factory.CreateClient(cfg)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if client == nil {
		t.Fatal("Expected client to be created")
	}

	_, ok := client.(*OpenAIClient)
	if !ok {
		t.Error("Expected OpenAI client type for lmstudio provider")
	}
}

func TestFactory_CreateClient_OllamaWithoutAPIKey(t *testing.T) {
	factory := NewFactory(nil, nil, nil, false, 0)

	cfg := config.LLMConfig{
		Provider: "ollama",
		Model:    "llama3",
		BaseURL:  "http://localhost:11434/v1",
		APIKey:   "",
	}

	client, err := factory.CreateClient(cfg)
	if err != nil {
		t.Fatalf("Should create client without API key for ollama: %v", err)
	}

	if client == nil {
		t.Fatal("Expected client to be created")
	}
}

func TestFactory_CreateClient_ErrorMessageListsAllProviders(t *testing.T) {
	factory := NewFactory(nil, nil, nil, false, 0)
	cfg := config.LLMConfig{
		Provider: "invalid",
	}

	_, err := factory.CreateClient(cfg)
	if err == nil {
		t.Fatal("Expected error for invalid provider")
	}

	errMsg := err.Error()
	expectedProviders := []string{"openai", "anthropic", "gemini", "ollama", "lmstudio"}

	for _, p := range expectedProviders {
		if !strings.Contains(errMsg, p) {
			t.Errorf("Error message should list %q as supported provider", p)
		}
	}
}

func TestFactory_CreateClient_LocalProviderWithCaching(t *testing.T) {
	memCache := llmcache.NewLRUCache(100)
	factory := NewFactory(nil, memCache, nil, true, time.Hour)

	cfg := config.LLMConfig{
		Provider: "ollama",
		Model:    "llama3",
		BaseURL:  "http://localhost:11434/v1",
	}

	client, err := factory.CreateClient(cfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	_, ok := client.(*CachedLLMClient)
	if !ok {
		t.Error("Expected client to be wrapped in CachedLLMClient when caching enabled")
	}
}

func TestFactory_CreateClient_OllamaIntegration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1234567890,\"model\":\"llama3\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Hello\"},\"finish_reason\":null}]}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1234567890,\"model\":\"llama3\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer server.Close()

	factory := NewFactory(nil, nil, nil, false, 0)
	cfg := config.LLMConfig{
		Provider: "ollama",
		Model:    "llama3",
		BaseURL:  server.URL,
	}

	client, err := factory.CreateClient(cfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.GenerateCompletion(ctx, CompletionRequest{
		Messages:  []Message{{Role: "user", Content: "test"}},
		MaxTokens: 10,
	})

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.Content != "Hello" {
		t.Errorf("Expected 'Hello', got %q", resp.Content)
	}
}

func TestFactory_CreateClient_LMStudioIntegration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-456\",\"object\":\"chat.completion.chunk\",\"created\":1234567890,\"model\":\"llama3\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"World\"},\"finish_reason\":null}]}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "data: {\"id\":\"chatcmpl-456\",\"object\":\"chat.completion.chunk\",\"created\":1234567890,\"model\":\"llama3\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer server.Close()

	factory := NewFactory(nil, nil, nil, false, 0)
	cfg := config.LLMConfig{
		Provider: "lmstudio",
		Model:    "llama3",
		BaseURL:  server.URL,
	}

	client, err := factory.CreateClient(cfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.GenerateCompletion(ctx, CompletionRequest{
		Messages:  []Message{{Role: "user", Content: "test"}},
		MaxTokens: 10,
	})

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.Content != "World" {
		t.Errorf("Expected 'World', got %q", resp.Content)
	}
}
