package github

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/go-github/v74/github"
	"golang.org/x/oauth2"

	"ado-gh-wi-migrator/config"
	"ado-gh-wi-migrator/models"
)

// Client represents a GitHub client
type Client struct {
	client *github.Client
	config *config.GitHubConfig
	logger *slog.Logger
}

// NewClient creates a new GitHub client
func NewClient(cfg *config.GitHubConfig, logger *slog.Logger) (*Client, error) {
	if cfg.Token == "" {
		return nil, fmt.Errorf("GitHub token is required")
	}

	if cfg.Owner == "" {
		return nil, fmt.Errorf("GitHub owner is required")
	}

	if cfg.Repository == "" {
		return nil, fmt.Errorf("GitHub repository is required")
	}

	// Create OAuth2 token source
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: cfg.Token},
	)
	tc := oauth2.NewClient(ctx, ts)

	// Create GitHub client
	var githubClient *github.Client
	if cfg.BaseURL != "" && cfg.BaseURL != "https://api.github.com" {
		// GitHub Enterprise
		githubClient, _ = github.NewClient(tc).WithEnterpriseURLs(cfg.BaseURL, cfg.BaseURL)
	} else {
		githubClient = github.NewClient(tc)
	}

	return &Client{
		client: githubClient,
		config: cfg,
		logger: logger,
	}, nil
}

// TestConnection tests the connection to GitHub
func (c *Client) TestConnection(ctx context.Context) error {
	c.logger.Info("Testing GitHub connection...")

	// Try to get repository information to test the connection
	_, _, err := c.client.Repositories.Get(ctx, c.config.Owner, c.config.Repository)
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	c.logger.Info("GitHub connection successful")
	return nil
}

// CreateIssue creates a new GitHub issue
func (c *Client) CreateIssue(ctx context.Context, issue *models.GitHubIssue) (*models.GitHubIssue, error) {
	c.logger.Debug("Creating GitHub issue", "issue", issue.Title)

	// Convert our model to GitHub API model
	githubIssue := &github.IssueRequest{
		Title:     &issue.Title,
		Body:      &issue.Body,
		Labels:    &issue.Labels,
		Assignees: &issue.Assignees,
	}

	if issue.Milestone != nil {
		githubIssue.Milestone = issue.Milestone
	}

	// Create the issue
	createdIssue, _, err := c.client.Issues.Create(ctx, c.config.Owner, c.config.Repository, githubIssue)
	if err != nil {
		return nil, fmt.Errorf("failed to create issue: %w", err)
	}

	// Convert back to our model
	result := &models.GitHubIssue{
		Number:     createdIssue.GetNumber(),
		Title:      createdIssue.GetTitle(),
		Body:       createdIssue.GetBody(),
		State:      createdIssue.GetState(),
		SourceWIID: issue.SourceWIID,
	}

	if createdIssue.CreatedAt != nil {
		result.CreatedAt = &createdIssue.CreatedAt.Time
	}

	if createdIssue.UpdatedAt != nil {
		result.UpdatedAt = &createdIssue.UpdatedAt.Time
	}

	c.logger.Info("Created GitHub issue", "issue", result.Number, "work item", issue.SourceWIID)
	return result, nil
}

// CreateIssueComment creates a comment on a GitHub issue
func (c *Client) CreateIssueComment(ctx context.Context, issueNumber int, comment *models.GitHubComment) error {
	c.logger.Debug("Creating comment on issue", "issue", issueNumber)

	githubComment := &github.IssueComment{
		Body: &comment.Body,
	}

	_, _, err := c.client.Issues.CreateComment(ctx, c.config.Owner, c.config.Repository, issueNumber, githubComment)
	if err != nil {
		return fmt.Errorf("failed to create comment on issue #%d: %w", issueNumber, err)
	}

	return nil
}

