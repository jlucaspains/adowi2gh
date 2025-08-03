package migration

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jlucaspains/adowi2gh/config"
	"github.com/jlucaspains/adowi2gh/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMapper(t *testing.T) {
	cfg := &config.MigrationConfig{
		FieldMapping: config.FieldMapping{
			TimeZone: "UTC",
		},
		UserMapping: map[string]string{
			"user@example.com": "githubuser",
		},
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	mapper := NewMapper(cfg, logger)

	assert.NotNil(t, mapper)
	assert.Equal(t, &cfg.FieldMapping, mapper.config)
	assert.Equal(t, cfg.UserMapping, mapper.userMapping)
	assert.Equal(t, logger, mapper.logger)
}

func TestMapWorkItemToIssue(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("basic mapping", func(t *testing.T) {
		cfg := &config.MigrationConfig{
			FieldMapping: config.FieldMapping{
				TimeZone: "UTC",
			},
		}
		mapper := NewMapper(cfg, logger)

		workItem := &models.WorkItem{
			ID:  123,
			URL: "https://dev.azure.com/org/project/_workitems/edit/123",
			Fields: map[string]interface{}{
				"System.Title":        "Test Bug",
				"System.Description":  "This is a test bug description",
				"System.State":        "New",
				"System.WorkItemType": "Bug",
			},
		}

		issue, err := mapper.MapWorkItemToIssue(workItem)

		require.NoError(t, err)
		assert.Equal(t, 123, issue.SourceWIID)
		assert.Equal(t, "Test Bug", issue.Title)
		assert.Contains(t, issue.Body, "Issue imported from Azure DevOps")
		assert.Contains(t, issue.Body, "#123")
		assert.Contains(t, issue.Body, "This is a test bug description")
		assert.Equal(t, "open", issue.State)
		assert.NotNil(t, issue.Metadata)
		assert.Equal(t, 123, issue.Metadata["original_id"])
		assert.Equal(t, "Bug", issue.Metadata["original_type"])
		assert.Equal(t, "https://dev.azure.com/org/project/_workitems/edit/123", issue.Metadata["original_url"])
	})

	t.Run("with acceptance criteria", func(t *testing.T) {
		cfg := &config.MigrationConfig{
			FieldMapping: config.FieldMapping{
				TimeZone: "UTC",
			},
		}
		mapper := NewMapper(cfg, logger)

		workItem := &models.WorkItem{
			ID:  456,
			URL: "https://dev.azure.com/org/project/_workitems/edit/456",
			Fields: map[string]interface{}{
				"System.Title":                             "Feature Request",
				"System.Description":                       "Add new feature",
				"System.State":                             "Active",
				"System.WorkItemType":                      "Feature",
				"Microsoft.VSTS.Common.AcceptanceCriteria": "<p>Given user clicks button</p><p>Then action happens</p>",
			},
		}

		issue, err := mapper.MapWorkItemToIssue(workItem)

		require.NoError(t, err)
		assert.Contains(t, issue.Body, "## Acceptance Criteria")
		assert.Contains(t, issue.Body, "Given user clicks button")
		assert.Contains(t, issue.Body, "Then action happens")
	})

	t.Run("with reproduction steps", func(t *testing.T) {
		cfg := &config.MigrationConfig{
			FieldMapping: config.FieldMapping{
				TimeZone: "UTC",
			},
		}
		mapper := NewMapper(cfg, logger)

		workItem := &models.WorkItem{
			ID:  789,
			URL: "https://dev.azure.com/org/project/_workitems/edit/789",
			Fields: map[string]interface{}{
				"System.Title":                  "Bug Report",
				"System.Description":            "Something is broken",
				"System.State":                  "New",
				"System.WorkItemType":           "Bug",
				"Microsoft.VSTS.TCM.ReproSteps": "<ol><li>Step 1</li><li>Step 2</li></ol>",
			},
		}

		issue, err := mapper.MapWorkItemToIssue(workItem)

		require.NoError(t, err)
		assert.Contains(t, issue.Body, "## Reproduction Steps")
		assert.Contains(t, issue.Body, "1. Step 1")
		assert.Contains(t, issue.Body, "2. Step 2")
	})
}

