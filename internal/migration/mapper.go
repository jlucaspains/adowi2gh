package migration

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jlucaspains/adowi2gh/internal/config"
	"github.com/jlucaspains/adowi2gh/internal/models"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
)

// Mapper handles the mapping between ADO work items and GitHub issues
type Mapper struct {
	config      *config.FieldMapping
	userMapping map[string]string
	logger      *slog.Logger
}

func NewMapper(cfg *config.MigrationConfig, logger *slog.Logger) *Mapper {
	return &Mapper{
		config:      &cfg.FieldMapping,
		userMapping: cfg.UserMapping,
		logger:      logger,
	}
}

func (m *Mapper) MapWorkItemToIssue(workItem *models.WorkItem) (*models.GitHubIssue, error) {
	issue := &models.GitHubIssue{
		SourceWIID: workItem.ID,
		Title:      workItem.GetTitle(),
		Body:       m.mapDescription(workItem),
		State:      m.mapState(workItem.GetState()),
		Labels:     m.mapLabels(workItem),
		Assignees:  m.mapAssignees(workItem),
	}

	// TODO: is metadata needed?
	if issue.Metadata == nil {
		issue.Metadata = make(map[string]interface{})
	}
	issue.Metadata["original_id"] = workItem.ID
	issue.Metadata["original_type"] = workItem.GetWorkItemType()
	issue.Metadata["original_url"] = workItem.URL

	return issue, nil
}

func (m *Mapper) mapDescription(workItem *models.WorkItem) string {
	// TODO: add support for images
	importedDescription := fmt.Sprintf("> Issue imported from Azure DevOps [#%d](%s)", workItem.ID, workItem.URL)
	description := workItem.GetDescription()

	// Clean up HTML if present
	description = importedDescription + "\n\n" + m.cleanHtmlContent(description)

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

func (m *Mapper) mapState(adoState string) string {
	if m.config.StateMapping != nil {
		if githubState, exists := m.config.StateMapping[adoState]; exists {
			return githubState
		}
	}

	switch strings.ToLower(adoState) {
	case "new", "active", "approved", "committed", "in progress", "resolved":
		return "open"
	case "done", "closed", "removed":
		return "closed"
	default:
		return "open"
	}
}

func (m *Mapper) mapLabels(workItem *models.WorkItem) []string {
	var labels []string = []string{}

	// Map work item type to labels
	workItemType := strings.ToLower(workItem.GetWorkItemType())
	if m.config.TypeMapping != nil {
		if typeLabels, exists := m.config.TypeMapping[workItemType]; exists {
			labels = append(labels, typeLabels...)
		}
	}

	// Map priority to labels
	if priority, ok := workItem.Fields["Microsoft.VSTS.Common.Priority"].(string); ok {
		if m.config.PriorityMapping != nil {
			if priorityLabels, exists := m.config.PriorityMapping[priority]; exists {
				labels = append(labels, priorityLabels...)
			}
		}
	}

	// Map severity to labels (for bugs)
	if severity, ok := workItem.Fields["Microsoft.VSTS.Common.Severity"].(string); ok && m.config.IncludeSeverityLabel {
		labels = append(labels, fmt.Sprintf("severity:%s", strings.ToLower(severity)))
	}

	// Add area path as label
	if areaPath, ok := workItem.Fields["System.AreaPath"].(string); ok && m.config.IncludeAreaPathLabel {
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

	labels = m.deduplicateLabels(labels)

	return labels
}

func (m *Mapper) mapAssignees(workItem *models.WorkItem) []string {
	var assignees []string = []string{}

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

	return assignees
}

func (m *Mapper) MapComments(workItemComments []models.WorkItemComment) []models.GitHubComment {
	// TODO: add support for images
	var githubComments []models.GitHubComment
	loc, err := time.LoadLocation(m.config.TimeZone)

	if err != nil {
		m.logger.Warn("Error loading location. Assuming server local", "error", err)
		loc = time.Local
	}

	for _, comment := range workItemComments {
		githubComment := models.GitHubComment{
			Body: m.cleanHtmlContent(comment.Text),
		}

		commentTime := comment.CreatedDate.In(loc).Format("2006-01-02 15:04:05 MST")
		if comment.CreatedBy.DisplayName != "" {
			githubComment.Body = fmt.Sprintf("*Comment by %s on %s:*\n\n%s",
				comment.CreatedBy.DisplayName, commentTime, githubComment.Body)
		}

		githubComments = append(githubComments, githubComment)
	}

	return githubComments
}

func (m *Mapper) cleanHtmlContent(content string) string {
	if content == "" {
		return ""
	}

	content, err := htmltomarkdown.ConvertString(content)
	if err != nil {
		m.logger.Error("Failed to convert HTML to Markdown", "error", err, "content", content)
		return ""
	}

	content = strings.TrimSpace(content)

	return content
}

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
