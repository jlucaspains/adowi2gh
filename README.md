# Azure DevOps to GitHub Work Items Migrator

A powerful command-line tool to migrate work items from Azure DevOps to GitHub issues with full field mapping, comments, and metadata preservation.

## Features

- **Full Migration Support**: Migrate work items with titles, descriptions, comments, and metadata
- **Field Mapping**: Configurable mapping of ADO fields to GitHub issue fields
- **State Management**: Map ADO work item states to GitHub issue states (open/closed)
- **Label Generation**: Automatic label creation based on work item type, priority, and tags
- **User Mapping**: Map ADO users to GitHub usernames for proper assignment
- **Batch Processing**: Process work items in configurable batches with rate limiting
- **Resume Capability**: Resume interrupted migrations from checkpoints
- **Dry Run Mode**: Preview migrations without making changes
- **Comprehensive Reporting**: Detailed migration reports with success/failure tracking

## Prerequisites

- Go 1.19 or later
- Azure DevOps Personal Access Token with Work Items (read) permission
- GitHub Personal Access Token with repository permissions
- Access to both Azure DevOps organization and target GitHub repository

## Installation

### Build from Source

```bash
git clone <repository-url>
cd ado-gh-wi-migrator
go build -o ado-gh-wi-migrator ./cmd/migrate
```

### Using Go Install

```bash
go install ./cmd/migrate
```

## Quick Start

1. **Initialize Configuration**
   ```bash
   ./ado-gh-wi-migrator config init
   ```

2. **Edit Configuration**
   Update `configs/config.yaml` with your Azure DevOps and GitHub settings:
   ```yaml
   azure_devops:
     organization_url: "https://dev.azure.com/your-organization"
     personal_access_token: "your-ado-pat"
     project: "your-project"
   
   github:
     token: "your-github-token"
     owner: "your-username"
     repository: "your-repo"
   ```

3. **Validate Configuration**
   ```bash
   ./ado-gh-wi-migrator validate
   ```

4. **Run Migration**
   ```bash
   # Dry run first
   ./ado-gh-wi-migrator migrate --dry-run
   
   # Actual migration
   ./ado-gh-wi-migrator migrate
   ```

## Configuration

### Azure DevOps Setup

1. Generate a Personal Access Token:
   - Go to Azure DevOps → User Settings → Personal Access Tokens
   - Create new token with "Work Items (read)" scope
   - Copy the token for configuration

2. Configure work item query:
   - Use WIQL for complex queries
   - Or specify work item types, states, and area paths
   - Or provide specific work item IDs

### GitHub Setup

1. Generate a Personal Access Token:
   - Go to GitHub → Settings → Developer settings → Personal access tokens
   - Create token with "repo" scope
   - Copy the token for configuration

2. Ensure target repository exists and you have write access

### Field Mapping

Configure how ADO fields map to GitHub:

```yaml
field_mapping:
  state_mapping:
    "New": "open"
    "Done": "closed"
  
  type_mapping:
    "Bug": ["bug"]
    "User Story": ["enhancement"]
  
  priority_mapping:
    "1": ["priority:critical"]
    "2": ["priority:high"]
```

### User Mapping

Map ADO users to GitHub usernames:

```yaml
user_mapping:
  "john.doe@company.com": "johndoe"
  "jane.smith@company.com": "janesmith"
```

## Usage

### Commands

```bash
# Show help
./ado-gh-wi-migrator --help

# Initialize configuration
./ado-gh-wi-migrator config init

# Validate configuration and test connections
./ado-gh-wi-migrator validate

# Run migration
./ado-gh-wi-migrator migrate [flags]
```

### Migration Flags

```bash
--dry-run          # Preview migration without making changes
--resume           # Resume from last checkpoint
--batch-size N     # Override batch size from config
--report FILE      # Specify output file for migration report
--config FILE      # Use specific configuration file
--verbose          # Enable verbose logging
```

### Examples

```bash
# Dry run to preview changes
./ado-gh-wi-migrator migrate --dry-run

# Migrate with custom batch size
./ado-gh-wi-migrator migrate --batch-size 25

# Resume interrupted migration
./ado-gh-wi-migrator migrate --resume

# Use custom config file
./ado-gh-wi-migrator migrate --config /path/to/config.yaml

# Verbose logging with custom report location
./ado-gh-wi-migrator migrate --verbose --report ./reports/migration.json
```

## Migration Process

1. **Connection Testing**: Validates connectivity to both services
2. **Work Item Retrieval**: Queries ADO based on configuration
3. **Field Mapping**: Converts ADO fields to GitHub format
4. **Issue Creation**: Creates GitHub issues with mapped data
5. **Comment Migration**: Migrates comments if enabled
6. **State Management**: Sets appropriate issue states
7. **Reporting**: Generates detailed migration report

## Output

### Console Output
Real-time progress with color-coded status messages:
- ✓ Success indicators
- ⚠ Warnings for non-critical issues  
- ✗ Error indicators for failures

### Migration Report
JSON report with detailed information:
- Total items processed
- Success/failure counts
- Individual item mappings
- Error details
- Processing duration

### Checkpoint Files
Automatic checkpoint creation for resume capability:
- `migration_checkpoint.json`: Current progress state
- Can resume from interruptions or failures

## Troubleshooting

### Common Issues

1. **Authentication Failures**
   - Verify PAT tokens have correct permissions
   - Check token expiration dates
   - Ensure organization/repository access

3. **Field Mapping Errors**
   - Validate field mapping configuration
   - Check for invalid GitHub label names
   - Verify user mapping accuracy

4. **Network Issues**
   - Check connectivity to both services
   - Verify proxy/firewall settings
   - Use appropriate base URLs for enterprise instances

### Debug Mode

Enable verbose logging for detailed troubleshooting:
```bash
./ado-gh-wi-migrator migrate --verbose
```

### Resume Failed Migrations

If migration is interrupted:
```bash
./ado-gh-wi-migrator migrate --resume
```

## API Limits

### Azure DevOps
- Rate limits vary by organization
- Batch size recommended: 50-100 items
- Monitor usage in Azure DevOps admin panel

### GitHub
- 5,000 requests per hour for authenticated requests
- Secondary rate limits apply for issue creation

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Submit a pull request

## License

[MIT License](LICENSE)

## Support

For issues and questions:
- Create an issue in the repository
- Check existing documentation
- Review troubleshooting section