func TestMapState(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("with custom state mapping", func(t *testing.T) {
		cfg := &config.MigrationConfig{
			FieldMapping: config.FieldMapping{
				StateMapping: map[string]string{
					"New":    "open",
					"Closed": "closed",
					"Done":   "closed",
				},
				TimeZone: "UTC",
			},
		}
		mapper := NewMapper(cfg, logger)

		assert.Equal(t, "open", mapper.mapState("New"))
		assert.Equal(t, "closed", mapper.mapState("Closed"))
		assert.Equal(t, "closed", mapper.mapState("Done"))
	})

	t.Run("default state mapping", func(t *testing.T) {
		cfg := &config.MigrationConfig{
			FieldMapping: config.FieldMapping{
				TimeZone: "UTC",
			},
		}
		mapper := NewMapper(cfg, logger)

		// Test default open states
		assert.Equal(t, "open", mapper.mapState("New"))
		assert.Equal(t, "open", mapper.mapState("Active"))
		assert.Equal(t, "open", mapper.mapState("Approved"))
		assert.Equal(t, "open", mapper.mapState("Committed"))
		assert.Equal(t, "open", mapper.mapState("In Progress"))
		assert.Equal(t, "open", mapper.mapState("Resolved"))

		// Test default closed states
		assert.Equal(t, "closed", mapper.mapState("Done"))
		assert.Equal(t, "closed", mapper.mapState("Closed"))
		assert.Equal(t, "closed", mapper.mapState("Removed"))

		// Test case insensitive
		assert.Equal(t, "open", mapper.mapState("new"))
		assert.Equal(t, "closed", mapper.mapState("done"))

		// Test unknown state defaults to open
		assert.Equal(t, "open", mapper.mapState("Unknown"))
	})
}

func TestMapLabels(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("with type mapping", func(t *testing.T) {
		cfg := &config.MigrationConfig{
			FieldMapping: config.FieldMapping{
				TypeMapping: map[string][]string{
					"bug":     {"bug", "defect"},
					"feature": {"enhancement"},
				},
				TimeZone: "UTC",
			},
		}
		mapper := NewMapper(cfg, logger)

		workItem := &models.WorkItem{
			Fields: map[string]interface{}{
				"System.WorkItemType": "Bug",
			},
		}

		labels := mapper.mapLabels(workItem)
		assert.Contains(t, labels, "bug")
		assert.Contains(t, labels, "defect")
	})

	t.Run("with priority mapping", func(t *testing.T) {
		cfg := &config.MigrationConfig{
			FieldMapping: config.FieldMapping{
				PriorityMapping: map[string][]string{
					"1": {"priority:critical"},
					"2": {"priority:high"},
				},
				TimeZone: "UTC",
			},
		}
		mapper := NewMapper(cfg, logger)

		workItem := &models.WorkItem{
			Fields: map[string]interface{}{
				"System.WorkItemType":            "Bug",
				"Microsoft.VSTS.Common.Priority": "1",
			},
		}

		labels := mapper.mapLabels(workItem)
		assert.Contains(t, labels, "priority:critical")
	})

	t.Run("with severity label", func(t *testing.T) {
		cfg := &config.MigrationConfig{
			FieldMapping: config.FieldMapping{
				IncludeSeverityLabel: true,
				TimeZone:             "UTC",
			},
		}
		mapper := NewMapper(cfg, logger)

		workItem := &models.WorkItem{
			Fields: map[string]interface{}{
				"System.WorkItemType":            "Bug",
				"Microsoft.VSTS.Common.Severity": "1 - Critical",
			},
		}

		labels := mapper.mapLabels(workItem)
		assert.Contains(t, labels, "severity:1 - critical")
	})

	t.Run("with area path label", func(t *testing.T) {
		cfg := &config.MigrationConfig{
			FieldMapping: config.FieldMapping{
				IncludeAreaPathLabel: true,
				TimeZone:             "UTC",
			},
		}
		mapper := NewMapper(cfg, logger)

		workItem := &models.WorkItem{
			Fields: map[string]interface{}{
				"System.WorkItemType": "Bug",
				"System.AreaPath":     "MyProject\\Frontend\\UI",
			},
		}

		labels := mapper.mapLabels(workItem)
		assert.Contains(t, labels, "area:ui")
	})

	t.Run("with tags", func(t *testing.T) {
		cfg := &config.MigrationConfig{
			FieldMapping: config.FieldMapping{
				TimeZone: "UTC",
			},
		}
		mapper := NewMapper(cfg, logger)

		workItem := &models.WorkItem{
			Fields: map[string]interface{}{
				"System.WorkItemType": "Bug",
				"System.Tags":         "urgent; needs-review; customer-reported",
			},
		}

		labels := mapper.mapLabels(workItem)
		assert.Contains(t, labels, "urgent")
		assert.Contains(t, labels, "needs-review")
		assert.Contains(t, labels, "customer-reported")
	})

	t.Run("deduplicates labels", func(t *testing.T) {
		cfg := &config.MigrationConfig{
			FieldMapping: config.FieldMapping{
				TypeMapping: map[string][]string{
					"bug": {"bug"},
				},
				TimeZone: "UTC",
			},
		}
		mapper := NewMapper(cfg, logger)

		workItem := &models.WorkItem{
			Fields: map[string]interface{}{
				"System.WorkItemType": "Bug",
				"System.Tags":         "bug; urgent",
			},
		}

		labels := mapper.mapLabels(workItem)
		// Should only contain "bug" once
		bugCount := 0
		for _, label := range labels {
			if label == "bug" {
				bugCount++
			}
		}
		assert.Equal(t, 1, bugCount)
		assert.Contains(t, labels, "urgent")
	})
}

