package llmcache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/user/gendocs/internal/llm"
)

// GenerateCacheKey generates a unique cache key from a CompletionRequest
func GenerateCacheKey(req llm.CompletionRequest) (string, error) {
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
			Name:        tool.Name,
			Description: tool.Description,
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
