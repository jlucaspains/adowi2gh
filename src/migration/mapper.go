package migration

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"ado-gh-wi-migrator/config"
	"ado-gh-wi-migrator/models"
)

// Mapper handles the mapping between ADO work items and GitHub issues
type Mapper struct {
	config      *config.FieldMapping
	userMapping map[string]string
	logger      *slog.Logger
}

// NewMapper creates a new field mapper
func NewMapper(cfg *config.MigrationConfig, logger *slog.Logger) *Mapper {
	return &Mapper{
		config:      &cfg.FieldMapping,
		userMapping: cfg.UserMapping,
		logger:      logger,
	}
}

// MapWorkItemToIssue converts an ADO work item to a GitHub issue
func (m *Mapper) MapWorkItemToIssue(workItem *models.WorkItem) (*models.GitHubIssue, error) {
	issue := &models.GitHubIssue{
		SourceWIID: workItem.ID,
		Title:      workItem.GetTitle(),
		Body:       m.mapDescription(workItem),
		State:      m.mapState(workItem.GetState()),
		Labels:     m.mapLabels(workItem),
		Assignees:  m.mapAssignees(workItem),
	}

	// Add any additional metadata
	if issue.Metadata == nil {
		issue.Metadata = make(map[string]interface{})
	}
	issue.Metadata["original_id"] = workItem.ID
	issue.Metadata["original_type"] = workItem.GetWorkItemType()
	issue.Metadata["original_url"] = workItem.URL

	return issue, nil
}

// mapDescription maps the work item description to issue body
func (m *Mapper) mapDescription(workItem *models.WorkItem) string {
	description := workItem.GetDescription()

	// Clean up HTML if present
	description = m.cleanHtmlContent(description)

	// Add acceptance criteria if present
	if acceptanceCriteria, ok := workItem.Fields["Microsoft.VSTS.Common.AcceptanceCriteria"].(string); ok && acceptanceCriteria != "" {
		description += "\n\n## Acceptance Criteria\n" + m.cleanHtmlContent(acceptanceCriteria)
	}

	// Add reproduction steps if present
	if repro, ok := workItem.Fields["Microsoft.VSTS.TCM.ReproSteps"].(string); ok && repro != "" {
		description += "\n\n## Reproduction Steps\n" + m.cleanHtmlContent(repro)
	}

	return description
}

// mapState maps ADO work item state to GitHub issue state
func (m *Mapper) mapState(adoState string) string {
	if m.config.StateMapping != nil {
		if githubState, exists := m.config.StateMapping[adoState]; exists {
			return githubState
		}
	}

	// Default mapping
	switch strings.ToLower(adoState) {
	case "new", "active", "approved", "committed", "in progress", "resolved":
		return "open"
	case "done", "closed", "removed":
		return "closed"
	default:
		return "open"
	}
}

// mapLabels generates GitHub labels based on work item properties
func (m *Mapper) mapLabels(workItem *models.WorkItem) []string {
	var labels []string

	// Map work item type to labels
	workItemType := strings.ToLower(workItem.GetWorkItemType())
	if m.config.TypeMapping != nil {
		if typeLabels, exists := m.config.TypeMapping[workItemType]; exists {
			labels = append(labels, typeLabels...)
		}
	} else {
		// Default type mapping
		switch strings.ToLower(workItemType) {
		case "bug":
			labels = append(labels, "bug")
		case "feature", "user story":
			labels = append(labels, "enhancement")
		case "task":
			labels = append(labels, "task")
		case "epic":
			labels = append(labels, "epic")
		}
	}

	// Map priority to labels
	if priority, ok := workItem.Fields["Microsoft.VSTS.Common.Priority"].(string); ok {
		if m.config.PriorityMapping != nil {
			if priorityLabels, exists := m.config.PriorityMapping[priority]; exists {
				labels = append(labels, priorityLabels...)
			}
		} else {
			// Default priority mapping
			switch priority {
			case "1":
				labels = append(labels, "priority:critical")
			case "2":
				labels = append(labels, "priority:high")
			case "3":
				labels = append(labels, "priority:medium")
			case "4":
				labels = append(labels, "priority:low")
			}
		}
	}

	// Map severity to labels (for bugs)
	if severity, ok := workItem.Fields["Microsoft.VSTS.Common.Severity"].(string); ok {
		labels = append(labels, fmt.Sprintf("severity:%s", strings.ToLower(severity)))
	}

	// Add area path as label
	if areaPath, ok := workItem.Fields["System.AreaPath"].(string); ok {
		// Extract the last part of the area path
		pathParts := strings.Split(areaPath, "\\")
		if len(pathParts) > 1 {
			areaLabel := fmt.Sprintf("area:%s", strings.ToLower(pathParts[len(pathParts)-1]))
			labels = append(labels, areaLabel)
		}
	}

	// Add tags as labels
	tags := workItem.GetTags()
	for _, tag := range tags {
		if tag != "" {
			labels = append(labels, strings.ToLower(strings.TrimSpace(tag)))
		}
	}

	// Remove duplicates and empty labels
	labels = m.deduplicateLabels(labels)

	return labels
}