func TestMapAssignees(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("with user mapping by email", func(t *testing.T) {
		cfg := &config.MigrationConfig{
			FieldMapping: config.FieldMapping{
				TimeZone: "UTC",
			},
			UserMapping: map[string]string{
				"john.doe@example.com": "johndoe",
			},
		}
		mapper := NewMapper(cfg, logger)

		workItem := &models.WorkItem{
			Fields: map[string]interface{}{
				"System.AssignedTo": map[string]interface{}{
					"id":          "user-123",
					"displayName": "John Doe",
					"email":       "john.doe@example.com",
					"uniqueName":  "john.doe@example.com",
				},
			},
		}

		assignees := mapper.mapAssignees(workItem)
		assert.Equal(t, []string{"johndoe"}, assignees)
	})

	t.Run("with user mapping by uniqueName", func(t *testing.T) {
		cfg := &config.MigrationConfig{
			FieldMapping: config.FieldMapping{
				TimeZone: "UTC",
			},
			UserMapping: map[string]string{
				"domain\\john.doe": "johndoe",
			},
		}
		mapper := NewMapper(cfg, logger)

		workItem := &models.WorkItem{
			Fields: map[string]interface{}{
				"System.AssignedTo": map[string]interface{}{
					"id":          "user-123",
					"displayName": "John Doe",
					"email":       "john.doe@example.com",
					"uniqueName":  "DOMAIN\\john.doe",
				},
			},
		}

		assignees := mapper.mapAssignees(workItem)
		assert.Equal(t, []string{"johndoe"}, assignees)
	})

	t.Run("with user mapping by displayName", func(t *testing.T) {
		cfg := &config.MigrationConfig{
			FieldMapping: config.FieldMapping{
				TimeZone: "UTC",
			},
			UserMapping: map[string]string{
				"john doe": "johndoe",
			},
		}
		mapper := NewMapper(cfg, logger)

		workItem := &models.WorkItem{
			Fields: map[string]interface{}{
				"System.AssignedTo": map[string]interface{}{
					"id":          "user-123",
					"displayName": "John Doe",
					"email":       "john.doe@example.com",
					"uniqueName":  "john.doe@example.com",
				},
			},
		}

		assignees := mapper.mapAssignees(workItem)
		assert.Equal(t, []string{"johndoe"}, assignees)
	})

	t.Run("no mapping found", func(t *testing.T) {
		cfg := &config.MigrationConfig{
			FieldMapping: config.FieldMapping{
				TimeZone: "UTC",
			},
			UserMapping: map[string]string{
				"other@example.com": "otheruser",
			},
		}
		mapper := NewMapper(cfg, logger)

		workItem := &models.WorkItem{
			Fields: map[string]interface{}{
				"System.AssignedTo": map[string]interface{}{
					"id":          "user-123",
					"displayName": "John Doe",
					"email":       "john.doe@example.com",
					"uniqueName":  "john.doe@example.com",
				},
			},
		}

		assignees := mapper.mapAssignees(workItem)
		assert.Empty(t, assignees)
	})

	t.Run("no assigned user", func(t *testing.T) {
		cfg := &config.MigrationConfig{
			FieldMapping: config.FieldMapping{
				TimeZone: "UTC",
			},
		}
		mapper := NewMapper(cfg, logger)

		workItem := &models.WorkItem{
			Fields: map[string]interface{}{},
		}

		assignees := mapper.mapAssignees(workItem)
		assert.Empty(t, assignees)
	})
}

