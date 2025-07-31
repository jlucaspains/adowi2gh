package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	AzureDevOps AzureDevOpsConfig `mapstructure:"azure_devops"`
	GitHub      GitHubConfig      `mapstructure:"github"`
	Migration   MigrationConfig   `mapstructure:"migration"`
}

// AzureDevOpsConfig contains Azure DevOps connection settings
type AzureDevOpsConfig struct {
	OrganizationURL     string        `mapstructure:"organization_url" yaml:"organization_url"`
	PersonalAccessToken string        `mapstructure:"personal_access_token" yaml:"personal_access_token"`
	Project             string        `mapstructure:"project"`
	Query               WorkItemQuery `mapstructure:"query"`
}

// GitHubConfig contains GitHub connection settings
type GitHubConfig struct {
	Token      string `mapstructure:"token"`
	Owner      string `mapstructure:"owner"`
	Repository string `mapstructure:"repository"`
	BaseURL    string `mapstructure:"base_url" yaml:"base_url"` // For GitHub Enterprise
}

// WorkItemQuery defines the query parameters for work items
type WorkItemQuery struct {
	WIQL          string   `mapstructure:"wiql"`
	IDs           []int    `mapstructure:"ids"`
	WorkItemTypes []string `mapstructure:"work_item_types" yaml:"work_item_types"`
	States        []string `mapstructure:"states"`
	AreaPaths     []string `mapstructure:"area_paths" yaml:"area_paths"`
}

// MigrationConfig contains migration-specific settings
type MigrationConfig struct {
	BatchSize            int               `mapstructure:"batch_size" yaml:"batch_size"`
	FieldMapping         FieldMapping      `mapstructure:"field_mapping" yaml:"field_mapping"`
	UserMapping          map[string]string `mapstructure:"user_mapping" yaml:"user_mapping"`
	DryRun               bool              `mapstructure:"dry_run" yaml:"dry_run"`
	IncludeComments      bool              `mapstructure:"include_comments" yaml:"include_comments"`
	ResumeFromCheckpoint bool              `mapstructure:"resume_from_checkpoint" yaml:"resume_from_checkpoint"`
}

// FieldMapping defines how ADO fields map to GitHub fields
type FieldMapping struct {
	StateMapping    map[string]string   `mapstructure:"state_mapping" yaml:"state_mapping"`
	LabelMapping    map[string][]string `mapstructure:"label_mapping" yaml:"label_mapping"`
	TypeMapping     map[string][]string `mapstructure:"type_mapping" yaml:"type_mapping"`
	PriorityMapping map[string][]string `mapstructure:"priority_mapping" yaml:"priority_mapping"`
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigType("yaml")

	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		// Set default config file locations
		viper.SetConfigName("config")
		viper.AddConfigPath(".")
		viper.AddConfigPath("./configs")
		viper.AddConfigPath("$HOME/.ado-gh-migrator")
	}

	// Environment variable overrides
	viper.SetEnvPrefix("ADO_GH")
	viper.AutomaticEnv()

	// Set defaults
	setDefaults()

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	viper.SetDefault("migration.batch_size", 50)
	viper.SetDefault("migration.dry_run", false)
	viper.SetDefault("migration.include_comments", true)
	viper.SetDefault("migration.include_attachments", false)
	viper.SetDefault("migration.create_milestones", true)
	viper.SetDefault("migration.resume_from_checkpoint", false)
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "text")
	viper.SetDefault("github.base_url", "https://api.github.com")
}

// validateConfig validates the loaded configuration
func validateConfig(config *Config) error {
	if config.AzureDevOps.OrganizationURL == "" {
		return fmt.Errorf("azure_devops.organization_url is required")
	}

	if config.AzureDevOps.PersonalAccessToken == "" {
		return fmt.Errorf("azure_devops.personal_access_token is required")
	}

	if config.AzureDevOps.Project == "" {
		return fmt.Errorf("azure_devops.project is required")
	}

	if config.GitHub.Token == "" {
		return fmt.Errorf("github.token is required")
	}

	if config.GitHub.Owner == "" {
		return fmt.Errorf("github.owner is required")
	}

	if config.GitHub.Repository == "" {
		return fmt.Errorf("github.repository is required")
	}

	if config.Migration.BatchSize <= 0 {
		return fmt.Errorf("migration.batch_size must be greater than 0")
	}

	return nil
}

// SaveConfig saves the current configuration to a file
func SaveConfig(config *Config, configPath string) error {
	viper.Set("azure_devops", config.AzureDevOps)
	viper.Set("github", config.GitHub)
	viper.Set("migration", config.Migration)

	if configPath == "" {
		configPath = "./configs/config.yaml"
	}

	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	return viper.WriteConfigAs(configPath)
}
