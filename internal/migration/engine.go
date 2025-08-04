package migration

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/jlucaspains/adowi2gh/internal/ado"
	"github.com/jlucaspains/adowi2gh/internal/config"
	"github.com/jlucaspains/adowi2gh/internal/github"
	"github.com/jlucaspains/adowi2gh/internal/models"
)

type Engine struct {
	adoClient    *ado.Client
	githubClient *github.Client
	mapper       *Mapper
	config       *config.MigrationConfig
	logger       *slog.Logger
	report       *models.MigrationReport
	checkpoint   *MigrationCheckpoint
}

type MigrationCheckpoint struct {
	LastProcessedID int                       `json:"last_processed_id"`
	ProcessedItems  []int                     `json:"processed_items"`
	FailedItems     []int                     `json:"failed_items"`
	Mappings        []models.MigrationMapping `json:"mappings"`
	StartTime       time.Time                 `json:"start_time"`
	LastUpdate      time.Time                 `json:"last_update"`
}

func NewEngine(
	adoClient *ado.Client,
	githubClient *github.Client,
	mapper *Mapper,
	config *config.MigrationConfig,
	logger *slog.Logger,
) *Engine {
	return &Engine{
		adoClient:    adoClient,
		githubClient: githubClient,
		mapper:       mapper,
		config:       config,
		logger:       logger,
		report: &models.MigrationReport{
			StartTime: time.Now(),
			Mappings:  []models.MigrationMapping{},
			Errors:    []string{},
		},
		checkpoint: &MigrationCheckpoint{
			ProcessedItems: []int{},
			FailedItems:    []int{},
			Mappings:       []models.MigrationMapping{},
			StartTime:      time.Now(),
		},
	}
}

func (e *Engine) Run(ctx context.Context) (*models.MigrationReport, error) {
	e.logger.Info("Starting migration process...")
	// Load checkpoint if resuming
	if e.config.ResumeFromCheckpoint {
		if err := e.loadCheckpoint(); err != nil {
			e.logger.Warn("Failed to load checkpoint", "error", err)
		}
	}

	if err := e.testConnections(ctx); err != nil {
		return nil, fmt.Errorf("connection test failed: %w", err)
	}

	workItems, err := e.adoClient.GetWorkItems(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve work items: %w", err)
	}
	e.report.TotalWorkItems = len(workItems)
	e.logger.Info("Found work items to migrate", "count", len(workItems))

	if e.config.DryRun {
		e.logger.Info("DRY RUN MODE - No changes will be made")
		return e.performDryRun(ctx, workItems)
	}

	return e.performMigration(ctx, workItems)
}

func (e *Engine) testConnections(ctx context.Context) error {
	e.logger.Info("Testing service connections...")

	if err := e.adoClient.TestConnection(ctx); err != nil {
		return fmt.Errorf("azure devops connection failed: %w", err)
	}

	if err := e.githubClient.TestConnection(ctx); err != nil {
		return fmt.Errorf("GitHub connection failed: %w", err)
	}

	e.logger.Info("All connections successful")
	return nil
}

func (e *Engine) performDryRun(ctx context.Context, workItems []*models.WorkItem) (*models.MigrationReport, error) {
	e.logger.Info("Performing dry run...")
	for i, workItem := range workItems {
		e.logger.Info("Processing work item",
			"current", i+1,
			"total", len(workItems),
			"id", workItem.ID,
			"title", workItem.GetTitle())

		issue, err := e.mapper.MapWorkItemToIssue(workItem)
		if err != nil {
			e.logger.Error("Failed to map work item", "id", workItem.ID, "error", err)
			e.report.FailedCount++
			continue
		}

		if err := e.githubClient.ValidateLabels(ctx, issue.Labels); err != nil {
			e.logger.Error("Label validation failed for work item", "id", workItem.ID, "error", err)
			e.report.FailedCount++
			continue
		}

		e.logger.Info("Work item would be migrated", "id", workItem.ID, "title", issue.Title)
		e.logger.Debug("Migration details",
			"labels", issue.Labels,
			"assignees", issue.Assignees,
			"state", issue.State)

		e.report.SuccessfulCount++
	}
	endTime := time.Now()
	e.report.EndTime = &endTime
	e.logger.Info("Dry run completed",
		"successful", e.report.SuccessfulCount,
		"failed", e.report.FailedCount)

	return e.report, nil
}

func (e *Engine) performMigration(ctx context.Context, workItems []*models.WorkItem) (*models.MigrationReport, error) {
	e.logger.Info("Starting actual migration...")

	batchSize := e.config.BatchSize
	if batchSize <= 0 {
		batchSize = 10
	}

	for i := 0; i < len(workItems); i += batchSize {
		end := i + batchSize
		if end > len(workItems) {
			end = len(workItems)
		}
		batch := workItems[i:end]
		e.logger.Info("Processing batch", "start", i+1, "end", end, "total", len(workItems))

		if err := e.processBatch(ctx, batch); err != nil {
			e.logger.Error("Batch processing failed", "error", err)
			// Continue with next batch
		}

		// Save checkpoint after each batch
		if err := e.saveCheckpoint(); err != nil {
			e.logger.Warn("Failed to save checkpoint", "error", err)
		}

		// Rate limiting
		if len(batch) > 0 {
			e.logger.Debug("Applying rate limiting...")
			time.Sleep(time.Second * 2)
		}
	}
	endTime := time.Now()
	e.report.EndTime = &endTime

	e.logger.Info("Migration completed",
		"successful", e.report.SuccessfulCount,
		"failed", e.report.FailedCount,
		"skipped", e.report.SkippedCount)

	return e.report, nil
}

