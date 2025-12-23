package agents

import (
	"github.com/user/gendocs/internal/config"
	"github.com/user/gendocs/internal/llm"
	"github.com/user/gendocs/internal/logging"
	"github.com/user/gendocs/internal/prompts"
)

// CreateStructureAnalyzer creates the structure analyzer sub-agent
func CreateStructureAnalyzer(llmCfg config.LLMConfig, repoPath string, llmFactory *llm.Factory, promptManager *prompts.Manager, logger *logging.Logger) (*SubAgent, error) {
	cfg := SubAgentConfig{
		Name:         "StructureAnalyzer",
		LLMConfig:    llmCfg,
		RepoPath:     repoPath,
		PromptSuffix: "structure_analyzer",
	}
	return NewSubAgent(cfg, llmFactory, promptManager, logger)
}

// CreateDependencyAnalyzer creates the dependency analyzer sub-agent
func CreateDependencyAnalyzer(llmCfg config.LLMConfig, repoPath string, llmFactory *llm.Factory, promptManager *prompts.Manager, logger *logging.Logger) (*SubAgent, error) {
	cfg := SubAgentConfig{
		Name:         "DependencyAnalyzer",
		LLMConfig:    llmCfg,
		RepoPath:     repoPath,
		PromptSuffix: "dependency_analyzer",
	}
	return NewSubAgent(cfg, llmFactory, promptManager, logger)
}

// CreateDataFlowAnalyzer creates the data flow analyzer sub-agent
func CreateDataFlowAnalyzer(llmCfg config.LLMConfig, repoPath string, llmFactory *llm.Factory, promptManager *prompts.Manager, logger *logging.Logger) (*SubAgent, error) {
	cfg := SubAgentConfig{
		Name:         "DataFlowAnalyzer",
		LLMConfig:    llmCfg,
		RepoPath:     repoPath,
		PromptSuffix: "data_flow_analyzer",
	}
	return NewSubAgent(cfg, llmFactory, promptManager, logger)
}

// CreateRequestFlowAnalyzer creates the request flow analyzer sub-agent
func CreateRequestFlowAnalyzer(llmCfg config.LLMConfig, repoPath string, llmFactory *llm.Factory, promptManager *prompts.Manager, logger *logging.Logger) (*SubAgent, error) {
	cfg := SubAgentConfig{
		Name:         "RequestFlowAnalyzer",
		LLMConfig:    llmCfg,
		RepoPath:     repoPath,
		PromptSuffix: "request_flow_analyzer",
	}
	return NewSubAgent(cfg, llmFactory, promptManager, logger)
}

// CreateAPIAnalyzer creates the API analyzer sub-agent
func CreateAPIAnalyzer(llmCfg config.LLMConfig, repoPath string, llmFactory *llm.Factory, promptManager *prompts.Manager, logger *logging.Logger) (*SubAgent, error) {
	cfg := SubAgentConfig{
		Name:         "APIAnalyzer",
		LLMConfig:    llmCfg,
		RepoPath:     repoPath,
		PromptSuffix: "api_analyzer",
	}
	return NewSubAgent(cfg, llmFactory, promptManager, logger)
}

// CreateDocumenterAgent creates the documenter agent (README generator)
func CreateDocumenterAgent(llmCfg config.LLMConfig, repoPath string, llmFactory *llm.Factory, promptManager *prompts.Manager, logger *logging.Logger) (*SubAgent, error) {
	cfg := SubAgentConfig{
		Name:         "DocumenterAgent",
		LLMConfig:    llmCfg,
		RepoPath:     repoPath,
		PromptSuffix: "documenter",
	}
	return NewSubAgent(cfg, llmFactory, promptManager, logger)
}

// CreateAIRulesGeneratorAgent creates the AI rules generator agent
func CreateAIRulesGeneratorAgent(llmCfg config.LLMConfig, repoPath string, llmFactory *llm.Factory, promptManager *prompts.Manager, logger *logging.Logger) (*SubAgent, error) {
	cfg := SubAgentConfig{
		Name:         "AIRulesGeneratorAgent",
		LLMConfig:    llmCfg,
		RepoPath:     repoPath,
		PromptSuffix: "ai_rules",
	}
	return NewSubAgent(cfg, llmFactory, promptManager, logger)
}
