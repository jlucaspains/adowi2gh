package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkItem_GetTitle(t *testing.T) {
	t.Run("returns title when present", func(t *testing.T) {
		workItem := &WorkItem{
			Fields: map[string]interface{}{
				"System.Title": "Test Work Item Title",
			},
		}

		title := workItem.GetTitle()
		assert.Equal(t, "Test Work Item Title", title)
	})

	t.Run("returns empty string when title is missing", func(t *testing.T) {
		workItem := &WorkItem{
			Fields: map[string]interface{}{},
		}

		title := workItem.GetTitle()
		assert.Equal(t, "", title)
	})

	t.Run("returns empty string when title is not a string", func(t *testing.T) {
		workItem := &WorkItem{
			Fields: map[string]interface{}{
				"System.Title": 12345,
			},
		}

		title := workItem.GetTitle()
		assert.Equal(t, "", title)
	})

	t.Run("returns empty string when title is nil", func(t *testing.T) {
		workItem := &WorkItem{
			Fields: map[string]interface{}{
				"System.Title": nil,
			},
		}

		title := workItem.GetTitle()
		assert.Equal(t, "", title)
	})
}

func TestWorkItem_GetDescription(t *testing.T) {
	t.Run("returns description when present", func(t *testing.T) {
		workItem := &WorkItem{
			Fields: map[string]interface{}{
				"System.Description": "This is a detailed description of the work item",
			},
		}

		description := workItem.GetDescription()
		assert.Equal(t, "This is a detailed description of the work item", description)
	})

	t.Run("returns empty string when description is missing", func(t *testing.T) {
		workItem := &WorkItem{
			Fields: map[string]interface{}{},
		}

		description := workItem.GetDescription()
		assert.Equal(t, "", description)
	})

	t.Run("returns empty string when description is not a string", func(t *testing.T) {
		workItem := &WorkItem{
			Fields: map[string]interface{}{
				"System.Description": []string{"not", "a", "string"},
			},
		}

		description := workItem.GetDescription()
		assert.Equal(t, "", description)
	})

	t.Run("handles HTML description", func(t *testing.T) {
		workItem := &WorkItem{
			Fields: map[string]interface{}{
				"System.Description": "<p>HTML <strong>description</strong> with tags</p>",
			},
		}

		description := workItem.GetDescription()
		assert.Equal(t, "<p>HTML <strong>description</strong> with tags</p>", description)
	})
}

func TestWorkItem_GetWorkItemType(t *testing.T) {
	t.Run("returns work item type when present", func(t *testing.T) {
		workItem := &WorkItem{
			Fields: map[string]interface{}{
				"System.WorkItemType": "Bug",
			},
		}

		wiType := workItem.GetWorkItemType()
		assert.Equal(t, "Bug", wiType)
	})

	t.Run("returns empty string when work item type is missing", func(t *testing.T) {
		workItem := &WorkItem{
			Fields: map[string]interface{}{},
		}

		wiType := workItem.GetWorkItemType()
		assert.Equal(t, "", wiType)
	})

	t.Run("returns empty string when work item type is not a string", func(t *testing.T) {
		workItem := &WorkItem{
			Fields: map[string]interface{}{
				"System.WorkItemType": 42,
			},
		}

		wiType := workItem.GetWorkItemType()
		assert.Equal(t, "", wiType)
	})

	t.Run("handles different work item types", func(t *testing.T) {
		testCases := []string{"Bug", "Feature", "Task", "User Story", "Epic"}

		for _, expectedType := range testCases {
			workItem := &WorkItem{
				Fields: map[string]interface{}{
					"System.WorkItemType": expectedType,
				},
			}

			wiType := workItem.GetWorkItemType()
			assert.Equal(t, expectedType, wiType)
		}
	})
}