func (e *Engine) processBatch(ctx context.Context, workItems []*models.WorkItem) error {
	for _, workItem := range workItems {
		if err := e.processWorkItem(ctx, workItem); err != nil {
			e.logger.Error("Failed to process work item", "id", workItem.ID, "error", err)
			e.recordFailure(workItem.ID, err.Error())
		}
	}
	return nil
}

func (e *Engine) processWorkItem(ctx context.Context, workItem *models.WorkItem) error { // Check if already processed (for resume functionality)
	if e.isAlreadyProcessed(workItem.ID) {
		e.logger.Debug("Work item already processed, skipping", "id", workItem.ID)
		e.report.SkippedCount++
		return nil
	}

	e.logger.Info("Processing work item", "id", workItem.ID, "title", workItem.GetTitle())

	// Check if issue already exists
	existingIssues, err := e.githubClient.SearchIssues(ctx, workItem.ID)
	if err != nil {
		return fmt.Errorf("failed to search for existing issues: %w", err)
	}
	if len(existingIssues) > 0 {
		e.logger.Info("Issue already exists for work item, skipping", "id", workItem.ID)
		e.report.SkippedCount++
		e.recordMapping(workItem.ID, existingIssues[0].GetNumber(), "skipped", "Issue already exists")
		return nil
	}

	issue, err := e.mapper.MapWorkItemToIssue(workItem)
	if err != nil {
		return fmt.Errorf("failed to map work item: %w", err)
	}

	createdIssue, err := e.githubClient.CreateIssue(ctx, issue)
	if err != nil {
		return fmt.Errorf("failed to create GitHub issue: %w", err)
	}
	if e.config.IncludeComments {
		if err := e.processComments(ctx, workItem, createdIssue.Number); err != nil {
			e.logger.Warn("Failed to migrate comments for work item", "id", workItem.ID, "error", err)
		}
	}

	if issue.State == "closed" {
		if err := e.githubClient.UpdateIssueState(ctx, createdIssue.Number, "closed"); err != nil {
			e.logger.Warn("Failed to close issue", "issue", createdIssue.Number, "error", err)
		}
	}

	e.recordSuccess(workItem.ID, createdIssue.Number)
	e.checkpoint.LastProcessedID = workItem.ID
	e.checkpoint.LastUpdate = time.Now()

	return nil
}

func (e *Engine) processComments(ctx context.Context, workItem *models.WorkItem, issueNumber int) error {
	comments, err := e.adoClient.GetWorkItemComments(ctx, workItem.ID)
	if err != nil {
		return fmt.Errorf("failed to get work item comments: %w", err)
	}

	if len(comments) == 0 {
		return nil
	}

	e.logger.Debug("Migrating comments for work item", "count", len(comments), "id", workItem.ID)

	githubComments := e.mapper.MapComments(comments)
	for _, comment := range githubComments {
		if err := e.githubClient.CreateIssueComment(ctx, issueNumber, &comment); err != nil {
			return fmt.Errorf("failed to create comment: %w", err)
		}
	}

	return nil
}

func (e *Engine) isAlreadyProcessed(workItemID int) bool {
	for _, id := range e.checkpoint.ProcessedItems {
		if id == workItemID {
			return true
		}
	}
	return false
}

func (e *Engine) recordSuccess(workItemID, issueNumber int) {
	e.report.SuccessfulCount++
	e.checkpoint.ProcessedItems = append(e.checkpoint.ProcessedItems, workItemID)
	e.recordMapping(workItemID, issueNumber, "success", "")
}

func (e *Engine) recordFailure(workItemID int, errorMsg string) {
	e.report.FailedCount++
	e.checkpoint.FailedItems = append(e.checkpoint.FailedItems, workItemID)
	e.report.Errors = append(e.report.Errors, fmt.Sprintf("Work Item %d: %s", workItemID, errorMsg))
	e.recordMapping(workItemID, 0, "failed", errorMsg)
}

func (e *Engine) recordMapping(workItemID, issueNumber int, status, errorMsg string) {
	mapping := models.MigrationMapping{
		AdoWorkItemID: workItemID,
		GitHubIssueID: issueNumber,
		MigratedAt:    time.Now(),
		Status:        status,
		ErrorMessage:  errorMsg,
	}

	e.report.Mappings = append(e.report.Mappings, mapping)
	e.checkpoint.Mappings = append(e.checkpoint.Mappings, mapping)
}

func (e *Engine) saveCheckpoint() error {
	checkpointPath := "./migration_checkpoint.json"

	data, err := json.MarshalIndent(e.checkpoint, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	if err := os.WriteFile(checkpointPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write checkpoint file: %w", err)
	}

	return nil
}

func (e *Engine) loadCheckpoint() error {
	checkpointPath := "./migration_checkpoint.json"

	data, err := os.ReadFile(checkpointPath)
	if err != nil {
		return fmt.Errorf("failed to read checkpoint file: %w", err)
	}

	if err := json.Unmarshal(data, e.checkpoint); err != nil {
		return fmt.Errorf("failed to unmarshal checkpoint: %w", err)
	}
	e.logger.Info("Loaded checkpoint",
		"processed_items", len(e.checkpoint.ProcessedItems),
		"last_id", e.checkpoint.LastProcessedID)

	return nil
}

func (e *Engine) SaveReport(filePath string) error {
	if filePath == "" {
		filePath = fmt.Sprintf("migration_report_%s.json", time.Now().Format("20060102_150405"))
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create report directory: %w", err)
	}

	data, err := json.MarshalIndent(e.report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write report file: %w", err)
	}
	e.logger.Info("Migration report saved", "path", filePath)
	return nil
}
