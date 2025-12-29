package agents

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/user/gendocs/internal/llm"
	"github.com/user/gendocs/internal/logging"
	"github.com/user/gendocs/internal/prompts"
	"github.com/user/gendocs/internal/tools"
)

// Context management constants
const (
	// MaxConversationTokens is the maximum estimated tokens allowed in conversation history
	MaxConversationTokens = 100000

	// MaxToolResponseTokens is the maximum tokens for a single tool response
	MaxToolResponseTokens = 15000

	// TokenEstimateRatio is the approximate characters per token (for estimation)
	TokenEstimateRatio = 4
)

// Agent is the interface that all agents must implement
type Agent interface {
	// Run executes the agent and returns the generated output
	Run(ctx context.Context) (string, error)

	// Name returns the agent name
	Name() string
}

// BaseAgent provides common functionality for all agents
type BaseAgent struct {
	name          string
	llmClient     llm.LLMClient
	tools         []tools.Tool
	promptManager *prompts.Manager
	logger        *logging.Logger
	systemPrompt  string
	maxRetries    int
	maxTokens     int
	temperature   float64
}

// NewBaseAgent creates a new base agent
func NewBaseAgent(
	name string,
	llmClient llm.LLMClient,
	tools []tools.Tool,
	promptManager *prompts.Manager,
	logger *logging.Logger,
	systemPrompt string,
	maxRetries int,
) *BaseAgent {
	return &BaseAgent{
		name:          name,
		llmClient:     llmClient,
		tools:         tools,
		promptManager: promptManager,
		logger:        logger,
		systemPrompt:  systemPrompt,
		maxRetries:    maxRetries,
		maxTokens:     8192,
		temperature:   0.0,
	}
}

// SetMaxTokens sets the maximum tokens for LLM responses
func (ba *BaseAgent) SetMaxTokens(maxTokens int) {
	ba.maxTokens = maxTokens
}

// SetTemperature sets the temperature for LLM responses
func (ba *BaseAgent) SetTemperature(temperature float64) {
	ba.temperature = temperature
}

