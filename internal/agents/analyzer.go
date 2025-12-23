package agents

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/user/gendocs/internal/config"
	"github.com/user/gendocs/internal/llm"
	"github.com/user/gendocs/internal/logging"
	"github.com/user/gendocs/internal/prompts"
	"github.com/user/gendocs/internal/worker_pool"
)

// AnalyzerAgent orchestrates all sub-agents for code analysis
type AnalyzerAgent struct {
	config        config.AnalyzerConfig
	llmFactory    *llm.Factory
	promptManager *prompts.Manager
	logger        *logging.Logger
	workerPool    *worker_pool.WorkerPool
}

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

// NewAnalyzerAgent creates a new analyzer agent
func NewAnalyzerAgent(cfg config.AnalyzerConfig, promptManager *prompts.Manager, logger *logging.Logger) *AnalyzerAgent {
	// Create retry client
	retryClient := llm.NewRetryClient(llm.DefaultRetryConfig())

	// Create LLM factory
	factory := llm.NewFactory(retryClient)

	return &AnalyzerAgent{
		config:        cfg,
		llmFactory:    factory,
		promptManager: promptManager,
		logger:        logger,
		workerPool:    worker_pool.NewWorkerPool(cfg.MaxWorkers),
	}
}

// Run executes all sub-agents concurrently
func (aa *AnalyzerAgent) Run(ctx context.Context) (*AnalysisResult, error) {
	aa.logger.Info("Starting analysis",
		logging.String("repo_path", aa.config.RepoPath),
		logging.Int("max_workers", aa.config.MaxWorkers),
	)

	// Use the existing factory
	factory := aa.llmFactory

	// Build task list based on configuration
	var tasks []worker_pool.Task
	var outputPaths []string

	docsDir := filepath.Join(aa.config.RepoPath, ".ai", "docs")

	if !aa.config.ExcludeStructure {
		task, outputPath := aa.createTask(ctx, factory, "structure_analyzer", CreateStructureAnalyzer,
			filepath.Join(docsDir, "structure_analysis.md"))
		tasks = append(tasks, task)
		outputPaths = append(outputPaths, outputPath)
	}

	if !aa.config.ExcludeDeps {
		task, outputPath := aa.createTask(ctx, factory, "dependency_analyzer", CreateDependencyAnalyzer,
			filepath.Join(docsDir, "dependency_analysis.md"))
		tasks = append(tasks, task)
		outputPaths = append(outputPaths, outputPath)
	}

	if !aa.config.ExcludeDataFlow {
		task, outputPath := aa.createTask(ctx, factory, "data_flow_analyzer", CreateDataFlowAnalyzer,
			filepath.Join(docsDir, "data_flow_analysis.md"))
		tasks = append(tasks, task)
		outputPaths = append(outputPaths, outputPath)
	}

	if !aa.config.ExcludeReqFlow {
		task, outputPath := aa.createTask(ctx, factory, "request_flow_analyzer", CreateRequestFlowAnalyzer,
			filepath.Join(docsDir, "request_flow_analysis.md"))
		tasks = append(tasks, task)
		outputPaths = append(outputPaths, outputPath)
	}

	if !aa.config.ExcludeAPI {
		task, outputPath := aa.createTask(ctx, factory, "api_analyzer", CreateAPIAnalyzer,
			filepath.Join(docsDir, "api_analysis.md"))
		tasks = append(tasks, task)
		outputPaths = append(outputPaths, outputPath)
	}

	if len(tasks) == 0 {
		return nil, fmt.Errorf("no analysis tasks to run (all agents excluded)")
	}

	aa.logger.Info(fmt.Sprintf("Running %d analysis tasks concurrently", len(tasks)))

	// Execute all tasks concurrently
	results := aa.workerPool.Run(ctx, tasks)

	// Process results
	return aa.processResults(outputPaths, results), nil
}

// AgentCreator is a function that creates an agent
type AgentCreator func(llmCfg config.LLMConfig, repoPath string, factory *llm.Factory, promptMgr *prompts.Manager, logger *logging.Logger) (*SubAgent, error)

