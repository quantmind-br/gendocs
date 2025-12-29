package llmcache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/user/gendocs/internal/llmtypes"
)

// CacheKeyRequest represents the fields used for cache key generation.
// It contains the essential elements of an LLM request that affect the response.
type CacheKeyRequest struct {
	SystemPrompt string            `json:"system_prompt"` // System prompt that guides the LLM behavior
	Messages     []CacheKeyMessage `json:"messages"`      // Conversation messages (order matters)
	Tools        []CacheKeyTool    `json:"tools"`         // Available tools (sorted for order independence)
	Temperature  float64           `json:"temperature"`   // Sampling temperature (affects response randomness)
}

// CacheKeyMessage represents a message in cache key generation.
// It captures the essential components of a chat message.
type CacheKeyMessage struct {
	Role    string `json:"role"`              // Message role: "system", "user", "assistant", or "tool"
	Content string `json:"content"`           // Message content text
	ToolID  string `json:"tool_id,omitempty"` // ID of the tool being responded to (for tool messages)
}

// CacheKeyTool represents a tool in cache key generation.
// It defines a tool/function that the LLM can call.
type CacheKeyTool struct {
	Name        string                 `json:"name"`        // Tool identifier name
	Description string                 `json:"description"` // Tool description for the LLM
	Parameters  map[string]interface{} `json:"parameters"`  // Tool parameter schema
}

// GenerateCacheKey generates a unique cache key from a CompletionRequest.
//
// The cache key is a SHA256 hash derived from the canonical JSON representation
// of the request. This ensures that identical requests generate the same key,
// enabling efficient cache lookups.
//
// Key generation details:
// - System prompt is trimmed and included
// - Messages are preserved in order (order affects LLM responses)
// - Tools are sorted by name for order-independent hashing
// - Temperature is included (affects response randomness)
//
// Returns an error if JSON marshaling fails.
func GenerateCacheKey(req llmtypes.CompletionRequest) (string, error) {
	// Create cache key request
	keyReq := CacheKeyRequest{
		SystemPrompt: strings.TrimSpace(req.SystemPrompt),
		Temperature:  req.Temperature,
	}

	// Convert messages (preserving order - message order is significant)
	for _, msg := range req.Messages {
		keyReq.Messages = append(keyReq.Messages, CacheKeyMessage{
			Role:    msg.Role,
			Content: strings.TrimSpace(msg.Content),
			ToolID:  msg.ToolID,
		})
	}

	// Convert tools and sort by name (tool order doesn't affect LLM behavior)
	tools := make([]CacheKeyTool, len(req.Tools))
	for i, tool := range req.Tools {
		tools[i] = CacheKeyTool{
			Name:        strings.TrimSpace(tool.Name),
			Description: strings.TrimSpace(tool.Description),
			Parameters:  tool.Parameters,
		}
	}
	// Sort tools by name for order-independent hashing
	sort.Slice(tools, func(i, j int) bool {
		return tools[i].Name < tools[j].Name
	})
	keyReq.Tools = tools

	// Marshal to canonical JSON
	data, err := json.Marshal(keyReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal cache key request: %w", err)
	}

	// Compute SHA256 hash
	hash := sha256.Sum256(data)

	// Return hex-encoded hash
	return hex.EncodeToString(hash[:]), nil
}

// CacheKeyRequestFrom converts a CompletionRequest to a CacheKeyRequest.
//
// This function extracts and normalizes the fields needed for cache key generation.
// It's useful when you need to store the request alongside the cached response
// for validation or debugging purposes.
func CacheKeyRequestFrom(req llmtypes.CompletionRequest) CacheKeyRequest {
	keyReq := CacheKeyRequest{
		SystemPrompt: strings.TrimSpace(req.SystemPrompt),
		Temperature:  req.Temperature,
	}

	// Convert messages (preserving order)
	for _, msg := range req.Messages {
		keyReq.Messages = append(keyReq.Messages, CacheKeyMessage{
			Role:    msg.Role,
			Content: strings.TrimSpace(msg.Content),
			ToolID:  msg.ToolID,
		})
	}

	// Convert tools and sort by name
	tools := make([]CacheKeyTool, len(req.Tools))
	for i, tool := range req.Tools {
		tools[i] = CacheKeyTool{
			Name:        strings.TrimSpace(tool.Name),
			Description: strings.TrimSpace(tool.Description),
			Parameters:  tool.Parameters,
		}
	}
	sort.Slice(tools, func(i, j int) bool {
		return tools[i].Name < tools[j].Name
	})
	keyReq.Tools = tools

	return keyReq
}
