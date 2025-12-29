package agents

import (
	"github.com/user/gendocs/internal/config"
	"github.com/user/gendocs/internal/llm"
	"github.com/user/gendocs/internal/logging"
	"github.com/user/gendocs/internal/prompts"
)

// AnalysisResult represents the result of an analysis
type AnalysisResult struct {
	Successful []string
	Failed     []FailedAnalysis
}

// FailedAnalysis represents a failed analysis
type FailedAnalysis struct {
	Name  string
	Error error
}

// AgentCreator is a function that creates an agent
type AgentCreator func(llmCfg config.LLMConfig, repoPath string, factory *llm.Factory, promptMgr *prompts.Manager, logger *logging.Logger) (*SubAgent, error)
