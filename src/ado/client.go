package ado

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/workitemtracking"

	"github.com/jlucaspains/adowi2gh/config"
	"github.com/jlucaspains/adowi2gh/models"
)

type Client struct {
	connection *azuredevops.Connection
	witClient  workitemtracking.Client
	config     *config.AzureDevOpsConfig
	logger     *slog.Logger
}

func NewClient(cfg *config.AzureDevOpsConfig, logger *slog.Logger) (*Client, error) {
	if cfg.OrganizationURL == "" {
		return nil, fmt.Errorf("organization URL is required")
	}

	if cfg.PersonalAccessToken == "" {
		return nil, fmt.Errorf("personal access token is required")
	}

	// Create a connection to Azure DevOps
	connection := azuredevops.NewPatConnection(cfg.OrganizationURL, cfg.PersonalAccessToken)

	// Create work item tracking client
	witClient, err := workitemtracking.NewClient(context.Background(), connection)
	if err != nil {
		return nil, fmt.Errorf("failed to create work item tracking client: %w", err)
	}

	return &Client{
		connection: connection,
		witClient:  witClient,
		config:     cfg,
		logger:     logger,
	}, nil
}

func (c *Client) TestConnection(ctx context.Context) error {
	c.logger.Info("Testing Azure DevOps connection...")

	// Try to execute a simple WIQL query to test the connection
	testQuery := fmt.Sprintf("SELECT [System.Id] FROM WorkItems WHERE [System.TeamProject] = '%s'", c.config.Project)

	queryArgs := workitemtracking.QueryByWiqlArgs{
		Project: &c.config.Project,
		Wiql: &workitemtracking.Wiql{
			Query: &testQuery,
		},
	}

	_, err := c.witClient.QueryByWiql(ctx, queryArgs)
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	c.logger.Info("Azure DevOps connection successful")
	return nil
}

func (c *Client) GetWorkItems(ctx context.Context) ([]*models.WorkItem, error) {
	c.logger.Info("Retrieving work items from Azure DevOps...")

	var workItemIds []int
	var err error

	// If specific IDs are provided, use them
	if len(c.config.Query.IDs) > 0 {
		workItemIds = c.config.Query.IDs
	} else if c.config.Query.WIQL != "" {
		// Execute WIQL query
		workItemIds, err = c.executeWIQL(ctx, c.config.Query.WIQL)
		if err != nil {
			return nil, fmt.Errorf("failed to execute WIQL query: %w", err)
		}
	} else {
		// Build a default query based on filters
		wiql := c.buildDefaultQuery()
		workItemIds, err = c.executeWIQL(ctx, wiql)
		if err != nil {
			return nil, fmt.Errorf("failed to execute default query: %w", err)
		}
	}

	if len(workItemIds) == 0 {
		c.logger.Warn("No work items found matching the query")
		return []*models.WorkItem{}, nil
	}

	c.logger.Info("Found work items, retrieving details", "count", len(workItemIds))

	// Get work item details
	return c.getWorkItemDetails(ctx, workItemIds)
}

func (c *Client) executeWIQL(ctx context.Context, wiql string) ([]int, error) {
	queryArgs := workitemtracking.QueryByWiqlArgs{
		Project: &c.config.Project,
		Wiql: &workitemtracking.Wiql{
			Query: &wiql,
		},
	}

	result, err := c.witClient.QueryByWiql(ctx, queryArgs)
	if err != nil {
		return nil, fmt.Errorf("WIQL query execution failed: %w", err)
	}

	var workItemIds []int
	if result.WorkItems != nil {
		for _, wi := range *result.WorkItems {
			if wi.Id != nil {
				workItemIds = append(workItemIds, *wi.Id)
			}
		}
	}

	return workItemIds, nil
}

