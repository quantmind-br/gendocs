package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/user/gendocs/internal/config"
)

// GeminiClient implements LLMClient for Google Gemini
type GeminiClient struct {
	*BaseLLMClient
	apiKey  string
	model   string
	baseURL string
}

// geminiRequest represents the request body for Gemini API
type geminiRequest struct {
	Contents       []geminiContent    `json:"contents"`
	Tools          []geminiTool       `json:"tools,omitempty"`
	GenerationConfig geminiGenerationConfig `json:"generationConfig,omitempty"`
	SystemInstruction *geminiContent  `json:"systemInstruction,omitempty"`
}

// geminiContent represents content in Gemini format
type geminiContent struct {
	Role  string           `json:"role,omitempty"`
	Parts []geminiPart     `json:"parts"`
}

// geminiPart represents a part of content
type geminiPart struct {
	Text             string                  `json:"text,omitempty"`
	FunctionCall     map[string]interface{}  `json:"functionCall,omitempty"`
	FunctionResponse *geminiFunctionResponse `json:"functionResponse,omitempty"`
	ThoughtSignature string                  `json:"thoughtSignature,omitempty"` // Required for Gemini 3 function calling
}

// geminiFunctionResponse represents a function response
// Gemini format: {"name": "function_name", "response": {...}}
type geminiFunctionResponse struct {
	Name     string                 `json:"name"`
	Response map[string]interface{} `json:"response,omitempty"`
}

// geminiTool represents a tool declaration
type geminiTool struct {
	FunctionDeclarations []geminiFunctionDeclaration `json:"functionDeclarations,omitempty"`
}

// geminiFunctionDeclaration represents a function declaration
type geminiFunctionDeclaration struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// geminiGenerationConfig represents generation configuration
type geminiGenerationConfig struct {
	Temperature float64 `json:"temperature,omitempty"`
	MaxOutputTokens int  `json:"maxOutputTokens,omitempty"`
}

// geminiResponse represents the response from Gemini API
type geminiResponse struct {
	Candidates []geminiCandidate `json:"candidates"`
	UsageMetadata geminiUsageMetadata `json:"usageMetadata,omitempty"`
	Error      *geminiError      `json:"error,omitempty"`
}

// geminiCandidate represents a candidate response
type geminiCandidate struct {
	Content   geminiContent `json:"content"`
	FinishReason string     `json:"finishReason,omitempty"`
}

// geminiUsageMetadata represents token usage
type geminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// geminiError represents an error
type geminiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// geminiStreamChunk represents a single streaming response chunk (NDJSON line)
type geminiStreamChunk struct {
	Candidates     []geminiStreamCandidate `json:"candidates"`
	UsageMetadata  *geminiUsage            `json:"usageMetadata,omitempty"`
	Error          *geminiError            `json:"error,omitempty"`
}

// geminiStreamCandidate represents a candidate in a streaming chunk
type geminiStreamCandidate struct {
	Content      geminiStreamContent `json:"content"`
	FinishReason *string             `json:"finishReason,omitempty"`
	Index        int                 `json:"index"`
}

// geminiStreamContent represents content in a streaming chunk
type geminiStreamContent struct {
	Parts []geminiStreamPart `json:"parts"`
	Role  string             `json:"role,omitempty"`
}

// geminiStreamPart represents a part in streaming content
type geminiStreamPart struct {
	Text         string                      `json:"text,omitempty"`
	FunctionCall *geminiStreamFunctionCall   `json:"functionCall,omitempty"`
}

// geminiStreamFunctionCall represents a function call in streaming
type geminiStreamFunctionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

// geminiAccumulator builds CompletionResponse from streaming chunks
type geminiAccumulator struct {
	textBuilder strings.Builder
	toolCalls   []ToolCall
	usage       geminiUsage
	finishReason string
	complete    bool
}

// newGeminiAccumulator creates a new accumulator
func newGeminiAccumulator() *geminiAccumulator {
	return &geminiAccumulator{}
}

