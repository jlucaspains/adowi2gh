package models

import (
	"strings"
	"time"
)

// WorkItem represents an Azure DevOps work item
type WorkItem struct {
	ID          int                    `json:"id"`
	URL         string                 `json:"url"`
	Rev         int                    `json:"rev"`
	Fields      map[string]interface{} `json:"fields"`
	Relations   []WorkItemRelation     `json:"relations,omitempty"`
	Comments    []WorkItemComment      `json:"comments,omitempty"`
	Attachments []WorkItemAttachment   `json:"attachments,omitempty"`
}

// WorkItemRelation represents a relation between work items
type WorkItemRelation struct {
	Rel        string                 `json:"rel"`
	URL        string                 `json:"url"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// WorkItemComment represents a comment on a work item
type WorkItemComment struct {
	ID           int        `json:"id"`
	Text         string     `json:"text"`
	CreatedBy    User       `json:"createdBy"`
	CreatedDate  time.Time  `json:"createdDate"`
	ModifiedBy   User       `json:"modifiedBy,omitempty"`
	ModifiedDate *time.Time `json:"modifiedDate,omitempty"`
}

// WorkItemAttachment represents an attachment on a work item
type WorkItemAttachment struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Size        int64  `json:"size"`
	ContentType string `json:"contentType"`
}

// User represents a user in the system
type User struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	UniqueName  string `json:"uniqueName"`
}

// GetTitle returns the title of the work item
func (wi *WorkItem) GetTitle() string {
	if title, ok := wi.Fields["System.Title"].(string); ok {
		return title
	}
	return ""
}

// GetDescription returns the description of the work item
func (wi *WorkItem) GetDescription() string {
	if desc, ok := wi.Fields["System.Description"].(string); ok {
		return desc
	}
	return ""
}

// GetWorkItemType returns the type of the work item
func (wi *WorkItem) GetWorkItemType() string {
	if wiType, ok := wi.Fields["System.WorkItemType"].(string); ok {
		return wiType
	}
	return ""
}

// GetState returns the state of the work item
func (wi *WorkItem) GetState() string {
	if state, ok := wi.Fields["System.State"].(string); ok {
		return state
	}
	return ""
}

// GetAssignedTo returns the assigned user
func (wi *WorkItem) GetAssignedTo() *User {
	if assignedTo, ok := wi.Fields["System.AssignedTo"].(map[string]interface{}); ok {
		return &User{
			ID:          getStringFromMap(assignedTo, "id"),
			DisplayName: getStringFromMap(assignedTo, "displayName"),
			Email:       getStringFromMap(assignedTo, "email"),
			UniqueName:  getStringFromMap(assignedTo, "uniqueName"),
		}
	}
	return nil
}

// GetCreatedBy returns the user who created the work item
func (wi *WorkItem) GetCreatedBy() *User {
	if createdBy, ok := wi.Fields["System.CreatedBy"].(map[string]interface{}); ok {
		return &User{
			ID:          getStringFromMap(createdBy, "id"),
			DisplayName: getStringFromMap(createdBy, "displayName"),
			Email:       getStringFromMap(createdBy, "email"),
			UniqueName:  getStringFromMap(createdBy, "uniqueName"),
		}
	}
	return nil
}

// GetCreatedDate returns the creation date
func (wi *WorkItem) GetCreatedDate() *time.Time {
	if createdDate, ok := wi.Fields["System.CreatedDate"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdDate); err == nil {
			return &t
		}
	}
	return nil
}

// GetTags returns the tags as a slice
func (wi *WorkItem) GetTags() []string {
	if tags, ok := wi.Fields["System.Tags"].(string); ok && tags != "" {
		// Tags are typically semicolon-separated in ADO
		return parseTagString(tags)
	}
	return []string{}
}

// Helper function to safely get string from map
func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// Helper function to parse tag string
func parseTagString(tags string) []string {
	if tags == "" {
		return []string{}
	}

	// Split by semicolon and clean up each tag
	var result []string
	parts := strings.Split(tags, ";")
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}