func (c *Client) buildDefaultQuery() string {
	query := fmt.Sprintf("SELECT [System.Id] FROM WorkItems WHERE [System.TeamProject] = '%s'", c.config.Project)

	if len(c.config.Query.WorkItemTypes) > 0 {
		query += " AND [System.WorkItemType] IN ("
		for i, wiType := range c.config.Query.WorkItemTypes {
			if i > 0 {
				query += ", "
			}
			query += fmt.Sprintf("'%s'", wiType)
		}
		query += ")"
	}

	if len(c.config.Query.States) > 0 {
		query += " AND [System.State] IN ("
		for i, state := range c.config.Query.States {
			if i > 0 {
				query += ", "
			}
			query += fmt.Sprintf("'%s'", state)
		}
		query += ")"
	}

	if len(c.config.Query.AreaPaths) > 0 {
		query += " AND [System.AreaPath] UNDER ("
		for i, areaPath := range c.config.Query.AreaPaths {
			if i > 0 {
				query += " OR [System.AreaPath] UNDER "
			}
			query += fmt.Sprintf("'%s'", areaPath)
		}
		query += ")"
	}

	return query
}

func (c *Client) getWorkItemDetails(ctx context.Context, workItemIds []int) ([]*models.WorkItem, error) {
	var workItems []*models.WorkItem

	// Get work items in batches to avoid API limits
	batchSize := 100 // ADO API limit
	for i := 0; i < len(workItemIds); i += batchSize {
		end := i + batchSize
		if end > len(workItemIds) {
			end = len(workItemIds)
		}

		batch := workItemIds[i:end]
		c.logger.Debug("Retrieving work item batch", "start", i+1, "end", end)

		batchItems, err := c.getWorkItemBatch(ctx, batch)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve work item batch: %w", err)
		}

		workItems = append(workItems, batchItems...)
	}

	return workItems, nil
}

func (c *Client) getWorkItemBatch(ctx context.Context, ids []int) ([]*models.WorkItem, error) {
	expand := workitemtracking.WorkItemExpandValues.All

	getWorkItemsArgs := workitemtracking.GetWorkItemsArgs{
		Project: &c.config.Project,
		Ids:     &ids,
		Expand:  &expand,
	}

	response, err := c.witClient.GetWorkItems(ctx, getWorkItemsArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to get work items: %w", err)
	}

	var workItems []*models.WorkItem
	if response != nil {
		for _, adoWorkItem := range *response {
			workItem := c.convertToWorkItem(adoWorkItem)
			workItems = append(workItems, workItem)
		}
	}

	return workItems, nil
}

func (c *Client) convertToWorkItem(adoWorkItem workitemtracking.WorkItem) *models.WorkItem {
	workItem := &models.WorkItem{
		Fields:      make(map[string]interface{}),
		Relations:   []models.WorkItemRelation{},
		Comments:    []models.WorkItemComment{},
		Attachments: []models.WorkItemAttachment{},
	}

	if adoWorkItem.Id != nil {
		workItem.ID = *adoWorkItem.Id
	}

	if adoWorkItem.Url != nil {
		workItem.URL = *adoWorkItem.Url
	}

	if adoWorkItem.Rev != nil {
		workItem.Rev = *adoWorkItem.Rev
	}

	if adoWorkItem.Fields != nil {
		for key, value := range *adoWorkItem.Fields {
			workItem.Fields[key] = value
		}
	}

	if adoWorkItem.Relations != nil {
		for _, relation := range *adoWorkItem.Relations {
			workItem.Relations = append(workItem.Relations, models.WorkItemRelation{
				Rel: getStringPtr(relation.Rel),
				URL: getStringPtr(relation.Url),
			})
		}
	}

	return workItem
}

func (c *Client) GetWorkItemComments(ctx context.Context, workItemID int) ([]models.WorkItemComment, error) {
	getCommentsArgs := workitemtracking.GetCommentsArgs{
		Project:    &c.config.Project,
		WorkItemId: &workItemID,
	}

	response, err := c.witClient.GetComments(ctx, getCommentsArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to get comments for work item %d: %w", workItemID, err)
	}

	var comments []models.WorkItemComment
	if response.Comments != nil {
		for _, comment := range *response.Comments {
			comments = append(comments, models.WorkItemComment{
				ID:   getIntPtr(comment.Id),
				Text: getStringPtr(comment.Text),
				CreatedBy: models.User{
					DisplayName: *comment.CreatedBy.DisplayName,
				},
				CreatedDate: comment.CreatedDate.Time,
			})
		}
	}

	return comments, nil
}

func getStringPtr(ptr *string) string {
	if ptr != nil {
		return *ptr
	}
	return ""
}

func getIntPtr(ptr *int) int {
	if ptr != nil {
		return *ptr
	}
	return 0
}