// RunOnce executes the agent once with the given user prompt
func (ba *BaseAgent) RunOnce(ctx context.Context, userPrompt string) (string, error) {
	// Initialize conversation history with the user prompt
	// This ensures the prompt is preserved across all iterations
	conversationHistory := []llm.Message{
		{Role: "user", Content: userPrompt},
	}

	// Maximum iterations to prevent infinite loops
	const maxIterations = 100
	iterations := 0

	// Tool calling loop
	for {
		iterations++
		if iterations > maxIterations {
			ba.logger.Warn("Maximum iterations reached, forcing completion",
				logging.String("agent", ba.name),
				logging.Int("iterations", iterations),
			)
			return "", fmt.Errorf("agent exceeded maximum iterations (%d)", maxIterations)
		}

		// Trim conversation history to prevent context overflow
		conversationHistory = trimConversationHistory(conversationHistory, MaxConversationTokens)

		// Log current context size
		currentTokens := estimateHistoryTokens(conversationHistory)
		ba.logger.Info("Calling LLM",
			logging.String("agent", ba.name),
			logging.Int("tool_count", len(ba.tools)),
			logging.Int("history_messages", len(conversationHistory)),
			logging.Int("estimated_tokens", currentTokens),
		)

		req := llm.CompletionRequest{
			SystemPrompt: ba.systemPrompt,
			Messages:     conversationHistory,
			Tools:        ba.convertTools(),
			MaxTokens:    ba.maxTokens,
			Temperature:  ba.temperature,
		}

		// Call LLM
		resp, err := ba.llmClient.GenerateCompletion(ctx, req)
		if err != nil {
			return "", fmt.Errorf("LLM call failed: %w", err)
		}

		ba.logger.Info("LLM response received",
			logging.String("agent", ba.name),
			logging.Int("input_tokens", resp.Usage.InputTokens),
			logging.Int("output_tokens", resp.Usage.OutputTokens),
			logging.Int("tool_calls", len(resp.ToolCalls)),
		)

		// If no tool calls, return content
		if len(resp.ToolCalls) == 0 {
			return resp.Content, nil
		}

		// Add assistant response to conversation history (including tool calls)
		conversationHistory = append(conversationHistory, llm.Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		// Execute tool calls
		for _, toolCall := range resp.ToolCalls {
			tool := ba.findTool(toolCall.Name)
			if tool == nil {
				ba.logger.Warn("Tool not found", logging.String("tool", toolCall.Name))
				// Add error response
				conversationHistory = append(conversationHistory, llm.Message{
					Role:    "tool",
					Content: fmt.Sprintf("Error: Tool '%s' not found", toolCall.Name),
					ToolID:  toolCall.Name,
				})
				continue
			}

			ba.logger.Info("Executing tool",
				logging.String("tool", tool.Name()),
				logging.String("agent", ba.name),
			)

			// Execute tool
			result, err := tool.Execute(ctx, toolCall.Arguments)
			if err != nil {
				ba.logger.Error("Tool execution failed",
					logging.String("tool", tool.Name()),
					logging.Error(err),
				)
				conversationHistory = append(conversationHistory, llm.Message{
					Role:    "tool",
					Content: fmt.Sprintf("Error: %v", err),
					ToolID:  toolCall.Name,
				})
			} else {
				// Format and truncate tool response
				formattedResult := formatToolResult(result)
				truncatedResult := truncateToolResponse(formattedResult, MaxToolResponseTokens)

				conversationHistory = append(conversationHistory, llm.Message{
					Role:    "tool",
					Content: truncatedResult,
					ToolID:  toolCall.Name,
				})
			}
		}

		// Continue loop to get final response from LLM
	}
}

// convertTools converts agent tools to LLM tool definitions
func (ba *BaseAgent) convertTools() []llm.ToolDefinition {
	var toolDefs []llm.ToolDefinition
	for _, tool := range ba.tools {
		toolDefs = append(toolDefs, llm.ToolDefinition{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  tool.Parameters(),
		})
	}
	return toolDefs
}

// findTool finds a tool by name
func (ba *BaseAgent) findTool(name string) tools.Tool {
	for _, tool := range ba.tools {
		if tool.Name() == name {
			return tool
		}
	}
	return nil
}

// Name returns the agent name
func (ba *BaseAgent) Name() string {
	return ba.name
}

// estimateTokens estimates the number of tokens in a string
func estimateTokens(text string) int {
	return len(text) / TokenEstimateRatio
}

// estimateHistoryTokens estimates total tokens in conversation history
func estimateHistoryTokens(history []llm.Message) int {
	total := 0
	for _, msg := range history {
		total += estimateTokens(msg.Content)
	}
	return total
}

// trimConversationHistory keeps conversation history within token limits
// It removes older messages while preserving the most recent context
func trimConversationHistory(history []llm.Message, maxTokens int) []llm.Message {
	if len(history) == 0 {
		return history
	}

	totalTokens := estimateHistoryTokens(history)

	// If within limits, return as is
	if totalTokens <= maxTokens {
		return history
	}

	// Remove older messages from the beginning, keeping at least the last 4 messages
	// (typically: assistant response, tool result, assistant response, tool result)
	minKeep := 4
	if len(history) < minKeep {
		minKeep = len(history)
	}

	trimmed := history
	for len(trimmed) > minKeep && estimateHistoryTokens(trimmed) > maxTokens {
		trimmed = trimmed[1:]
	}

	// If still too large, truncate individual messages
	if estimateHistoryTokens(trimmed) > maxTokens {
		for i := range trimmed {
			if trimmed[i].Role == "tool" && len(trimmed[i].Content) > MaxToolResponseTokens*TokenEstimateRatio {
				// Truncate tool responses that are too large
				maxChars := MaxToolResponseTokens * TokenEstimateRatio
				trimmed[i].Content = trimmed[i].Content[:maxChars] + "\n[TRUNCATED - response exceeded token limit]"
			}
		}
	}

	return trimmed
}

// truncateToolResponse truncates a tool response if it exceeds the limit
func truncateToolResponse(response string, maxTokens int) string {
	maxChars := maxTokens * TokenEstimateRatio
	if len(response) <= maxChars {
		return response
	}

	return response[:maxChars] + "\n\n[TRUNCATED - Tool response exceeded " + fmt.Sprintf("%d", maxTokens) + " token limit]"
}

// formatToolResult formats a tool result for inclusion in conversation history
func formatToolResult(result interface{}) string {
	// Try to marshal as JSON for cleaner output
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return fmt.Sprintf("%v", result)
	}
	return string(jsonBytes)
}
