package gitlab

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/user/gendocs/internal/config"
	"github.com/user/gendocs/internal/logging"
)

// Client represents a GitLab API client
type Client struct {
	httpClient   *http.Client
	apiURL       string
	OAuthToken   string
	UserName     string
	UserUsername string
	UserEmail    string
	logger       *logging.Logger
}

// Project represents a GitLab project
type Project struct {
	ID                int       `json:"id"`
	Name              string    `json:"name"`
	PathWithNamespace string    `json:"path_with_namespace"`
	HTTPURL           string    `json:"http_url_to_repo"`
	SSHURL            string    `json:"ssh_url_to_repo"`
	DefaultBranch     string    `json:"default_branch"`
	LastActivityAt    time.Time `json:"last_activity_at"`
	CreatedAt         time.Time `json:"created_at"`
	Archived          bool      `json:"archived"`
}

// MergeRequest represents a GitLab merge request
type MergeRequest struct {
	ID          int    `json:"id"`
	IID         int    `json:"iid"`
	Title       string `json:"title"`
	Description string `json:"description"`
	SourceBranch string `json:"source_branch"`
	TargetBranch string `json:"target_branch"`
	WebURL      string `json:"web_url"`
}

// NewClient creates a new GitLab client
func NewClient(cfg config.GitLabConfig, logger *logging.Logger) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiURL:       cfg.APIURL,
		OAuthToken:   cfg.OAuthToken,
		UserName:     cfg.UserName,
		UserUsername: cfg.UserUsername,
		UserEmail:    cfg.UserEmail,
		logger:       logger,
	}
}

// FetchProjectsInGroup fetches all projects in a group (including subgroups)
func (c *Client) FetchProjectsInGroup(ctx context.Context, groupID int) ([]Project, error) {
	c.logger.Info(fmt.Sprintf("Fetching projects in group %d", groupID))

	// For now, return empty list - would need full GitLab API implementation
	// This would require using xanzy/go-gitlab or implementing the API calls
	return []Project{}, nil
}

// ProjectFilter determines if a project should be analyzed
type ProjectFilter struct {
	MaxDaysSinceLastCommit int
	IgnoreProjects         map[string]bool
	IgnoreSubgroups        map[string]bool
}

// ShouldAnalyze determines if a project should be analyzed
func (c *Client) ShouldAnalyze(ctx context.Context, project Project, filter ProjectFilter) (bool, error) {
	// Skip archived projects
	if project.Archived {
		return false, nil
	}

	// Skip ignored projects
	if filter.IgnoreProjects[project.PathWithNamespace] {
		return false, nil
	}

	// Check if last commit is too old
	if filter.MaxDaysSinceLastCommit > 0 {
		daysSince := time.Since(project.LastActivityAt).Hours() / 24
		if int(daysSince) > filter.MaxDaysSinceLastCommit {
			return false, nil
		}
	}

	// Check if branch already exists
	branchName := fmt.Sprintf("ai-analyzer-%s", time.Now().Format("2006-01-02"))
	if branchExists, err := c.BranchExists(ctx, project, branchName); err == nil && branchExists {
		return false, nil
	}

	// Check if open MR exists
	if hasMR, err := c.HasOpenMR(ctx, project, branchName); err == nil && hasMR {
		return false, nil
	}

	return true, nil
}

// BranchExists checks if a branch exists in a project
func (c *Client) BranchExists(ctx context.Context, project Project, branchName string) (bool, error) {
	// Placeholder - would implement GitLab API call
	return false, nil
}

// HasOpenMR checks if an open MR exists for a branch
func (c *Client) HasOpenMR(ctx context.Context, project Project, branchName string) (bool, error) {
	// Placeholder - would implement GitLab API call
	return false, nil
}

// CreateBranch creates a new branch in a project
func (c *Client) CreateBranch(ctx context.Context, project Project, branchName, fromBranch string) error {
	// Placeholder - would implement GitLab API call
	c.logger.Info(fmt.Sprintf("Would create branch '%s' in %s", branchName, project.PathWithNamespace))
	return nil
}

// CreateCommit creates a commit with the given files
func (c *Client) CreateCommit(ctx context.Context, project Project, branchName, message string, files map[string]string) error {
	// Placeholder - would implement GitLab API call
	c.logger.Info(fmt.Sprintf("Would create commit in %s on branch %s", project.PathWithNamespace, branchName))
	return nil
}

// CreateMR creates a merge request
func (c *Client) CreateMR(ctx context.Context, project Project, sourceBranch, targetBranch, title, description string) (*MergeRequest, error) {
	// Placeholder - would implement GitLab API call
	c.logger.Info(fmt.Sprintf("Would create MR in %s: %s -> %s", project.PathWithNamespace, sourceBranch, targetBranch))
	return &MergeRequest{
		Title:       title,
		Description: description,
		SourceBranch: sourceBranch,
		TargetBranch: targetBranch,
	}, nil
}
