package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/jlucaspains/adowi2gh/internal/ado"
	"github.com/jlucaspains/adowi2gh/internal/config"
	"github.com/jlucaspains/adowi2gh/internal/github"
	"github.com/jlucaspains/adowi2gh/internal/migration"
	"github.com/jlucaspains/adowi2gh/internal/models"
)

var (
	// Version information - set by build flags
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"

	// CLI flags
	configFile string
	dryRun     bool
	verbose    bool
	resume     bool
	batchSize  int
	reportFile string
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "adowi2gh",
	Short: "Migrate work items from Azure DevOps to GitHub issues",
	Long: `A command-line tool to migrate work items from Azure DevOps to GitHub issues.
	
This tool connects to both Azure DevOps and GitHub, retrieves work items based on
your configuration, and creates corresponding GitHub issues with proper field mapping,
comments, and metadata preservation.`,
}

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Start the migration process",
	Long: `Start migrating work items from Azure DevOps to GitHub issues.

The migration process will:
1. Connect to Azure DevOps and GitHub
2. Retrieve work items based on your query
3. Map work item fields to GitHub issue fields
4. Create GitHub issues with comments and proper labeling
5. Generate a detailed migration report

Use --dry-run to preview the migration without making changes.`,
	RunE: runMigration,
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management commands",
	Long:  "Commands for managing configuration files and settings.",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new configuration file",
	Long:  "Create a new configuration file with default settings and examples.",
	RunE:  initConfig,
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration and connections",
	Long:  "Validate the configuration file and test connections to Azure DevOps and GitHub.",
	RunE:  validateConfig,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  "Display the version, commit, and build time of the application.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("adowi2gh version %s\n", Version)
		fmt.Printf("Commit: %s\n", Commit)
		fmt.Printf("Built: %s\n", BuildTime)
	},
}

func init() {
	// Root command flags
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Config file path (default: ./configs/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")

	// Migrate command flags
	migrateCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview migration without making changes")
	migrateCmd.Flags().BoolVar(&resume, "resume", false, "Resume from last checkpoint")
	migrateCmd.Flags().IntVar(&batchSize, "batch-size", 0, "Number of items to process in each batch (0 = use config)")
	migrateCmd.Flags().StringVar(&reportFile, "report", "", "Output file for migration report")

	// Add subcommands
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(versionCmd)
	configCmd.AddCommand(configInitCmd)
}

func runMigration(cmd *cobra.Command, args []string) error {
	// Setup logger
	logger := setupLogger()

	// Load configuration
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Override config with CLI flags
	if dryRun {
		cfg.Migration.DryRun = true
	}
	if resume {
		cfg.Migration.ResumeFromCheckpoint = true
	}
	if batchSize > 0 {
		cfg.Migration.BatchSize = batchSize
	}
	logger.Info("Starting Azure DevOps to GitHub migration...")
	logger.Info("Azure DevOps", "url", cfg.AzureDevOps.OrganizationURL+"/"+cfg.AzureDevOps.Project)
	logger.Info("GitHub", "repo", cfg.GitHub.Owner+"/"+cfg.GitHub.Repository)
	if cfg.Migration.DryRun {
		logger.Info("DRY RUN MODE - No changes will be made")
	}

	// Create clients
	adoClient, err := ado.NewClient(&cfg.AzureDevOps, logger)
	if err != nil {
		return fmt.Errorf("failed to create Azure DevOps client: %w", err)
	}

	githubClient, err := github.NewClient(&cfg.GitHub, logger)
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Create mapper
	mapper := migration.NewMapper(&cfg.Migration, logger)

	// Create migration engine
	engine := migration.NewEngine(adoClient, githubClient, mapper, &cfg.Migration, logger)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupts
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Warn("Received interrupt signal, shutting down gracefully...")
		cancel()
	}()

	// Run migration
	report, err := engine.Run(ctx)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// Save report
	reportPath := reportFile
	if reportPath == "" {
		reportPath = fmt.Sprintf("./reports/migration_report_%s.json", report.StartTime.Format("20060102_150405"))
	}
	if err := engine.SaveReport(reportPath); err != nil {
		logger.Warn("Failed to save report", "error", err)
	}

	// Print summary
	printMigrationSummary(report, logger)

	return nil
}

