package models

import (
	"time"
)

// GitHubIssue represents a GitHub issue to be created
type GitHubIssue struct {
	Number     int                    `json:"number,omitempty"`
	Title      string                 `json:"title"`
	Body       string                 `json:"body"`
	State      string                 `json:"state"`
	Labels     []string               `json:"labels"`
	Assignees  []string               `json:"assignees"`
	Milestone  *int                   `json:"milestone,omitempty"`
	CreatedAt  *time.Time             `json:"created_at,omitempty"`
	UpdatedAt  *time.Time             `json:"updated_at,omitempty"`
	ClosedAt   *time.Time             `json:"closed_at,omitempty"`
	Comments   []GitHubComment        `json:"comments,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	SourceWIID int                    `json:"source_wi_id"` // Original ADO work item ID
}

// GitHubComment represents a comment on a GitHub issue
type GitHubComment struct {
	Body string `json:"body"`
}

// MigrationMapping represents the mapping between ADO work item and GitHub issue
type MigrationMapping struct {
	AdoWorkItemID   int       `json:"ado_work_item_id"`
	AdoWorkItemType string    `json:"ado_work_item_type"`
	GitHubIssueID   int       `json:"github_issue_id"`
	GitHubIssueURL  string    `json:"github_issue_url"`
	MigratedAt      time.Time `json:"migrated_at"`
	Status          string    `json:"status"` // "success", "failed", "skipped"
	ErrorMessage    string    `json:"error_message,omitempty"`
}

// MigrationReport represents a summary of the migration process
type MigrationReport struct {
	StartTime       time.Time          `json:"start_time"`
	EndTime         *time.Time         `json:"end_time,omitempty"`
	TotalWorkItems  int                `json:"total_work_items"`
	SuccessfulCount int                `json:"successful_count"`
	FailedCount     int                `json:"failed_count"`
	SkippedCount    int                `json:"skipped_count"`
	Mappings        []MigrationMapping `json:"mappings"`
	Errors          []string           `json:"errors,omitempty"`
}

// MigrationStatus represents the current status of the migration
type MigrationStatus struct {
	IsRunning      bool      `json:"is_running"`
	CurrentItem    int       `json:"current_item"`
	TotalItems     int       `json:"total_items"`
	LastCheckpoint time.Time `json:"last_checkpoint"`
	CanResume      bool      `json:"can_resume"`
}
