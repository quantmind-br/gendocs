# Gendocs - Local LLM Setup (Ollama & LM Studio)

## Overview
Gendocs supports local LLM providers by treating them as OpenAI-compatible clients but with specific configuration defaults.

## Supported Providers
- **Ollama**: Default URL `http://localhost:11434/v1`
- **LM Studio**: Default URL `http://localhost:1234/v1`

## Implementation Details

### Configuration (`internal/tui/dashboard/sections/llm.go`)
- **BaseURL Auto-population**: Selecting "ollama" or "lmstudio" in the TUI automatically sets the `BaseURL`.
- **API Key**: The API Key field is marked as "optional/not required" for local providers and validation is skipped.
- **Model Placeholder**: Shows provider-specific examples (e.g., `llama3`, `codellama`).

### LLM Client (`internal/llm/factory.go`)
- **Routing**: `ollama` and `lmstudio` providers are routed to the `OpenAIClient` implementation.
- **Compatibility**: They use the OpenAI API format (chat completions).

## Testing
- **Unit Tests**: `internal/tui/dashboard/sections/llm_test.go` covers TUI behavior (dropdown, defaults).
- **Factory Tests**: `internal/llm/factory_test.go` verifies client creation.
- **Mocking**: Tests use `httptest.NewServer` to mock local LLM endpoints.

## Setup Instructions
1. Install provider (Ollama/LM Studio).
2. Pull/Download model.
3. Start server (Ollama: `ollama serve`, LM Studio: "Start Server").
4. In Gendocs TUI: Select provider -> BaseURL auto-fills -> Enter model name.
