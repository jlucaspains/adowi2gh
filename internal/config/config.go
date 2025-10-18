package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v4"
)

type Config struct {
	AzureDevOps AzureDevOpsConfig `yaml:"azure_devops"`
	GitHub      GitHubConfig      `yaml:"github"`
	Migration   MigrationConfig   `yaml:"migration"`
}

type AzureDevOpsConfig struct {
	OrganizationURL     string        `yaml:"organization_url"`
	PersonalAccessToken string        `yaml:"personal_access_token"`
	Project             string        `yaml:"project"`
	Query               WorkItemQuery `yaml:"query"`
}

type GitHubConfig struct {
	Token              string `yaml:"token"`
	AppCertificatePath string `yaml:"app_certificate_path"`
	AppId              int64  `yaml:"app_id"`
	InstallationId     int64  `yaml:"installation_id"`
	Owner              string `yaml:"owner"`
	Repository         string `yaml:"repository"`
	BaseURL            string `yaml:"base_url"` // For GitHub Enterprise
}

type WorkItemQuery struct {
	WIQL          string   `yaml:"wiql"`
	IDs           []int    `yaml:"ids"`
	WorkItemTypes []string `yaml:"work_item_types"`
	States        []string `yaml:"states"`
	AreaPaths     []string `yaml:"area_paths"`
}

type MigrationConfig struct {
	BatchSize            int               `yaml:"batch_size"`
	FieldMapping         FieldMapping      `yaml:"field_mapping"`
	UserMapping          map[string]string `yaml:"user_mapping"`
	DryRun               bool              `yaml:"dry_run"`
	IncludeComments      bool              `yaml:"include_comments"`
	ResumeFromCheckpoint bool              `yaml:"resume_from_checkpoint"`
}

type FieldMapping struct {
	StateMapping         map[string]string   `yaml:"state_mapping"`
	LabelMapping         map[string][]string `yaml:"label_mapping"`
	TypeMapping          map[string][]string `yaml:"type_mapping"`
	PriorityMapping      map[string][]string `yaml:"priority_mapping"`
	TimeZone             string              `yaml:"time_zone"`
	IncludeSeverityLabel bool                `yaml:"include_severity_label"`
	IncludeAreaPathLabel bool                `yaml:"include_area_path_label"`
}

func LoadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = "./configs/config.yaml"
	}

	slog.Info("Loading configuration", "file", configPath)

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	config := &Config{}
	setDefaults(config)

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

func setDefaults(config *Config) {
	config.Migration.BatchSize = 50
	config.Migration.DryRun = false
	config.Migration.IncludeComments = true
	config.Migration.ResumeFromCheckpoint = false
	config.GitHub.BaseURL = "https://api.github.com"
}

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

func SaveConfig(config *Config, configPath string) error {
	if configPath == "" {
		configPath = "./configs/config.yaml"
	}

	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("error marshaling config: %w", err)
	}

	return os.WriteFile(configPath, data, 0644)
}
