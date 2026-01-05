package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/user/gendocs/internal/cache"
	"github.com/user/gendocs/internal/config"
	"github.com/user/gendocs/internal/logging"
)

type DriftSeverity string

const (
	DriftSeverityNone     DriftSeverity = "none"
	DriftSeverityMinor    DriftSeverity = "minor"
	DriftSeverityModerate DriftSeverity = "moderate"
	DriftSeverityMajor    DriftSeverity = "major"
)

type AgentDriftStatus struct {
	Name          string    `json:"name"`
	DisplayName   string    `json:"display_name"`
	LastRun       time.Time `json:"last_run"`
	Success       bool      `json:"success"`
	OutputExists  bool      `json:"output_exists"`
	NeedsRerun    bool      `json:"needs_rerun"`
	AffectedFiles int       `json:"affected_files"`
	RerunReason   string    `json:"rerun_reason,omitempty"`
}

type DriftReport struct {
	HasDrift         bool               `json:"has_drift"`
	Severity         DriftSeverity      `json:"severity"`
	LastAnalysis     time.Time          `json:"last_analysis"`
	CurrentGitCommit string             `json:"current_git_commit"`
	CachedGitCommit  string             `json:"cached_git_commit"`
	GitCommitChanged bool               `json:"git_commit_changed"`
	NewFiles         []string           `json:"new_files"`
	ModifiedFiles    []string           `json:"modified_files"`
	DeletedFiles     []string           `json:"deleted_files"`
	AgentStatus      []AgentDriftStatus `json:"agent_status"`
	Summary          string             `json:"summary"`
	Recommendation   string             `json:"recommendation"`
	DocsDir          string             `json:"docs_dir"`
	CacheFile        string             `json:"cache_file"`
	IsFirstRun       bool               `json:"is_first_run"`
}

type CheckHandler struct {
	*BaseHandler
	config config.CheckConfig
}

func NewCheckHandler(cfg config.CheckConfig, logger *logging.Logger) *CheckHandler {
	return &CheckHandler{
		BaseHandler: &BaseHandler{
			Config: cfg.BaseConfig,
			Logger: logger,
		},
		config: cfg,
	}
}