func TestWorkItem_GetState(t *testing.T) {
	t.Run("returns state when present", func(t *testing.T) {
		workItem := &WorkItem{
			Fields: map[string]interface{}{
				"System.State": "Active",
			},
		}

		state := workItem.GetState()
		assert.Equal(t, "Active", state)
	})

	t.Run("returns empty string when state is missing", func(t *testing.T) {
		workItem := &WorkItem{
			Fields: map[string]interface{}{},
		}

		state := workItem.GetState()
		assert.Equal(t, "", state)
	})

	t.Run("returns empty string when state is not a string", func(t *testing.T) {
		workItem := &WorkItem{
			Fields: map[string]interface{}{
				"System.State": true,
			},
		}

		state := workItem.GetState()
		assert.Equal(t, "", state)
	})

	t.Run("handles different states", func(t *testing.T) {
		testCases := []string{"New", "Active", "Resolved", "Closed", "Removed"}

		for _, expectedState := range testCases {
			workItem := &WorkItem{
				Fields: map[string]interface{}{
					"System.State": expectedState,
				},
			}

			state := workItem.GetState()
			assert.Equal(t, expectedState, state)
		}
	})
}

func TestWorkItem_GetAssignedTo(t *testing.T) {
	t.Run("returns user when assigned to is present", func(t *testing.T) {
		workItem := &WorkItem{
			Fields: map[string]interface{}{
				"System.AssignedTo": map[string]interface{}{
					"id":          "user-123",
					"displayName": "John Doe",
					"email":       "john.doe@example.com",
					"uniqueName":  "john.doe@example.com",
				},
			},
		}

		user := workItem.GetAssignedTo()
		require.NotNil(t, user)
		assert.Equal(t, "user-123", user.ID)
		assert.Equal(t, "John Doe", user.DisplayName)
		assert.Equal(t, "john.doe@example.com", user.Email)
		assert.Equal(t, "john.doe@example.com", user.UniqueName)
	})

	t.Run("returns user with partial data", func(t *testing.T) {
		workItem := &WorkItem{
			Fields: map[string]interface{}{
				"System.AssignedTo": map[string]interface{}{
					"displayName": "Jane Smith",
					"email":       "jane.smith@example.com",
					// Missing id and uniqueName
				},
			},
		}

		user := workItem.GetAssignedTo()
		require.NotNil(t, user)
		assert.Equal(t, "", user.ID)
		assert.Equal(t, "Jane Smith", user.DisplayName)
		assert.Equal(t, "jane.smith@example.com", user.Email)
		assert.Equal(t, "", user.UniqueName)
	})

	t.Run("returns nil when assigned to is missing", func(t *testing.T) {
		workItem := &WorkItem{
			Fields: map[string]interface{}{},
		}

		user := workItem.GetAssignedTo()
		assert.Nil(t, user)
	})

	t.Run("returns nil when assigned to is not a map", func(t *testing.T) {
		workItem := &WorkItem{
			Fields: map[string]interface{}{
				"System.AssignedTo": "not a map",
			},
		}

		user := workItem.GetAssignedTo()
		assert.Nil(t, user)
	})

	t.Run("handles non-string values in user map", func(t *testing.T) {
		workItem := &WorkItem{
			Fields: map[string]interface{}{
				"System.AssignedTo": map[string]interface{}{
					"id":          123, // Not a string
					"displayName": "John Doe",
					"email":       nil, // Nil value
					"uniqueName":  "john.doe@example.com",
				},
			},
		}

		user := workItem.GetAssignedTo()
		require.NotNil(t, user)
		assert.Equal(t, "", user.ID)
		assert.Equal(t, "John Doe", user.DisplayName)
		assert.Equal(t, "", user.Email)
		assert.Equal(t, "john.doe@example.com", user.UniqueName)
	})
}

func TestWorkItem_GetCreatedBy(t *testing.T) {
	t.Run("returns user when created by is present", func(t *testing.T) {
		workItem := &WorkItem{
			Fields: map[string]interface{}{
				"System.CreatedBy": map[string]interface{}{
					"id":          "creator-456",
					"displayName": "Alice Johnson",
					"email":       "alice.johnson@example.com",
					"uniqueName":  "DOMAIN\\alice.johnson",
				},
			},
		}

		user := workItem.GetCreatedBy()
		require.NotNil(t, user)
		assert.Equal(t, "creator-456", user.ID)
		assert.Equal(t, "Alice Johnson", user.DisplayName)
		assert.Equal(t, "alice.johnson@example.com", user.Email)
		assert.Equal(t, "DOMAIN\\alice.johnson", user.UniqueName)
	})

	t.Run("returns nil when created by is missing", func(t *testing.T) {
		workItem := &WorkItem{
			Fields: map[string]interface{}{},
		}

		user := workItem.GetCreatedBy()
		assert.Nil(t, user)
	})

	t.Run("returns nil when created by is not a map", func(t *testing.T) {
		workItem := &WorkItem{
			Fields: map[string]interface{}{
				"System.CreatedBy": "not a map",
			},
		}

		user := workItem.GetCreatedBy()
		assert.Nil(t, user)
	})
}