func TestMapComments(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("maps comments with timezone", func(t *testing.T) {
		cfg := &config.MigrationConfig{
			FieldMapping: config.FieldMapping{
				TimeZone: "America/New_York",
			},
		}
		mapper := NewMapper(cfg, logger)

		createdDate := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		comments := []models.WorkItemComment{
			{
				Text:        "<p>This is a <strong>HTML</strong> comment</p>",
				CreatedDate: createdDate,
				CreatedBy: models.User{
					DisplayName: "Jane Smith",
					Email:       "jane@example.com",
				},
			},
		}

		githubComments := mapper.MapComments(comments)

		require.Len(t, githubComments, 1)
		assert.Contains(t, githubComments[0].Body, "Comment by Jane Smith")
		assert.Contains(t, githubComments[0].Body, "This is a **HTML** comment")
		assert.Contains(t, githubComments[0].Body, "2024-01-15")
	})

	t.Run("handles invalid timezone gracefully", func(t *testing.T) {
		cfg := &config.MigrationConfig{
			FieldMapping: config.FieldMapping{
				TimeZone: "Invalid/Timezone",
			},
		}
		mapper := NewMapper(cfg, logger)

		createdDate := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		comments := []models.WorkItemComment{
			{
				Text:        "Simple comment",
				CreatedDate: createdDate,
				CreatedBy: models.User{
					DisplayName: "John Doe",
				},
			},
		}

		// Should not panic and use local time
		githubComments := mapper.MapComments(comments)
		require.Len(t, githubComments, 1)
		assert.Contains(t, githubComments[0].Body, "Comment by John Doe")
	})

	t.Run("handles empty comments", func(t *testing.T) {
		cfg := &config.MigrationConfig{
			FieldMapping: config.FieldMapping{
				TimeZone: "UTC",
			},
		}
		mapper := NewMapper(cfg, logger)

		comments := []models.WorkItemComment{}
		githubComments := mapper.MapComments(comments)
		assert.Empty(t, githubComments)
	})
}