// HandleChunk processes a single streaming chunk
func (a *geminiAccumulator) HandleChunk(chunk geminiStreamChunk) error {
	// Check for API error
	if chunk.Error != nil {
		return fmt.Errorf("API error: %s", chunk.Error.Message)
	}

	// Skip if no candidates
	if len(chunk.Candidates) == 0 {
		return nil
	}

	candidate := chunk.Candidates[0]

	// Check for safety block
	if candidate.FinishReason != nil && *candidate.FinishReason == "SAFETY" {
		return fmt.Errorf("response blocked for safety reasons")
	}

	// Accumulate usage metadata if present
	if chunk.UsageMetadata != nil {
		a.usage = *chunk.UsageMetadata
	}

	// Process parts
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			a.textBuilder.WriteString(part.Text)
		}
		if part.FunctionCall != nil {
			// Function calls arrive complete in Gemini (no partial JSON)
			a.toolCalls = append(a.toolCalls, ToolCall{
				Name:      part.FunctionCall.Name,
				Arguments: part.FunctionCall.Args,
				RawFunctionCall: map[string]interface{}{
					"name": part.FunctionCall.Name,
					"args": part.FunctionCall.Args,
				},
			})
		}
	}

	// Check if complete (finishReason is set)
	if candidate.FinishReason != nil {
		a.finishReason = *candidate.FinishReason
		a.complete = true
	}

	return nil
}

// Build constructs the final CompletionResponse
func (a *geminiAccumulator) Build() CompletionResponse {
	return CompletionResponse{
		Content: a.textBuilder.String(),
		ToolCalls: a.toolCalls,
		Usage: TokenUsage{
			InputTokens:  a.usage.PromptTokenCount,
			OutputTokens: a.usage.CandidatesTokenCount,
			TotalTokens:  a.usage.TotalTokenCount,
		},
	}
}

// IsComplete returns true if finishReason was received
func (a *geminiAccumulator) IsComplete() bool {
	return a.complete
}

// NewGeminiClient creates a new Gemini client
func NewGeminiClient(cfg config.LLMConfig, retryClient *RetryClient) *GeminiClient {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com"
	}
	return &GeminiClient{
		BaseLLMClient: NewBaseLLMClient(retryClient),
		apiKey:        cfg.APIKey,
		model:         cfg.Model,
		baseURL:       baseURL,
	}
}