func TestWorkItem_GetCreatedDate(t *testing.T) {
	t.Run("returns date when created date is present and valid", func(t *testing.T) {
		workItem := &WorkItem{
			Fields: map[string]interface{}{
				"System.CreatedDate": "2024-01-15T10:30:00Z",
			},
		}

		createdDate := workItem.GetCreatedDate()
		require.NotNil(t, createdDate)

		expectedTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		assert.Equal(t, expectedTime, *createdDate)
	})

	t.Run("returns nil when created date is missing", func(t *testing.T) {
		workItem := &WorkItem{
			Fields: map[string]interface{}{},
		}

		createdDate := workItem.GetCreatedDate()
		assert.Nil(t, createdDate)
	})

	t.Run("returns nil when created date is not a string", func(t *testing.T) {
		workItem := &WorkItem{
			Fields: map[string]interface{}{
				"System.CreatedDate": 1642248600, // Unix timestamp as int
			},
		}

		createdDate := workItem.GetCreatedDate()
		assert.Nil(t, createdDate)
	})

	t.Run("returns nil when created date is invalid format", func(t *testing.T) {
		workItem := &WorkItem{
			Fields: map[string]interface{}{
				"System.CreatedDate": "invalid-date-format",
			},
		}

		createdDate := workItem.GetCreatedDate()
		assert.Nil(t, createdDate)
	})

	t.Run("handles different valid RFC3339 formats", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected time.Time
		}{
			{
				input:    "2024-01-15T10:30:00Z",
				expected: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			},
			{
				input:    "2024-12-25T15:45:30.123Z",
				expected: time.Date(2024, 12, 25, 15, 45, 30, 123000000, time.UTC),
			},
			{
				input:    "2024-06-01T08:00:00-05:00",
				expected: time.Date(2024, 6, 1, 8, 0, 0, 0, time.FixedZone("", -5*60*60)),
			},
		}

		for _, tc := range testCases {
			workItem := &WorkItem{
				Fields: map[string]interface{}{
					"System.CreatedDate": tc.input,
				},
			}

			createdDate := workItem.GetCreatedDate()
			require.NotNil(t, createdDate, "Failed for input: %s", tc.input)
			assert.Equal(t, tc.expected, *createdDate)
		}
	})
}

func TestWorkItem_GetTags(t *testing.T) {
	t.Run("returns tags when present and valid", func(t *testing.T) {
		workItem := &WorkItem{
			Fields: map[string]interface{}{
				"System.Tags": "urgent; customer-reported; bug",
			},
		}

		tags := workItem.GetTags()
		expected := []string{"urgent", "customer-reported", "bug"}
		assert.Equal(t, expected, tags)
	})

	t.Run("returns empty slice when tags are missing", func(t *testing.T) {
		workItem := &WorkItem{
			Fields: map[string]interface{}{},
		}

		tags := workItem.GetTags()
		assert.Empty(t, tags)
	})

	t.Run("returns empty slice when tags is empty string", func(t *testing.T) {
		workItem := &WorkItem{
			Fields: map[string]interface{}{
				"System.Tags": "",
			},
		}

		tags := workItem.GetTags()
		assert.Empty(t, tags)
	})

	t.Run("returns empty slice when tags is not a string", func(t *testing.T) {
		workItem := &WorkItem{
			Fields: map[string]interface{}{
				"System.Tags": []string{"tag1", "tag2"},
			},
		}

		tags := workItem.GetTags()
		assert.Empty(t, tags)
	})

	t.Run("handles single tag", func(t *testing.T) {
		workItem := &WorkItem{
			Fields: map[string]interface{}{
				"System.Tags": "single-tag",
			},
		}

		tags := workItem.GetTags()
		expected := []string{"single-tag"}
		assert.Equal(t, expected, tags)
	})

	t.Run("handles tags with extra whitespace", func(t *testing.T) {
		workItem := &WorkItem{
			Fields: map[string]interface{}{
				"System.Tags": "  urgent  ;  customer-reported  ;  bug  ",
			},
		}

		tags := workItem.GetTags()
		expected := []string{"urgent", "customer-reported", "bug"}
		assert.Equal(t, expected, tags)
	})

	t.Run("handles empty tags in string", func(t *testing.T) {
		workItem := &WorkItem{
			Fields: map[string]interface{}{
				"System.Tags": "urgent;;customer-reported;",
			},
		}

		tags := workItem.GetTags()
		expected := []string{"urgent", "customer-reported"}
		assert.Equal(t, expected, tags)
	})
}