func TestCleanHtmlContent(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := &config.MigrationConfig{
		FieldMapping: config.FieldMapping{
			TimeZone: "UTC",
		},
	}
	mapper := NewMapper(cfg, logger)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty content",
			input:    "",
			expected: "",
		},
		{
			name:     "plain text",
			input:    "This is plain text",
			expected: "This is plain text",
		},
		{
			name:     "HTML with bold",
			input:    "<p>This is <strong>bold</strong> text</p>",
			expected: "This is **bold** text",
		},
		{
			name:     "HTML with list",
			input:    "<ul><li>Item 1</li><li>Item 2</li></ul>",
			expected: "- Item 1\n- Item 2",
		},
		{
			name:     "HTML with ordered list",
			input:    "<ol><li>Step 1</li><li>Step 2</li></ol>",
			expected: "1. Step 1\n2. Step 2",
		},
		{
			name:     "Complex HTML",
			input:    "<div><p>Paragraph with <em>italic</em> and <strong>bold</strong></p><br/><p>Second paragraph</p></div>",
			expected: "Paragraph with *italic* and **bold**\n\nSecond paragraph",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapper.cleanHtmlContent(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDeduplicateLabels(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := &config.MigrationConfig{
		FieldMapping: config.FieldMapping{
			TimeZone: "UTC",
		},
	}
	mapper := NewMapper(cfg, logger)

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "no duplicates",
			input:    []string{"bug", "urgent", "frontend"},
			expected: []string{"bug", "urgent", "frontend"},
		},
		{
			name:     "with duplicates",
			input:    []string{"bug", "urgent", "bug", "frontend", "urgent"},
			expected: []string{"bug", "urgent", "frontend"},
		},
		{
			name:     "with empty strings",
			input:    []string{"bug", "", "urgent", "", "frontend"},
			expected: []string{"bug", "urgent", "frontend"},
		},
		{
			name:     "empty input",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "only empty strings",
			input:    []string{"", "", ""},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapper.deduplicateLabels(tt.input)
			if len(tt.expected) == 0 && len(result) == 0 {
				// Both are empty - either nil or empty slice is fine
				return
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test integration scenarios
func TestMapperIntegration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("complete work item mapping", func(t *testing.T) {
		cfg := &config.MigrationConfig{
			FieldMapping: config.FieldMapping{
				StateMapping: map[string]string{
					"New":    "open",
					"Closed": "closed",
				},
				TypeMapping: map[string][]string{
					"bug": {"bug", "defect"},
				},
				PriorityMapping: map[string][]string{
					"1": {"priority:critical"},
				},
				IncludeSeverityLabel: true,
				IncludeAreaPathLabel: true,
				TimeZone:             "UTC",
			},
			UserMapping: map[string]string{
				"john.doe@example.com": "johndoe",
			},
		}
		mapper := NewMapper(cfg, logger)

		workItem := &models.WorkItem{
			ID:  12345,
			URL: "https://dev.azure.com/org/project/_workitems/edit/12345",
			Fields: map[string]interface{}{
				"System.Title":                             "Critical bug in login",
				"System.Description":                       "<p>User cannot <strong>login</strong> to the application</p>",
				"System.State":                             "New",
				"System.WorkItemType":                      "Bug",
				"System.Tags":                              "urgent; customer-facing",
				"System.AreaPath":                          "MyProject\\Authentication\\Login",
				"Microsoft.VSTS.Common.Priority":           "1",
				"Microsoft.VSTS.Common.Severity":           "1 - Critical",
				"Microsoft.VSTS.Common.AcceptanceCriteria": "<p>User should be able to login successfully</p>",
				"Microsoft.VSTS.TCM.ReproSteps":            "<ol><li>Go to login page</li><li>Enter credentials</li><li>Click login</li></ol>",
				"System.AssignedTo": map[string]interface{}{
					"displayName": "John Doe",
					"email":       "john.doe@example.com",
					"uniqueName":  "john.doe@example.com",
				},
			},
		}

		issue, err := mapper.MapWorkItemToIssue(workItem)

		require.NoError(t, err)
		assert.Equal(t, "Critical bug in login", issue.Title)
		assert.Contains(t, issue.Body, "User cannot **login** to the application")
		assert.Contains(t, issue.Body, "## Acceptance Criteria")
		assert.Contains(t, issue.Body, "## Reproduction Steps")
		assert.Equal(t, "open", issue.State)
		assert.Contains(t, issue.Labels, "bug")
		assert.Contains(t, issue.Labels, "defect")
		assert.Contains(t, issue.Labels, "priority:critical")
		assert.Contains(t, issue.Labels, "severity:1 - critical")
		assert.Contains(t, issue.Labels, "area:login")
		assert.Contains(t, issue.Labels, "urgent")
		assert.Contains(t, issue.Labels, "customer-facing")
		assert.Equal(t, []string{"johndoe"}, issue.Assignees)
		assert.Equal(t, 12345, issue.Metadata["original_id"])
		assert.Equal(t, "Bug", issue.Metadata["original_type"])
	})
}