// mapAssignees maps ADO assigned user to GitHub assignees
func (m *Mapper) mapAssignees(workItem *models.WorkItem) []string {
	var assignees []string

	assignedTo := workItem.GetAssignedTo()
	if assignedTo == nil {
		return assignees
	}

	// Try to map using configured user mapping first
	if m.userMapping != nil {
		// Try different variations of the user identifier
		candidates := []string{
			strings.ToLower(assignedTo.UniqueName),
			strings.ToLower(assignedTo.Email),
			strings.ToLower(assignedTo.DisplayName),
		}

		for _, candidate := range candidates {
			if githubUser, exists := m.userMapping[candidate]; exists {
				assignees = append(assignees, githubUser)
				return assignees
			}
		}
	}

	// If no mapping found, try to extract GitHub username from email
	if assignedTo.Email != "" {
		githubUser := m.extractGitHubUsernameFromEmail(assignedTo.Email)
		if githubUser != "" {
			assignees = append(assignees, githubUser)
		}
	}

	return assignees
}

// MapComments maps ADO work item comments to GitHub issue comments
func (m *Mapper) MapComments(workItemComments []models.WorkItemComment) []models.GitHubComment {
	var githubComments []models.GitHubComment

	for _, comment := range workItemComments {
		githubComment := models.GitHubComment{
			Body: m.cleanHtmlContent(comment.Text),
		}

		// TODO: adjust the date format
		// Add metadata about original author if mapping wasn't found
		if comment.CreatedBy.DisplayName != "" {
			githubComment.Body = fmt.Sprintf("*Comment by %s on %s:*\n\n%s",
				comment.CreatedBy.DisplayName, comment.CreatedDate, githubComment.Body)
		}

		githubComments = append(githubComments, githubComment)
	}

	return githubComments
}

// cleanHtmlContent removes or converts HTML content to Markdown
func (m *Mapper) cleanHtmlContent(content string) string {
	if content == "" {
		return ""
	}

	// Basic HTML to Markdown conversion
	content = m.convertHtmlToMarkdown(content)

	// Remove extra whitespace
	content = strings.TrimSpace(content)

	return content
}

// convertHtmlToMarkdown performs basic HTML to Markdown conversion
func (m *Mapper) convertHtmlToMarkdown(html string) string {
	// This is a simplified conversion - you might want to use a proper HTML to Markdown library
	// TODO: improve html to markdown conversion

	// Convert common HTML tags
	replacements := map[string]string{
		"<br>":      "\n",
		"<br/>":     "\n",
		"<br />":    "\n",
		"<p>":       "\n",
		"</p>":      "\n",
		"<strong>":  "**",
		"</strong>": "**",
		"<b>":       "**",
		"</b>":      "**",
		"<em>":      "*",
		"</em>":     "*",
		"<i>":       "*",
		"</i>":      "*",
		"<ul>":      "\n",
		"</ul>":     "\n",
		"<ol>":      "\n",
		"</ol>":     "\n",
		"<li>":      "- ",
		"</li>":     "\n",
	}

	result := html
	for htmlTag, markdown := range replacements {
		result = strings.ReplaceAll(result, htmlTag, markdown)
	}

	// Remove remaining HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	result = re.ReplaceAllString(result, "")

	// Clean up multiple newlines
	re = regexp.MustCompile(`\n\s*\n\s*\n`)
	result = re.ReplaceAllString(result, "\n\n")

	return strings.TrimSpace(result)
}

// extractGitHubUsernameFromEmail tries to extract a GitHub username from an email
func (m *Mapper) extractGitHubUsernameFromEmail(email string) string {
	if email == "" {
		return ""
	}

	// If email is from GitHub, extract username
	if strings.HasSuffix(email, "@users.noreply.github.com") {
		parts := strings.Split(email, "@")
		if len(parts) > 0 {
			return parts[0]
		}
	}

	// Extract username part from email (before @)
	parts := strings.Split(email, "@")
	if len(parts) > 0 {
		username := parts[0]
		// Clean up username (GitHub usernames can only contain alphanumeric characters and hyphens)
		re := regexp.MustCompile(`[^a-zA-Z0-9-]`)
		username = re.ReplaceAllString(username, "-")
		return username
	}

	return ""
}

// deduplicateLabels removes duplicate labels while preserving order
func (m *Mapper) deduplicateLabels(labels []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, label := range labels {
		if label != "" && !seen[label] {
			seen[label] = true
			result = append(result, label)
		}
	}

	return result
}