func (h *CheckHandler) Handle(ctx context.Context) (*DriftReport, error) {
	h.Logger.Info("Starting drift detection",
		logging.String("repo_path", h.config.RepoPath),
	)

	report := &DriftReport{
		DocsDir:   filepath.Join(h.config.RepoPath, ".ai", "docs"),
		CacheFile: filepath.Join(h.config.RepoPath, cache.CacheFileName),
	}

	analysisCache, err := cache.LoadCache(h.config.RepoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load cache: %w", err)
	}

	if len(analysisCache.Files) == 0 || analysisCache.LastAnalysis.IsZero() {
		report.IsFirstRun = true
		report.HasDrift = true
		report.Severity = DriftSeverityMajor
		report.Summary = "No previous analysis found"
		report.Recommendation = "Run 'gendocs analyze' to generate initial documentation"
		return report, nil
	}

	report.LastAnalysis = analysisCache.LastAnalysis
	report.CachedGitCommit = analysisCache.GitCommit
	report.CurrentGitCommit = cache.GetCurrentGitCommit(h.config.RepoPath)

	if report.CurrentGitCommit != "" && report.CachedGitCommit != "" {
		report.GitCommitChanged = report.CurrentGitCommit != report.CachedGitCommit
	}

	var scanMetrics cache.ScanMetrics
	currentFiles, err := cache.ScanFiles(
		h.config.RepoPath,
		nil,
		analysisCache,
		&scanMetrics,
		h.config.GetMaxHashWorkers(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan files: %w", err)
	}

	changeReport := analysisCache.DetectChanges(h.config.RepoPath, currentFiles)
	report.NewFiles = changeReport.NewFiles
	report.ModifiedFiles = changeReport.ModifiedFiles
	report.DeletedFiles = changeReport.DeletedFiles
	report.HasDrift = changeReport.HasChanges

	report.AgentStatus = h.buildAgentStatus(analysisCache, changeReport)

	report.Severity = h.calculateSeverity(report)
	report.Summary = h.generateSummary(report)
	report.Recommendation = h.generateRecommendation(report)

	return report, nil
}

func (h *CheckHandler) buildAgentStatus(analysisCache *cache.AnalysisCache, changeReport *cache.ChangeReport) []AgentDriftStatus {
	agentDisplayNames := map[string]string{
		"structure_analyzer":    "Structure Analysis",
		"dependency_analyzer":   "Dependency Analysis",
		"data_flow_analyzer":    "Data Flow Analysis",
		"request_flow_analyzer": "Request Flow Analysis",
		"api_analyzer":          "API Analysis",
	}

	agentOutputFiles := map[string]string{
		"structure_analyzer":    "structure_analysis.md",
		"dependency_analyzer":   "dependency_analysis.md",
		"data_flow_analyzer":    "data_flow_analysis.md",
		"request_flow_analyzer": "request_flow_analysis.md",
		"api_analyzer":          "api_analysis.md",
	}

	var statuses []AgentDriftStatus
	changedFiles := append(changeReport.NewFiles, changeReport.ModifiedFiles...)
	changedFiles = append(changedFiles, changeReport.DeletedFiles...)

	for agentName, displayName := range agentDisplayNames {
		status := AgentDriftStatus{
			Name:        agentName,
			DisplayName: displayName,
		}

		if cachedStatus, exists := analysisCache.Agents[agentName]; exists {
			status.LastRun = cachedStatus.LastRun
			status.Success = cachedStatus.Success
		}

		if outputFile, exists := agentOutputFiles[agentName]; exists {
			outputPath := filepath.Join(h.config.RepoPath, ".ai", "docs", outputFile)
			if _, err := os.Stat(outputPath); err == nil {
				status.OutputExists = true
			}
		}

		for _, agent := range changeReport.AgentsToRun {
			if agent == agentName {
				status.NeedsRerun = true
				break
			}
		}

		if status.NeedsRerun {
			affectedCount := 0
			if patterns, exists := cache.AgentFilePatterns[agentName]; exists {
				for _, file := range changedFiles {
					for _, pattern := range patterns {
						if matchesPattern(file, pattern) {
							affectedCount++
							break
						}
					}
				}
			}
			status.AffectedFiles = affectedCount
			status.RerunReason = h.buildRerunReason(agentName, affectedCount, status)
		}

		statuses = append(statuses, status)
	}

	sort.Slice(statuses, func(i, j int) bool {
		if statuses[i].NeedsRerun != statuses[j].NeedsRerun {
			return statuses[i].NeedsRerun
		}
		return statuses[i].Name < statuses[j].Name
	})

	return statuses
}

func (h *CheckHandler) buildRerunReason(_ string, affectedCount int, status AgentDriftStatus) string {
	if !status.Success && !status.LastRun.IsZero() {
		return "Previous run failed"
	}
	if status.LastRun.IsZero() {
		return "Never run"
	}
	if !status.OutputExists {
		return "Output file missing"
	}
	if affectedCount > 0 {
		return fmt.Sprintf("%d affected file(s) changed", affectedCount)
	}
	return "Related files changed"
}

func (h *CheckHandler) calculateSeverity(report *DriftReport) DriftSeverity {
	if !report.HasDrift {
		return DriftSeverityNone
	}

	totalChanges := len(report.NewFiles) + len(report.ModifiedFiles) + len(report.DeletedFiles)
	agentsNeedingRerun := 0
	for _, agent := range report.AgentStatus {
		if agent.NeedsRerun {
			agentsNeedingRerun++
		}
	}

	if totalChanges > 20 || agentsNeedingRerun >= 4 {
		return DriftSeverityMajor
	}

	if totalChanges > 10 || agentsNeedingRerun >= 2 {
		return DriftSeverityModerate
	}

	return DriftSeverityMinor
}

func (h *CheckHandler) generateSummary(report *DriftReport) string {
	if !report.HasDrift {
		return fmt.Sprintf("Documentation is up to date (last analyzed: %s)",
			report.LastAnalysis.Format("2006-01-02 15:04"))
	}

	parts := []string{}
	totalChanges := len(report.NewFiles) + len(report.ModifiedFiles) + len(report.DeletedFiles)

	if totalChanges > 0 {
		changeParts := []string{}
		if len(report.NewFiles) > 0 {
			changeParts = append(changeParts, fmt.Sprintf("%d new", len(report.NewFiles)))
		}
		if len(report.ModifiedFiles) > 0 {
			changeParts = append(changeParts, fmt.Sprintf("%d modified", len(report.ModifiedFiles)))
		}
		if len(report.DeletedFiles) > 0 {
			changeParts = append(changeParts, fmt.Sprintf("%d deleted", len(report.DeletedFiles)))
		}
		parts = append(parts, fmt.Sprintf("%d file(s) changed (%s)", totalChanges, strings.Join(changeParts, ", ")))
	}

	agentsNeedingRerun := 0
	for _, agent := range report.AgentStatus {
		if agent.NeedsRerun {
			agentsNeedingRerun++
		}
	}
	if agentsNeedingRerun > 0 {
		parts = append(parts, fmt.Sprintf("%d agent(s) need re-run", agentsNeedingRerun))
	}

	if report.GitCommitChanged {
		parts = append(parts, fmt.Sprintf("Git commit changed: %s → %s", report.CachedGitCommit, report.CurrentGitCommit))
	}

	return strings.Join(parts, "; ")
}

func (h *CheckHandler) generateRecommendation(report *DriftReport) string {
	if !report.HasDrift {
		return "No action needed"
	}

	switch report.Severity {
	case DriftSeverityMajor:
		return "Run 'gendocs analyze' immediately - significant documentation drift detected"
	case DriftSeverityModerate:
		return "Run 'gendocs analyze' soon to update documentation"
	case DriftSeverityMinor:
		return "Consider running 'gendocs analyze' to keep documentation current"
	default:
		return "No action needed"
	}
}

func (h *CheckHandler) FormatTextReport(report *DriftReport) string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("📋 Documentation Drift Report\n")
	sb.WriteString("==============================\n\n")

	if report.IsFirstRun {
		sb.WriteString("⚠️  Status: No previous analysis found\n\n")
		sb.WriteString("   This appears to be a new project or the cache has been cleared.\n")
		sb.WriteString("   Run 'gendocs analyze' to generate initial documentation.\n\n")
		return sb.String()
	}

	statusIcon := "✅"
	if report.HasDrift {
		switch report.Severity {
		case DriftSeverityMajor:
			statusIcon = "🔴"
		case DriftSeverityModerate:
			statusIcon = "🟠"
		case DriftSeverityMinor:
			statusIcon = "🟡"
		}
	}

	sb.WriteString(fmt.Sprintf("%s Status: %s\n", statusIcon, toTitleCase(string(report.Severity))))
	sb.WriteString(fmt.Sprintf("   Last Analysis: %s\n", report.LastAnalysis.Format("2006-01-02 15:04:05")))
	if report.GitCommitChanged {
		sb.WriteString(fmt.Sprintf("   Git Commit: %s → %s (changed)\n", report.CachedGitCommit, report.CurrentGitCommit))
	} else if report.CurrentGitCommit != "" {
		sb.WriteString(fmt.Sprintf("   Git Commit: %s\n", report.CurrentGitCommit))
	}
	sb.WriteString("\n")

	if report.HasDrift {
		sb.WriteString("📁 File Changes:\n")
		if len(report.NewFiles) > 0 {
			sb.WriteString(fmt.Sprintf("   ➕ New: %d file(s)\n", len(report.NewFiles)))
			if h.config.Verbose {
				for _, f := range limitSlice(report.NewFiles, 10) {
					sb.WriteString(fmt.Sprintf("      • %s\n", f))
				}
				if len(report.NewFiles) > 10 {
					sb.WriteString(fmt.Sprintf("      ... and %d more\n", len(report.NewFiles)-10))
				}
			}
		}
		if len(report.ModifiedFiles) > 0 {
			sb.WriteString(fmt.Sprintf("   📝 Modified: %d file(s)\n", len(report.ModifiedFiles)))
			if h.config.Verbose {
				for _, f := range limitSlice(report.ModifiedFiles, 10) {
					sb.WriteString(fmt.Sprintf("      • %s\n", f))
				}
				if len(report.ModifiedFiles) > 10 {
					sb.WriteString(fmt.Sprintf("      ... and %d more\n", len(report.ModifiedFiles)-10))
				}
			}
		}
		if len(report.DeletedFiles) > 0 {
			sb.WriteString(fmt.Sprintf("   ➖ Deleted: %d file(s)\n", len(report.DeletedFiles)))
			if h.config.Verbose {
				for _, f := range limitSlice(report.DeletedFiles, 10) {
					sb.WriteString(fmt.Sprintf("      • %s\n", f))
				}
				if len(report.DeletedFiles) > 10 {
					sb.WriteString(fmt.Sprintf("      ... and %d more\n", len(report.DeletedFiles)-10))
				}
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("🤖 Agent Status:\n")
	for _, agent := range report.AgentStatus {
		icon := "✅"
		if agent.NeedsRerun {
			icon = "⚠️"
		} else if !agent.OutputExists {
			icon = "❌"
		}

		sb.WriteString(fmt.Sprintf("   %s %s", icon, agent.DisplayName))
		if agent.NeedsRerun {
			sb.WriteString(fmt.Sprintf(" - needs re-run (%s)", agent.RerunReason))
		} else if !agent.OutputExists {
			sb.WriteString(" - output missing")
		} else {
			sb.WriteString(" - up to date")
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	sb.WriteString("📊 Summary: ")
	sb.WriteString(report.Summary)
	sb.WriteString("\n\n")

	sb.WriteString("💡 Recommendation: ")
	sb.WriteString(report.Recommendation)
	sb.WriteString("\n\n")

	return sb.String()
}

func (h *CheckHandler) FormatJSONReport(report *DriftReport) (string, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal report: %w", err)
	}
	return string(data), nil
}

func matchesPattern(filename, pattern string) bool {
	if strings.Contains(pattern, "*") {
		pattern = strings.ToLower(pattern)
		filename = strings.ToLower(filepath.Base(filename))

		if strings.HasPrefix(pattern, "*.") && !strings.Contains(pattern[1:], "*") {
			ext := strings.TrimPrefix(pattern, "*")
			return strings.HasSuffix(filename, ext)
		}

		if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
			keyword := strings.Trim(pattern, "*")
			return strings.Contains(filename, keyword)
		}

		if strings.HasPrefix(pattern, "*") {
			suffix := strings.TrimPrefix(pattern, "*")
			return strings.HasSuffix(filename, suffix)
		}
	}

	return strings.EqualFold(filepath.Base(filename), pattern)
}

func limitSlice(slice []string, limit int) []string {
	if len(slice) <= limit {
		return slice
	}
	return slice[:limit]
}

func toTitleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
