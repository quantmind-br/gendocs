package llmtypes

// Message represents a chat message
type Message struct {
	Role      string // "system", "user", "assistant", "tool"
	Content   string
	ToolID    string     // ID of the tool that was called (for role="tool")
	ToolCalls []ToolCall // Tool calls made by assistant (for role="assistant")
}

// ToolCall represents a tool/function call from the LLM
type ToolCall struct {
	Name             string                 // Name of the tool to call
	Arguments        map[string]interface{} // Arguments for the tool
	RawFunctionCall  map[string]interface{} // Preserves complete function call data
	ThoughtSignature string                 // Required for Gemini 3 function calling
}

// CompletionRequest is a request for LLM completion
type CompletionRequest struct {
	SystemPrompt string
	Messages     []Message
	Tools        []ToolDefinition
	MaxTokens    int
	Temperature  float64
}

// CompletionResponse is the response from LLM
type CompletionResponse struct {
	Content   string
	ToolCalls []ToolCall
	Usage     TokenUsage
}

// TokenUsage tracks token usage
type TokenUsage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

// ToolDefinition defines a tool for the LLM
type ToolDefinition struct {
	Name        string
	Description string
	Parameters  map[string]interface{}
}