// createTask creates a task for the worker pool
func (aa *AnalyzerAgent) createTask(ctx context.Context, factory *llm.Factory, name string, creator AgentCreator, outputPath string) (worker_pool.Task, string) {
	task := func(ctx context.Context) (interface{}, error) {
		aa.logger.Debug(fmt.Sprintf("Creating %s", name))

		// Create agent
		agent, err := creator(aa.config.LLM, aa.config.RepoPath, factory, aa.promptManager, aa.logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create %s: %w", name, err)
		}

		// Run agent
		output, err := agent.Run(ctx)
		if err != nil {
			return nil, fmt.Errorf("%s failed: %w", name, err)
		}

		// Save output
		if err := agent.SaveOutput(output, outputPath); err != nil {
			return nil, fmt.Errorf("failed to save %s output: %w", name, err)
		}

		aa.logger.Info(fmt.Sprintf("%s completed successfully", name))
		return output, nil
	}

	return task, outputPath
}

// processResults processes worker pool results
func (aa *AnalyzerAgent) processResults(outputPaths []string, results []worker_pool.Result) *AnalysisResult {
	result := &AnalysisResult{
		Successful: []string{},
		Failed:     []FailedAnalysis{},
	}

	for i, r := range results {
		// Get agent name from output path
		name := filepath.Base(outputPaths[i])
		name = name[:len(name)-11] // Remove "_analysis.md"

		if r.Error != nil {
			result.Failed = append(result.Failed, FailedAnalysis{
				Name:  name,
				Error: r.Error,
			})
			aa.logger.Error(fmt.Sprintf("%s failed", name), logging.Error(r.Error))
		} else {
			result.Successful = append(result.Successful, name)
			aa.logger.Info(fmt.Sprintf("%s succeeded", name))
		}
	}

	aa.logger.Info(fmt.Sprintf("Analysis complete: %d/%d successful",
		len(result.Successful), len(result.Successful)+len(result.Failed)))

	return result
}

// DocumenterAgent generates README.md
type DocumenterAgent struct {
	config        config.DocumenterConfig
	promptManager *prompts.Manager
	logger        *logging.Logger
}

// NewDocumenterAgent creates a new documenter agent
func NewDocumenterAgent(cfg config.DocumenterConfig, promptManager *prompts.Manager, logger *logging.Logger) *DocumenterAgent {
	return &DocumenterAgent{
		config:        cfg,
		promptManager: promptManager,
		logger:        logger,
	}
}

// Run generates the README
func (da *DocumenterAgent) Run(ctx context.Context) error {
	retryClient := llm.NewRetryClient(llm.DefaultRetryConfig())
	factory := llm.NewFactory(retryClient)

	// Create documenter agent
	agent, err := CreateDocumenterAgent(da.config.LLM, da.config.RepoPath, factory, da.promptManager, da.logger)
	if err != nil {
		return fmt.Errorf("failed to create documenter agent: %w", err)
	}

	// Run agent
	output, err := agent.Run(ctx)
	if err != nil {
		return fmt.Errorf("documenter agent failed: %w", err)
	}

	// Save to README.md
	outputPath := filepath.Join(da.config.RepoPath, "README.md")
	if err := agent.SaveOutput(output, outputPath); err != nil {
		return err
	}

	da.logger.Info(fmt.Sprintf("README.md generated at %s", outputPath))
	return nil
}

// AIRulesGeneratorAgent generates AI assistant config files
type AIRulesGeneratorAgent struct {
	config        config.AIRulesConfig
	promptManager *prompts.Manager
	logger        *logging.Logger
}

// NewAIRulesGeneratorAgent creates a new AI rules generator agent
func NewAIRulesGeneratorAgent(cfg config.AIRulesConfig, promptManager *prompts.Manager, logger *logging.Logger) *AIRulesGeneratorAgent {
	return &AIRulesGeneratorAgent{
		config:        cfg,
		promptManager: promptManager,
		logger:        logger,
	}
}

// Run generates AI rules files
func (aa *AIRulesGeneratorAgent) Run(ctx context.Context) error {
	retryClient := llm.NewRetryClient(llm.DefaultRetryConfig())
	factory := llm.NewFactory(retryClient)

	// For now, generate CLAUDE.md
	agent, err := CreateAIRulesGeneratorAgent(aa.config.LLM, aa.config.RepoPath, factory, aa.promptManager, aa.logger)
	if err != nil {
		return fmt.Errorf("failed to create AI rules agent: %w", err)
	}

	// Run agent
	output, err := agent.Run(ctx)
	if err != nil {
		return fmt.Errorf("AI rules agent failed: %w", err)
	}

	// Save to CLAUDE.md
	outputPath := filepath.Join(aa.config.RepoPath, "CLAUDE.md")
	if err := agent.SaveOutput(output, outputPath); err != nil {
		return err
	}

	aa.logger.Info(fmt.Sprintf("CLAUDE.md generated at %s", outputPath))
	return nil
}