// UpdateIssueState updates the state of a GitHub issue
func (c *Client) UpdateIssueState(ctx context.Context, issueNumber int, state string) error {
	c.logger.Debug("Updating issue", "issue", issueNumber, "state", state)

	issueRequest := &github.IssueRequest{
		State: &state,
	}

	_, _, err := c.client.Issues.Edit(ctx, c.config.Owner, c.config.Repository, issueNumber, issueRequest)
	if err != nil {
		return fmt.Errorf("failed to update issue #%d state: %w", issueNumber, err)
	}

	return nil
}

// CreateLabel creates a new label in the repository if it doesn't exist
func (c *Client) CreateLabel(ctx context.Context, name, color, description string) error {
	c.logger.Debug("Creating/ensuring label", "label", name)

	// Check if label already exists
	_, resp, err := c.client.Issues.GetLabel(ctx, c.config.Owner, c.config.Repository, name)
	if err == nil {
		// Label already exists
		return nil
	}

	// If it's not a 404, it's a real error
	if resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("failed to check if label exists: %w", err)
	}

	// Create the label
	label := &github.Label{
		Name:        &name,
		Color:       &color,
		Description: &description,
	}

	_, _, err = c.client.Issues.CreateLabel(ctx, c.config.Owner, c.config.Repository, label)
	if err != nil {
		return fmt.Errorf("failed to create label %s: %w", name, err)
	}

	c.logger.Debug("created label", "label", name)
	return nil
}

// SearchIssues searches for issues that might already exist for a work item
func (c *Client) SearchIssues(ctx context.Context, workItemID int) ([]*github.Issue, error) {
	// Search for issues that contain the work item ID in the body
	query := fmt.Sprintf("repo:%s/%s \"%d\" in:body", c.config.Owner, c.config.Repository, workItemID)

	searchResult, _, err := c.client.Search.Issues(ctx, query, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to search for existing issues: %w", err)
	}

	return searchResult.Issues, nil
}

// ValidateLabels ensures all required labels exist in the repository
func (c *Client) ValidateLabels(ctx context.Context, labels []string) error {
	c.logger.Debug("Validating labels in repository")

	for _, label := range labels {
		_, resp, err := c.client.Issues.GetLabel(ctx, c.config.Owner, c.config.Repository, label)
		if err != nil && resp.StatusCode == http.StatusNotFound {
			// Label doesn't exist, create it with a default color
			if err := c.CreateLabel(ctx, label, "e1e4e8", fmt.Sprintf("Label for %s", label)); err != nil {
				return fmt.Errorf("failed to create missing label %s: %w", label, err)
			}
		} else if err != nil {
			return fmt.Errorf("failed to validate label %s: %w", label, err)
		}
	}

	return nil
}

// BuildIssueBody creates a formatted issue body with metadata
func (c *Client) BuildIssueBody(workItem *models.WorkItem, originalDescription string) string {
	var body strings.Builder

	// Add original description
	if originalDescription != "" {
		body.WriteString(originalDescription)
		body.WriteString("\n\n")
	}

	// Add migration metadata
	body.WriteString("---\n")
	body.WriteString("**Migration Information**\n")
	body.WriteString(fmt.Sprintf("- **Original Work Item ID:** %d\n", workItem.ID))
	body.WriteString(fmt.Sprintf("- **Work Item Type:** %s\n", workItem.GetWorkItemType()))
	body.WriteString(fmt.Sprintf("- **Original State:** %s\n", workItem.GetState()))

	if createdBy := workItem.GetCreatedBy(); createdBy != nil {
		body.WriteString(fmt.Sprintf("- **Originally Created By:** %s\n", createdBy.DisplayName))
	}

	if createdDate := workItem.GetCreatedDate(); createdDate != nil {
		body.WriteString(fmt.Sprintf("- **Originally Created:** %s\n", createdDate.Format("2006-01-02 15:04:05")))
	}

	if workItem.URL != "" {
		body.WriteString(fmt.Sprintf("- **Original URL:** %s\n", workItem.URL))
	}

	body.WriteString("\n*This issue was automatically migrated from Azure DevOps*")

	return body.String()
}