func validateConfig(cmd *cobra.Command, args []string) error {
	logger := setupLogger()

	// Load configuration
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	logger.Info("Configuration file is valid")

	// Test connections
	adoClient, err := ado.NewClient(&cfg.AzureDevOps, logger)
	if err != nil {
		return fmt.Errorf("failed to create Azure DevOps client: %w", err)
	}

	ctx := context.Background()
	if err := adoClient.TestConnection(ctx); err != nil {
		return fmt.Errorf("ado connection failed: %w", err)
	}

	githubClient, err := github.NewClient(&cfg.GitHub, logger)
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}
	if err := githubClient.TestConnection(ctx); err != nil {
		return fmt.Errorf("GitHub connection failed: %w", err)
	}

	logger.Info("✓ All connections successful")
	logger.Info("✓ Configuration is valid and ready for migration")

	return nil
}

func initConfig(cmd *cobra.Command, args []string) error {
	logger := setupLogger()

	configPath := configFile
	if configPath == "" {
		configPath = "./configs/config.yaml"
	}
	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		logger.Warn("Configuration file already exists", "path", configPath)
		fmt.Print("Do you want to overwrite it? (y/N): ")
		var response string
		_, err := fmt.Scanln(&response)

		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		if response != "y" && response != "Y" {
			logger.Info("Configuration initialization cancelled")
			return nil
		}
	}

	// Create default configuration
	defaultConfig := createDefaultConfig()

	if err := config.SaveConfig(defaultConfig, configPath); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}
	logger.Info("✓ Configuration file created", "path", configPath)
	logger.Info("Please edit the configuration file with your Azure DevOps and GitHub settings")

	return nil
}

func createDefaultConfig() *config.Config {
	return &config.Config{
		AzureDevOps: config.AzureDevOpsConfig{
			OrganizationURL:     "https://dev.azure.com/your-organization",
			PersonalAccessToken: "your-ado-pat-token",
			Project:             "your-project-name",
			Query: config.WorkItemQuery{
				WIQL:          "",
				WorkItemTypes: []string{"Bug", "User Story", "Task"},
				States:        []string{"New", "Active", "Resolved"},
			},
		},
		GitHub: config.GitHubConfig{
			Token:      "your-github-token",
			Owner:      "your-github-username-or-org",
			Repository: "your-repository-name",
			BaseURL:    "https://api.github.com",
		},
		Migration: config.MigrationConfig{
			BatchSize: 50,
			FieldMapping: config.FieldMapping{
				StateMapping: map[string]string{
					"New":      "open",
					"Active":   "open",
					"Resolved": "open",
					"Closed":   "closed",
					"Done":     "closed",
				},
				TypeMapping: map[string][]string{
					"Bug":        {"bug"},
					"User Story": {"enhancement"},
					"Task":       {"task"},
					"Epic":       {"epic"},
				},
				PriorityMapping: map[string][]string{
					"1": {"priority:critical"},
					"2": {"priority:high"},
					"3": {"priority:medium"},
					"4": {"priority:low"},
				},
				IncludeSeverityLabel: true,
				IncludeAreaPathLabel: true,
				TimeZone:             "UTC",
			},
			UserMapping:          map[string]string{},
			DryRun:               false,
			IncludeComments:      true,
			ResumeFromCheckpoint: false,
		},
	}
}

func setupLogger() *slog.Logger {
	opts := &slog.HandlerOptions{}

	if verbose {
		opts.Level = slog.LevelDebug
	} else {
		opts.Level = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	logger := slog.New(handler)

	return logger
}

func printMigrationSummary(report *models.MigrationReport, logger *slog.Logger) {
	logger.Info("=== Migration Summary ===")
	logger.Info("Migration results",
		"total", report.TotalWorkItems,
		"successful", report.SuccessfulCount,
		"failed", report.FailedCount,
		"skipped", report.SkippedCount)

	if report.EndTime != nil {
		duration := report.EndTime.Sub(report.StartTime)
		logger.Info("Migration duration", "duration", duration)
	}

	if len(report.Errors) > 0 {
		logger.Warn("Errors encountered:")
		for _, err := range report.Errors {
			logger.Warn("Error", "message", err)
		}
	}

	if report.SuccessfulCount > 0 {
		logger.Info("✓ Migration completed successfully!")
	}
}
