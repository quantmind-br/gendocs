package agents

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/user/gendocs/internal/config"
	"github.com/user/gendocs/internal/llm"
	"github.com/user/gendocs/internal/llmcache"
	"github.com/user/gendocs/internal/logging"
	"github.com/user/gendocs/internal/prompts"
	"github.com/user/gendocs/internal/tools"
)

// SubAgentConfig holds configuration for sub-agents
type SubAgentConfig struct {
	Name         string
	LLMConfig    config.LLMConfig
	RepoPath     string
	PromptSuffix string // e.g., "structure_analyzer"
}

// SubAgent is a specialized analysis agent
type SubAgent struct {
	*BaseAgent
	config SubAgentConfig
}

// NewSubAgent creates a new sub-agent
func NewSubAgent(cfg SubAgentConfig, llmFactory *llm.Factory, promptManager *prompts.Manager, logger *logging.Logger) (*SubAgent, error) {
	// Create LLM client
	llmClient, err := llmFactory.CreateClient(cfg.LLMConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}

	// Create tools
	toolList := []tools.Tool{
		tools.NewFileReadTool(2),
		tools.NewListFilesTool(2),
	}

	// Load system prompt
	systemPrompt, err := promptManager.Get(cfg.PromptSuffix + "_system")
	if err != nil {
		return nil, fmt.Errorf("failed to load system prompt: %w", err)
	}

	baseAgent := NewBaseAgent(
		cfg.Name,
		llmClient,
		toolList,
		promptManager,
		logger,
		systemPrompt,
		cfg.LLMConfig.GetRetries(),
	)

	return &SubAgent{
		BaseAgent: baseAgent,
		config:    cfg,
	}, nil
}

// setupCaches initializes LLM response caches based on configuration
// Returns memory cache, disk cache, cleanup function, and error
// The cleanup function should be called when done to stop auto-save
func setupCaches(llmCfg config.LLMConfig, logger *logging.Logger) (*llmcache.LRUCache, *llmcache.DiskCache, func(), error) {
	// Check if caching is enabled
	if !llmCfg.Cache.IsEnabled() {
		return nil, nil, func() {}, nil
	}

	// Create memory cache
	memoryCache := llmcache.NewLRUCache(llmCfg.Cache.GetMaxSize())

	// Create disk cache
	diskCache := llmcache.NewDiskCache(
		llmCfg.Cache.GetCachePath(),
		llmCfg.Cache.GetTTL(),
		100*1024*1024, // 100MB max disk size
	)

	// Load existing disk cache
	if err := diskCache.Load(); err != nil {
		logger.Warn(fmt.Sprintf("Failed to load disk cache: %v (starting with empty cache)", err))
	}

	// Start auto-save (every 5 minutes)
	diskCache.StartAutoSave(5 * time.Minute)

	// Create cleanup function
	cleanup := func() {
		diskCache.Stop()
	}

	logger.Info(fmt.Sprintf("LLM response caching enabled (max_size=%d, ttl=%s, path=%s)",
		llmCfg.Cache.GetMaxSize(),
		llmCfg.Cache.GetTTL(),
		llmCfg.Cache.GetCachePath()))

	return memoryCache, diskCache, cleanup, nil
}

// Run executes the sub-agent
func (sa *SubAgent) Run(ctx context.Context) (string, error) {
	// Render user prompt with variables
	userPrompt, err := sa.promptManager.Render(sa.config.PromptSuffix+"_user", map[string]interface{}{
		"RepoPath": sa.config.RepoPath,
	})
	if err != nil {
		return "", fmt.Errorf("failed to render user prompt: %w", err)
	}

	// Run with retry logic
	var lastErr error
	for attempt := 0; attempt < sa.maxRetries; attempt++ {
		sa.logger.Info(fmt.Sprintf("Running sub-agent %s (attempt %d/%d)", sa.config.Name, attempt+1, sa.maxRetries))

		result, err := sa.RunOnce(ctx, userPrompt)
		if err == nil {
			sa.logger.Info(fmt.Sprintf("Sub-agent %s completed successfully", sa.config.Name))
			return result, nil
		}

		lastErr = err
		sa.logger.Warn(fmt.Sprintf("Sub-agent %s attempt %d failed: %v", sa.config.Name, attempt+1, err))
	}

	return "", fmt.Errorf("sub-agent %s failed after %d retries: %w", sa.config.Name, sa.maxRetries, lastErr)
}

// SaveOutput saves the agent output to a file
func (sa *SubAgent) SaveOutput(output, outputPath string) error {
	// Clean the output to remove unwanted preambles and code fences
	cleanedOutput := cleanLLMOutput(output)

	// Ensure directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write output
	if err := os.WriteFile(outputPath, []byte(cleanedOutput), 0644); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	sa.logger.Info(fmt.Sprintf("Output saved to %s", outputPath))
	return nil
}

// cleanLLMOutput removes common LLM output artifacts like markdown code fences and preambles
func cleanLLMOutput(output string) string {
	lines := strings.Split(output, "\n")

	// Find the start of actual markdown content
	startIdx := -1

	// First, try to find markdown code fence
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "```markdown" {
			// Start after the code fence
			startIdx = i + 1
			break
		}
	}

	// If no code fence found, look for markdown heading (# Something)
	if startIdx == -1 {
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			// Look for any markdown heading (# or ## or ###, etc.)
			if strings.HasPrefix(trimmed, "#") && len(trimmed) > 1 && trimmed[1] != '`' {
				startIdx = i
				break
			}
		}
	}

	// If still no markdown found, look for common preamble patterns to skip
	if startIdx == -1 {
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			lower := strings.ToLower(trimmed)
			// Skip lines that look like preambles or tool outputs
			if strings.HasPrefix(lower, "okay,") ||
			   strings.HasPrefix(lower, "here's") ||
			   strings.HasPrefix(lower, "here is") ||
			   strings.Contains(trimmed, "```tool_outputs") ||
			   strings.Contains(trimmed, "{\"read_file_response\"") ||
			   (strings.HasPrefix(trimmed, "*") && strings.Contains(trimmed, "**")) {
				continue
			}
			// Found a line that doesn't match preamble patterns
			// If this line starts a list or paragraph, it might be the start of content
			if trimmed != "" && !strings.HasPrefix(trimmed, "```") {
				startIdx = i
				break
			}
		}
	}

	// If no markdown heading found, return original output
	if startIdx == -1 {
		return output
	}

	// Take everything from the first heading onward
	relevantLines := lines[startIdx:]

	// Remove trailing code fence if present
	endIdx := len(relevantLines)
	for i := len(relevantLines) - 1; i >= 0; i-- {
		trimmed := strings.TrimSpace(relevantLines[i])
		if trimmed == "" {
			continue
		}
		if trimmed == "```" {
			endIdx = i
		} else {
			break
		}
	}

	return strings.Join(relevantLines[:endIdx], "\n")
}