func TestGetStringFromMap(t *testing.T) {
	t.Run("returns string when key exists and value is string", func(t *testing.T) {
		m := map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		}

		result := getStringFromMap(m, "key1")
		assert.Equal(t, "value1", result)
	})

	t.Run("returns empty string when key does not exist", func(t *testing.T) {
		m := map[string]interface{}{
			"key1": "value1",
		}

		result := getStringFromMap(m, "nonexistent")
		assert.Equal(t, "", result)
	})

	t.Run("returns empty string when value is not a string", func(t *testing.T) {
		m := map[string]interface{}{
			"key1": 12345,
			"key2": []string{"array"},
			"key3": map[string]string{"nested": "map"},
		}

		assert.Equal(t, "", getStringFromMap(m, "key1"))
		assert.Equal(t, "", getStringFromMap(m, "key2"))
		assert.Equal(t, "", getStringFromMap(m, "key3"))
	})

	t.Run("returns empty string when value is nil", func(t *testing.T) {
		m := map[string]interface{}{
			"key1": nil,
		}

		result := getStringFromMap(m, "key1")
		assert.Equal(t, "", result)
	})

	t.Run("handles empty map", func(t *testing.T) {
		m := map[string]interface{}{}

		result := getStringFromMap(m, "any-key")
		assert.Equal(t, "", result)
	})
}

func TestParseTagString(t *testing.T) {
	t.Run("parses semicolon-separated tags", func(t *testing.T) {
		input := "tag1;tag2;tag3"
		expected := []string{"tag1", "tag2", "tag3"}

		result := parseTagString(input)
		assert.Equal(t, expected, result)
	})

	t.Run("handles empty string", func(t *testing.T) {
		result := parseTagString("")
		assert.Empty(t, result)
	})

	t.Run("handles single tag", func(t *testing.T) {
		input := "single-tag"
		expected := []string{"single-tag"}

		result := parseTagString(input)
		assert.Equal(t, expected, result)
	})

	t.Run("trims whitespace from tags", func(t *testing.T) {
		input := "  tag1  ;  tag2  ;  tag3  "
		expected := []string{"tag1", "tag2", "tag3"}

		result := parseTagString(input)
		assert.Equal(t, expected, result)
	})

	t.Run("filters out empty tags", func(t *testing.T) {
		input := "tag1;;tag2;;"
		expected := []string{"tag1", "tag2"}

		result := parseTagString(input)
		assert.Equal(t, expected, result)
	})

	t.Run("handles only empty tags", func(t *testing.T) {
		input := ";;;"
		result := parseTagString(input)
		assert.Empty(t, result)
	})

	t.Run("handles complex tag names", func(t *testing.T) {
		input := "high-priority; customer-reported; needs-review"
		expected := []string{"high-priority", "customer-reported", "needs-review"}

		result := parseTagString(input)
		assert.Equal(t, expected, result)
	})
}

