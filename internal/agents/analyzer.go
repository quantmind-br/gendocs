package agents

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/user/gendocs/internal/cache"
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

	// Load cache and detect changes (unless force mode)
	var analysisCache *cache.AnalysisCache
	var changeReport *cache.ChangeReport
	var currentFiles map[string]cache.FileInfo
	var scanErr error
	var scanMetrics cache.ScanMetrics

	// Always load/create cache (needed for selective hashing)
	analysisCache, _ = cache.LoadCache(aa.config.RepoPath)
	if analysisCache == nil {
		analysisCache = cache.NewCache()
	}

	// Always scan files for cache update (with cache for selective hashing and metrics tracking)
	currentFiles, scanErr = cache.ScanFiles(aa.config.RepoPath, nil, analysisCache, &scanMetrics, aa.config.GetMaxHashWorkers())
	if scanErr != nil {
		aa.logger.Warn(fmt.Sprintf("Failed to scan files: %v", scanErr))
	}

	// Log scan metrics to show optimization effectiveness
	aa.logger.Debug("File scan metrics",
		logging.Int("total_files", scanMetrics.TotalFiles),
		logging.Int("cached_files", scanMetrics.CachedFiles),
		logging.Int("hashed_files", scanMetrics.HashedFiles),
	)
	if scanMetrics.TotalFiles > 0 {
		cacheHitRate := float64(scanMetrics.CachedFiles) / float64(scanMetrics.TotalFiles) * 100
		aa.logger.Debug(fmt.Sprintf("Cache hit rate: %.1f%% (%d/%d files reused cached hashes)",
			cacheHitRate, scanMetrics.CachedFiles, scanMetrics.TotalFiles))
	}

	if !aa.config.Force && scanErr == nil {
		// Detect changes
		changeReport = analysisCache.DetectChanges(aa.config.RepoPath, currentFiles)

		if !changeReport.HasChanges {
			aa.logger.Info("No changes detected since last analysis",
				logging.String("last_analysis", analysisCache.LastAnalysis.Format("2006-01-02 15:04:05")),
			)
			return &AnalysisResult{
				Successful: []string{"No changes - using cached results"},
				Failed:     []FailedAnalysis{},
			}, nil
		}

		aa.logger.Info("Incremental analysis",
			logging.Int("new_files", len(changeReport.NewFiles)),
			logging.Int("modified_files", len(changeReport.ModifiedFiles)),
			logging.Int("deleted_files", len(changeReport.DeletedFiles)),
			logging.Int("agents_to_run", len(changeReport.AgentsToRun)),
			logging.Int("agents_to_skip", len(changeReport.AgentsToSkip)),
		)

		if len(changeReport.AgentsToSkip) > 0 {
			aa.logger.Info(fmt.Sprintf("Skipping unchanged agents: %v", changeReport.AgentsToSkip))
		}
	} else {
		aa.logger.Info("Force mode enabled - running full analysis")
	}

	// Use the existing factory
	factory := aa.llmFactory

	// Build task list based on configuration and change report
	var tasks []worker_pool.Task
	var outputPaths []string
	var agentNames []string

	docsDir := filepath.Join(aa.config.RepoPath, ".ai", "docs")

	// Helper to check if agent should run
	shouldRunAgent := func(agentName string) bool {
		if aa.config.Force || changeReport == nil {
			return true
		}
		for _, a := range changeReport.AgentsToRun {
			if a == agentName {
				return true
			}
		}
		return false
	}

	if !aa.config.ExcludeStructure && shouldRunAgent("structure_analyzer") {
		task, outputPath := aa.createTask(ctx, factory, "structure_analyzer", CreateStructureAnalyzer,
			filepath.Join(docsDir, "structure_analysis.md"))
		tasks = append(tasks, task)
		outputPaths = append(outputPaths, outputPath)
		agentNames = append(agentNames, "structure_analyzer")
	}

	if !aa.config.ExcludeDeps && shouldRunAgent("dependency_analyzer") {
		task, outputPath := aa.createTask(ctx, factory, "dependency_analyzer", CreateDependencyAnalyzer,
			filepath.Join(docsDir, "dependency_analysis.md"))
		tasks = append(tasks, task)
		outputPaths = append(outputPaths, outputPath)
		agentNames = append(agentNames, "dependency_analyzer")
	}

	if !aa.config.ExcludeDataFlow && shouldRunAgent("data_flow_analyzer") {
		task, outputPath := aa.createTask(ctx, factory, "data_flow_analyzer", CreateDataFlowAnalyzer,
			filepath.Join(docsDir, "data_flow_analysis.md"))
		tasks = append(tasks, task)
		outputPaths = append(outputPaths, outputPath)
		agentNames = append(agentNames, "data_flow_analyzer")
	}

	if !aa.config.ExcludeReqFlow && shouldRunAgent("request_flow_analyzer") {
		task, outputPath := aa.createTask(ctx, factory, "request_flow_analyzer", CreateRequestFlowAnalyzer,
			filepath.Join(docsDir, "request_flow_analysis.md"))
		tasks = append(tasks, task)
		outputPaths = append(outputPaths, outputPath)
		agentNames = append(agentNames, "request_flow_analyzer")
	}

	if !aa.config.ExcludeAPI && shouldRunAgent("api_analyzer") {
		task, outputPath := aa.createTask(ctx, factory, "api_analyzer", CreateAPIAnalyzer,
			filepath.Join(docsDir, "api_analysis.md"))
		tasks = append(tasks, task)
		outputPaths = append(outputPaths, outputPath)
		agentNames = append(agentNames, "api_analyzer")
	}

	if len(tasks) == 0 {
		if changeReport != nil && len(changeReport.AgentsToSkip) > 0 {
			aa.logger.Info("All required agents already up-to-date")
			return &AnalysisResult{
				Successful: changeReport.AgentsToSkip,
				Failed:     []FailedAnalysis{},
			}, nil
		}
		return nil, fmt.Errorf("no analysis tasks to run (all agents excluded)")
	}

	aa.logger.Info(fmt.Sprintf("Running %d analysis tasks concurrently", len(tasks)))

	// Execute all tasks concurrently
	results := aa.workerPool.Run(ctx, tasks)

	// Process results
	analysisResult := aa.processResults(outputPaths, results)

	// Update cache with results
	if analysisCache != nil && len(currentFiles) > 0 {
		agentResults := make(map[string]bool)
		for i, name := range agentNames {
			agentResults[name] = results[i].Error == nil
		}
		// Also mark skipped agents as successful (they were already cached)
		if changeReport != nil {
			for _, skipped := range changeReport.AgentsToSkip {
				agentResults[skipped] = true
			}
		}
		// In force mode, mark all agents as successful
		if aa.config.Force {
			for _, name := range []string{"structure_analyzer", "dependency_analyzer", "data_flow_analyzer", "request_flow_analyzer", "api_analyzer"} {
				if _, exists := agentResults[name]; !exists {
					agentResults[name] = true
				}
			}
		}

		analysisCache.UpdateAfterAnalysis(aa.config.RepoPath, currentFiles, agentResults)
		if err := analysisCache.Save(aa.config.RepoPath); err != nil {
			aa.logger.Warn(fmt.Sprintf("Failed to save cache: %v", err))
		} else {
			aa.logger.Info("Analysis cache updated")
		}
	}

	return analysisResult, nil
}

// createTask creates a task for the worker pool
func (aa *AnalyzerAgent) createTask(ctx context.Context, factory *llm.Factory, name string, creator AgentCreator, outputPath string) (worker_pool.Task, string) {
	task := func(ctx context.Context) (interface{}, error) {
		aa.logger.Info(fmt.Sprintf("Creating %s", name))

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
