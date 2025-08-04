package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	// Reset viper before each test
	viper.Reset()

	t.Run("load valid config file", func(t *testing.T) {
		// Create temporary config file
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "config.yaml")

		configContent := `
azure_devops:
  organization_url: "https://dev.azure.com/myorg"
  personal_access_token: "pat123"
  project: "myproject"
  query:
    wiql: "SELECT * FROM WorkItems"
    ids: [1, 2, 3]
    work_item_types: ["Bug", "Task"]
    states: ["New", "Active"]
    area_paths: ["\\MyProject\\Area1"]

github:
  token: "ghp_token123"
  owner: "myowner"
  repository: "myrepo"
  base_url: "https://api.github.com"

migration:
  batch_size: 25
  dry_run: true
  include_comments: false
  resume_from_checkpoint: true
  field_mapping:
    state_mapping:
      "New": "open"
      "Closed": "closed"
    label_mapping:
      "Bug": ["bug"]
      "Task": ["enhancement"]
    type_mapping:
      "Bug": ["bug"]
    priority_mapping:
      "1": ["priority-high"]
  user_mapping:
    "user1": "githubuser1"
`
		err := os.WriteFile(configFile, []byte(configContent), 0644)
		require.NoError(t, err)

		config, err := LoadConfig(configFile)
		require.NoError(t, err)
		assert.NotNil(t, config)

		// Verify loaded values
		assert.Equal(t, "https://dev.azure.com/myorg", config.AzureDevOps.OrganizationURL)
		assert.Equal(t, "pat123", config.AzureDevOps.PersonalAccessToken)
		assert.Equal(t, "myproject", config.AzureDevOps.Project)
		assert.Equal(t, "ghp_token123", config.GitHub.Token)
		assert.Equal(t, "myowner", config.GitHub.Owner)
		assert.Equal(t, "myrepo", config.GitHub.Repository)
		assert.Equal(t, 25, config.Migration.BatchSize)
		assert.True(t, config.Migration.DryRun)
		assert.False(t, config.Migration.IncludeComments)
	})
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: &Config{
				AzureDevOps: AzureDevOpsConfig{
					OrganizationURL:     "https://dev.azure.com/org",
					PersonalAccessToken: "pat123",
					Project:             "project",
				},
				GitHub: GitHubConfig{
					Token:      "token123",
					Owner:      "owner",
					Repository: "repo",
				},
				Migration: MigrationConfig{
					BatchSize: 50,
				},
			},
			expectError: false,
		},
		{
			name: "missing organization URL",
			config: &Config{
				AzureDevOps: AzureDevOpsConfig{
					PersonalAccessToken: "pat123",
					Project:             "project",
				},
				GitHub: GitHubConfig{
					Token:      "token123",
					Owner:      "owner",
					Repository: "repo",
				},
			},
			expectError: true,
			errorMsg:    "azure_devops.organization_url is required",
		},
		{
			name: "missing PAT",
			config: &Config{
				AzureDevOps: AzureDevOpsConfig{
					OrganizationURL: "https://dev.azure.com/org",
					Project:         "project",
				},
				GitHub: GitHubConfig{
					Token:      "token123",
					Owner:      "owner",
					Repository: "repo",
				},
			},
			expectError: true,
			errorMsg:    "azure_devops.personal_access_token is required",
		},
		{
			name: "missing project",
			config: &Config{
				AzureDevOps: AzureDevOpsConfig{
					OrganizationURL:     "https://dev.azure.com/org",
					PersonalAccessToken: "pat123",
				},
				GitHub: GitHubConfig{
					Token:      "token123",
					Owner:      "owner",
					Repository: "repo",
				},
			},
			expectError: true,
			errorMsg:    "azure_devops.project is required",
		},
		{
			name: "missing GitHub token",
			config: &Config{
				AzureDevOps: AzureDevOpsConfig{
					OrganizationURL:     "https://dev.azure.com/org",
					PersonalAccessToken: "pat123",
					Project:             "project",
				},
				GitHub: GitHubConfig{
					Owner:      "owner",
					Repository: "repo",
				},
			},
			expectError: true,
			errorMsg:    "github.token is required",
		},
		{
			name: "missing GitHub owner",
			config: &Config{
				AzureDevOps: AzureDevOpsConfig{
					OrganizationURL:     "https://dev.azure.com/org",
					PersonalAccessToken: "pat123",
					Project:             "project",
				},
				GitHub: GitHubConfig{
					Token:      "token123",
					Repository: "repo",
				},
			},
			expectError: true,
			errorMsg:    "github.owner is required",
		},
		{
			name: "missing GitHub repository",
			config: &Config{
				AzureDevOps: AzureDevOpsConfig{
					OrganizationURL:     "https://dev.azure.com/org",
					PersonalAccessToken: "pat123",
					Project:             "project",
				},
				GitHub: GitHubConfig{
					Token: "token123",
					Owner: "owner",
				},
			},
			expectError: true,
			errorMsg:    "github.repository is required",
		},
		{
			name: "invalid batch size",
			config: &Config{
				AzureDevOps: AzureDevOpsConfig{
					OrganizationURL:     "https://dev.azure.com/org",
					PersonalAccessToken: "pat123",
					Project:             "project",
				},
				GitHub: GitHubConfig{
					Token:      "token123",
					Owner:      "owner",
					Repository: "repo",
				},
				Migration: MigrationConfig{
					BatchSize: 0,
				},
			},
			expectError: true,
			errorMsg:    "migration.batch_size must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSaveConfig(t *testing.T) {
	t.Run("save config to file", func(t *testing.T) {
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "test_config.yaml")

		config := &Config{
			AzureDevOps: AzureDevOpsConfig{
				OrganizationURL:     "https://dev.azure.com/testorg",
				PersonalAccessToken: "testpat",
				Project:             "testproject",
			},
			GitHub: GitHubConfig{
				Token:      "testtoken",
				Owner:      "testowner",
				Repository: "testrepo",
				BaseURL:    "https://api.github.com",
			},
			Migration: MigrationConfig{
				BatchSize:       100,
				DryRun:          true,
				IncludeComments: false,
			},
		}

		err := SaveConfig(config, configFile)
		require.NoError(t, err)

		// Verify file exists
		_, err = os.Stat(configFile)
		assert.NoError(t, err)

		// Load and verify saved config
		viper.Reset()
		loadedConfig, err := LoadConfig(configFile)
		require.NoError(t, err)

		assert.Equal(t, config.AzureDevOps.OrganizationURL, loadedConfig.AzureDevOps.OrganizationURL)
		assert.Equal(t, config.AzureDevOps.PersonalAccessToken, loadedConfig.AzureDevOps.PersonalAccessToken)
		assert.Equal(t, config.GitHub.Token, loadedConfig.GitHub.Token)
		assert.Equal(t, config.Migration.BatchSize, loadedConfig.Migration.BatchSize)
		assert.Equal(t, config.Migration.DryRun, loadedConfig.Migration.DryRun)
	})

	t.Run("save config with default path", func(t *testing.T) {
		originalWd, _ := os.Getwd()
		tempDir := t.TempDir()
		os.Chdir(tempDir)
		defer os.Chdir(originalWd)

		config := &Config{
			AzureDevOps: AzureDevOpsConfig{
				OrganizationURL:     "https://dev.azure.com/org",
				PersonalAccessToken: "pat",
				Project:             "project",
			},
			GitHub: GitHubConfig{
				Token:      "token",
				Owner:      "owner",
				Repository: "repo",
			},
		}

		err := SaveConfig(config, "")
		require.NoError(t, err)

		// Verify default path was created
		_, err = os.Stat("./configs/config.yaml")
		assert.NoError(t, err)
	})
}

func TestSetDefaults(t *testing.T) {
	viper.Reset()
	setDefaults()

	assert.Equal(t, 50, viper.GetInt("migration.batch_size"))
	assert.False(t, viper.GetBool("migration.dry_run"))
	assert.True(t, viper.GetBool("migration.include_comments"))
	assert.False(t, viper.GetBool("migration.include_attachments"))
	assert.True(t, viper.GetBool("migration.create_milestones"))
	assert.False(t, viper.GetBool("migration.resume_from_checkpoint"))
	assert.Equal(t, "info", viper.GetString("logging.level"))
	assert.Equal(t, "text", viper.GetString("logging.format"))
	assert.Equal(t, "https://api.github.com", viper.GetString("github.base_url"))
}