// Integration tests for WorkItem struct
func TestWorkItem_Integration(t *testing.T) {
	t.Run("complete work item with all fields", func(t *testing.T) {
		workItem := &WorkItem{
			ID:  12345,
			URL: "https://dev.azure.com/org/project/_workitems/edit/12345",
			Rev: 5,
			Fields: map[string]interface{}{
				"System.Title":        "Sample Bug Report",
				"System.Description":  "This is a detailed description of the bug",
				"System.WorkItemType": "Bug",
				"System.State":        "Active",
				"System.Tags":         "urgent; customer-facing; high-priority",
				"System.CreatedDate":  "2024-01-15T10:30:00Z",
				"System.AssignedTo": map[string]interface{}{
					"id":          "user-123",
					"displayName": "John Doe",
					"email":       "john.doe@example.com",
					"uniqueName":  "john.doe@example.com",
				},
				"System.CreatedBy": map[string]interface{}{
					"id":          "creator-456",
					"displayName": "Jane Smith",
					"email":       "jane.smith@example.com",
					"uniqueName":  "DOMAIN\\jane.smith",
				},
			},
			Comments: []WorkItemComment{
				{
					ID:          1,
					Text:        "This is a comment",
					CreatedDate: time.Date(2024, 1, 16, 9, 0, 0, 0, time.UTC),
					CreatedBy: User{
						ID:          "commenter-789",
						DisplayName: "Bob Wilson",
						Email:       "bob.wilson@example.com",
						UniqueName:  "bob.wilson@example.com",
					},
				},
			},
			Attachments: []WorkItemAttachment{
				{
					ID:          "attachment-1",
					Name:        "screenshot.png",
					URL:         "https://dev.azure.com/attachment/1",
					Size:        1024,
					ContentType: "image/png",
				},
			},
		}

		// Test all getter methods
		assert.Equal(t, "Sample Bug Report", workItem.GetTitle())
		assert.Equal(t, "This is a detailed description of the bug", workItem.GetDescription())
		assert.Equal(t, "Bug", workItem.GetWorkItemType())
		assert.Equal(t, "Active", workItem.GetState())
		assert.Equal(t, []string{"urgent", "customer-facing", "high-priority"}, workItem.GetTags())

		createdDate := workItem.GetCreatedDate()
		require.NotNil(t, createdDate)
		assert.Equal(t, time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC), *createdDate)

		assignedTo := workItem.GetAssignedTo()
		require.NotNil(t, assignedTo)
		assert.Equal(t, "John Doe", assignedTo.DisplayName)
		assert.Equal(t, "john.doe@example.com", assignedTo.Email)

		createdBy := workItem.GetCreatedBy()
		require.NotNil(t, createdBy)
		assert.Equal(t, "Jane Smith", createdBy.DisplayName)
		assert.Equal(t, "jane.smith@example.com", createdBy.Email)

		// Test struct fields
		assert.Equal(t, 12345, workItem.ID)
		assert.Equal(t, "https://dev.azure.com/org/project/_workitems/edit/12345", workItem.URL)
		assert.Equal(t, 5, workItem.Rev)
		assert.Len(t, workItem.Comments, 1)
		assert.Len(t, workItem.Attachments, 1)
	})

	t.Run("minimal work item with only required fields", func(t *testing.T) {
		workItem := &WorkItem{
			ID: 1,
			Fields: map[string]interface{}{
				"System.Title":        "Minimal Work Item",
				"System.WorkItemType": "Task",
				"System.State":        "New",
			},
		}

		assert.Equal(t, "Minimal Work Item", workItem.GetTitle())
		assert.Equal(t, "", workItem.GetDescription())
		assert.Equal(t, "Task", workItem.GetWorkItemType())
		assert.Equal(t, "New", workItem.GetState())
		assert.Empty(t, workItem.GetTags())
		assert.Nil(t, workItem.GetCreatedDate())
		assert.Nil(t, workItem.GetAssignedTo())
		assert.Nil(t, workItem.GetCreatedBy())
	})

	t.Run("work item with empty fields map", func(t *testing.T) {
		workItem := &WorkItem{
			ID:     1,
			Fields: map[string]interface{}{},
		}

		assert.Equal(t, "", workItem.GetTitle())
		assert.Equal(t, "", workItem.GetDescription())
		assert.Equal(t, "", workItem.GetWorkItemType())
		assert.Equal(t, "", workItem.GetState())
		assert.Empty(t, workItem.GetTags())
		assert.Nil(t, workItem.GetCreatedDate())
		assert.Nil(t, workItem.GetAssignedTo())
		assert.Nil(t, workItem.GetCreatedBy())
	})
}
