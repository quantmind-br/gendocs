package handlers

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/user/gendocs/internal/config"
	"github.com/user/gendocs/internal/errors"
	"github.com/user/gendocs/internal/gitlab"
	"github.com/user/gendocs/internal/logging"
)

// CronjobHandler handles the cronjob analyze command
type CronjobHandler struct {
	*BaseHandler
	config    config.CronjobConfig
	gitLabCfg config.GitLabConfig
	analyzerCfg config.AnalyzerConfig
	gitlab    *gitlab.Client
}

// NewCronjobHandler creates a new cronjob handler
func NewCronjobHandler(
	cronjobCfg config.CronjobConfig,
	gitLabCfg config.GitLabConfig,
	analyzerCfg config.AnalyzerConfig,
	logger *logging.Logger,
) *CronjobHandler {
	return &CronjobHandler{
		BaseHandler: &BaseHandler{
			Config: config.BaseConfig{
				RepoPath: cronjobCfg.WorkingPath,
				Debug:    false,
			},
			Logger: logger,
		},
		config:      cronjobCfg,
		gitLabCfg:   gitLabCfg,
		analyzerCfg: analyzerCfg,
		gitlab:      gitlab.NewClient(gitLabCfg, logger),
	}
}

// ProcessedResult holds the results of processing projects
type ProcessedResult struct {
	ProcessedCount int
	SuccessCount   int
	ErrorCount     int
	SkippedCount   int
	FailedProjects []FailedProject
}

// FailedProject represents a project that failed to process
type FailedProject struct {
	Name string
	Error error
}

// Handle executes the cronjob analysis
func (h *CronjobHandler) Handle(ctx context.Context) error {
	h.Logger.Info("Starting cronjob analysis",
		logging.String("working_path", h.config.WorkingPath),
		logging.Int("group_project_id", h.config.GroupProjectID),
		logging.Int("max_days", h.config.MaxDaysSinceLastCommit),
	)

	// Fetch all projects in the group
	projects, err := h.gitlab.FetchProjectsInGroup(ctx, h.config.GroupProjectID)
	if err != nil {
		return errors.NewCronjobError("failed to fetch projects", err)
	}

	h.Logger.Info(fmt.Sprintf("Found %d projects in group", len(projects)))

	// Filter projects
	filter := gitlab.ProjectFilter{
		MaxDaysSinceLastCommit: h.config.MaxDaysSinceLastCommit,
	}

	var applicableProjects []gitlab.Project
	for _, project := range projects {
		shouldAnalyze, err := h.gitlab.ShouldAnalyze(ctx, project, filter)
		if err != nil {
			h.Logger.Warn(fmt.Sprintf("Error checking project %s: %v", project.PathWithNamespace, err))
			continue
		}
		if shouldAnalyze {
			applicableProjects = append(applicableProjects, project)
		}
	}

	h.Logger.Info(fmt.Sprintf("%d projects applicable for analysis", len(applicableProjects)))

	// Process each applicable project
	result := &ProcessedResult{
		FailedProjects: []FailedProject{},
	}

	for _, project := range applicableProjects {
		h.Logger.Info(fmt.Sprintf("Processing %s", project.PathWithNamespace))

		if err := h.processProject(ctx, project); err != nil {
			result.ErrorCount++
			result.FailedProjects = append(result.FailedProjects, FailedProject{
				Name: project.PathWithNamespace,
				Error: err,
			})
			h.Logger.Error(fmt.Sprintf("Failed to process %s: %v", project.PathWithNamespace, err))
		} else {
			result.SuccessCount++
		}
		result.ProcessedCount++
	}

	// Log summary
	h.Logger.Info(fmt.Sprintf("Cronjob complete: %d processed, %d succeeded, %d failed, %d skipped",
		result.ProcessedCount, result.SuccessCount, result.ErrorCount,
		len(projects)-len(applicableProjects)))

	if result.ErrorCount > 0 && result.SuccessCount == 0 {
		return errors.NewCronjobError("all projects failed", fmt.Errorf("%d failures", result.ErrorCount))
	}

	return nil
}

// processProject processes a single project
func (h *CronjobHandler) processProject(ctx context.Context, project gitlab.Project) error {
	// Create temp directory for cloning
	tempDir := filepath.Join(h.config.WorkingPath, "tmp", fmt.Sprintf("project_%d", project.ID))
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Clone repository
	if err := h.cloneRepository(ctx, project, tempDir); err != nil {
		return errors.NewGitCloneError(project.HTTPURL, "clone failed", err)
	}

	// Create branch
	branchName := fmt.Sprintf("ai-analyzer-%s", time.Now().Format("2006-01-02"))
	if err := h.gitlab.CreateBranch(ctx, project, branchName, project.DefaultBranch); err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	// Run analysis
	analyzerCfg := h.analyzerCfg
	analyzerCfg.RepoPath = tempDir

	// Run analyze command (via subprocess for now, could be refactored to use handler directly)
	if err := h.runAnalysis(ctx, tempDir); err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	// Commit results
	commitMsg := fmt.Sprintf("[skip ci] AI analysis: %s", time.Now().Format("2006-01-02"))
	if err := h.gitlab.CreateCommit(ctx, project, branchName, commitMsg, nil); err != nil {
		return fmt.Errorf("failed to create commit: %w", err)
	}

	// Create merge request
	mrTitle := fmt.Sprintf("AI Analysis: %s", time.Now().Format("2006-01-02"))
	mrDescription := fmt.Sprintf("Automated AI analysis generated on %s\n\nThis MR contains:\n- Structure analysis\n- Dependency analysis\n- Data flow analysis\n- Request flow analysis\n- API analysis", time.Now().Format("2006-01-02"))
	mr, err := h.gitlab.CreateMR(ctx, project, branchName, project.DefaultBranch, mrTitle, mrDescription)
	if err != nil {
		return fmt.Errorf("failed to create MR: %w", err)
	}

	h.Logger.Info(fmt.Sprintf("Created MR %d for %s", mr.IID, project.PathWithNamespace))
	return nil
}

// cloneRepository clones a GitLab repository
func (h *CronjobHandler) cloneRepository(ctx context.Context, project gitlab.Project, destDir string) error {
	// Clone with authentication
	url := project.HTTPURL
	if h.gitlab.OAuthToken != "" {
		// Inject token into URL
		url = fmt.Sprintf("https://oauth2:%s@%s", h.gitlab.OAuthToken, project.HTTPURL[8:]) // Strip https://
	}

	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", url, destDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		h.Logger.Info(fmt.Sprintf("Git clone output: %s", string(output)))
		return err
	}

	return nil
}

// runAnalysis runs the analysis on a repository
func (h *CronjobHandler) runAnalysis(ctx context.Context, repoPath string) error {
	// Run gendocs analyze command as subprocess
	cmd := exec.CommandContext(ctx, "./gendocs", "analyze", "--repo-path", repoPath)
	cmd.Dir = filepath.Dir(repoPath) // Run from parent directory to find binary

	output, err := cmd.CombinedOutput()
	if err != nil {
		h.Logger.Info(fmt.Sprintf("Analysis output: %s", string(output)))
		return err
	}

	h.Logger.Info(fmt.Sprintf("Analysis output: %s", string(output)))
	return nil
}
