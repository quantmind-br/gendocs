package agents

import (
	"context"
	"fmt"

	"github.com/user/gendocs/internal/llm"
	"github.com/user/gendocs/internal/logging"
	"github.com/user/gendocs/internal/prompts"
	"github.com/user/gendocs/internal/tools"
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
	// Build request
	messages := []llm.Message{
		{Role: "user", Content: userPrompt},
	}

	var conversationHistory []llm.Message

	// Tool calling loop
	for {
		req := llm.CompletionRequest{
			SystemPrompt: ba.systemPrompt,
			Messages:     append(conversationHistory, messages...),
			Tools:        ba.convertTools(),
			MaxTokens:    ba.maxTokens,
			Temperature:  ba.temperature,
		}

		ba.logger.Debug("Calling LLM",
			logging.String("agent", ba.name),
			logging.Int("tool_count", len(req.Tools)),
		)

		// Call LLM
		resp, err := ba.llmClient.GenerateCompletion(ctx, req)
		if err != nil {
			return "", fmt.Errorf("LLM call failed: %w", err)
		}

		ba.logger.Debug("LLM response received",
			logging.String("agent", ba.name),
			logging.Int("input_tokens", resp.Usage.InputTokens),
			logging.Int("output_tokens", resp.Usage.OutputTokens),
			logging.Int("tool_calls", len(resp.ToolCalls)),
		)

		// If no tool calls, return content
		if len(resp.ToolCalls) == 0 {
			return resp.Content, nil
		}

		// Add assistant response to conversation history
		conversationHistory = append(conversationHistory, llm.Message{
			Role:    "assistant",
			Content: resp.Content,
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
				})
				continue
			}

			ba.logger.Debug("Executing tool",
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
				})
			} else {
				conversationHistory = append(conversationHistory, llm.Message{
					Role:    "tool",
					Content: fmt.Sprintf("%v", result),
				})
			}
		}

		// Continue loop to get final response from LLM
		messages = []llm.Message{} // Clear, using conversationHistory now
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