// GenerateCompletion generates a completion from Gemini
func (c *GeminiClient) GenerateCompletion(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	// Convert to Gemini format
	gemReq := c.convertRequest(req)

	jsonData, err := json.Marshal(gemReq)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	// Model format: models/gemini-1.5-pro or models/gemini-pro
	// Use streaming endpoint: streamGenerateContent
	modelName := c.model
	if !strings.HasPrefix(modelName, "models/") {
		modelName = "models/" + modelName
	}
	url := fmt.Sprintf("%s/v1beta/%s:streamGenerateContent?key=%s", c.baseURL, modelName, c.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Execute with retry
	resp, err := c.retryClient.Do(httpReq)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check for error status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return CompletionResponse{}, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse streaming response
	return c.parseStreamingResponse(resp.Body)
}

// parseStreamingResponse parses Gemini's NDJSON stream and builds the response
func (c *GeminiClient) parseStreamingResponse(body io.ReadCloser) (CompletionResponse, error) {
	scanner := bufio.NewScanner(body)
	accumulator := newGeminiAccumulator()

	for scanner.Scan() {
		line := scanner.Bytes()

		// Skip empty lines
		if len(line) == 0 {
			continue
		}

		// Parse JSON line (NDJSON format)
		var chunk geminiStreamChunk
		if err := json.Unmarshal(line, &chunk); err != nil {
			return CompletionResponse{}, fmt.Errorf("failed to parse stream chunk: %w", err)
		}

		// Handle chunk
		if err := accumulator.HandleChunk(chunk); err != nil {
			return CompletionResponse{}, fmt.Errorf("chunk handling error: %w", err)
		}

		// Check if stream is complete
		if accumulator.IsComplete() {
			break
		}
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		return CompletionResponse{}, fmt.Errorf("stream reading error: %w", err)
	}

	return accumulator.Build(), nil
}

// SupportsTools returns true
func (c *GeminiClient) SupportsTools() bool {
	return true
}

// GetProvider returns the provider name
func (c *GeminiClient) GetProvider() string {
	return "gemini"
}

// convertRequest converts internal request to Gemini format
func (c *GeminiClient) convertRequest(req CompletionRequest) geminiRequest {
	// Build contents
	contents := []geminiContent{}

	// Add system instruction as first content with role "user"
	// Gemini doesn't have a separate system field, it's part of content
	if req.SystemPrompt != "" {
		contents = append(contents, geminiContent{
			Role: "user",
			Parts: []geminiPart{
				{Text: req.SystemPrompt},
			},
		})
		// Add empty model response
		contents = append(contents, geminiContent{
			Role: "model",
			Parts: []geminiPart{
				{Text: "Understood. I will analyze the codebase according to your instructions."},
			},
		})
	}

	// Add messages
	for _, msg := range req.Messages {
		if msg.Role == "tool" {
			// Tool response - extract function name from tool ID or content
			// Format: {"name": "function_name", "response": {"result": "content"}}
			funcName := msg.ToolID
			if funcName == "" {
				// Try to extract from Content if it's JSON
				var toolData map[string]interface{}
				if err := json.Unmarshal([]byte(msg.Content), &toolData); err == nil {
					if name, ok := toolData["name"].(string); ok {
						funcName = name
					}
				}
			}
			// Fallback to a default name if still empty
			if funcName == "" {
				funcName = "unknown_function"
			}

			contents = append(contents, geminiContent{
				Role: "user",
				Parts: []geminiPart{
					{
						FunctionResponse: &geminiFunctionResponse{
							Name: funcName,
							Response: map[string]interface{}{
								"result": msg.Content,
							},
						},
					},
				},
			})
		} else if msg.Role == "assistant" {
			// Model/assistant message - include function calls if present
			var parts []geminiPart

			// Add text content if present
			if msg.Content != "" {
				parts = append(parts, geminiPart{Text: msg.Content})
			}

			// Add function calls if present - include ThoughtSignature for Gemini 3
			for _, tc := range msg.ToolCalls {
				part := geminiPart{
					ThoughtSignature: tc.ThoughtSignature, // Include thought signature at part level
				}
				if tc.RawFunctionCall != nil {
					part.FunctionCall = tc.RawFunctionCall
				} else {
					part.FunctionCall = map[string]interface{}{
						"name": tc.Name,
						"args": tc.Arguments,
					}
				}
				parts = append(parts, part)
			}

			// Only add the message if there are parts (text or function calls)
			if len(parts) > 0 {
				contents = append(contents, geminiContent{
					Role:  "model",
					Parts: parts,
				})
			}
		} else if msg.Role == "user" {
			// User message - skip empty content
			if msg.Content == "" {
				continue
			}
			contents = append(contents, geminiContent{
				Role: "user",
				Parts: []geminiPart{
					{Text: msg.Content},
				},
			})
		}
	}

	// Build tools
	var tools []geminiTool
	if len(req.Tools) > 0 {
		tools = make([]geminiTool, 1)
		functions := make([]geminiFunctionDeclaration, len(req.Tools))
		for i, tool := range req.Tools {
			functions[i] = geminiFunctionDeclaration{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			}
		}
		tools[0] = geminiTool{
			FunctionDeclarations: functions,
		}
	}

	return geminiRequest{
		Contents: contents,
		Tools:    tools,
		GenerationConfig: geminiGenerationConfig{
			Temperature:    req.Temperature,
			MaxOutputTokens: req.MaxTokens,
		},
	}
}

// convertResponse converts Gemini response to internal format
func (c *GeminiClient) convertResponse(resp geminiResponse) CompletionResponse {
	result := CompletionResponse{
		Usage: TokenUsage{
			InputTokens:  resp.UsageMetadata.PromptTokenCount,
			OutputTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:  resp.UsageMetadata.TotalTokenCount,
		},
	}

	if len(resp.Candidates) == 0 {
		return result
	}

	candidate := resp.Candidates[0]
	var textContent string
	var toolCalls []ToolCall

	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			textContent += part.Text
		}
		if part.FunctionCall != nil {
			name, _ := part.FunctionCall["name"].(string)
			args, _ := part.FunctionCall["args"].(map[string]interface{})
			toolCalls = append(toolCalls, ToolCall{
				Name:             name,
				Arguments:        args,
				RawFunctionCall:  part.FunctionCall,
				ThoughtSignature: part.ThoughtSignature, // Capture thought signature from part level
			})
		}
	}

	result.Content = textContent
	result.ToolCalls = toolCalls

	return result
}
