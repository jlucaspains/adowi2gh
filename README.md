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
- **HTML to Markdown Conversion**: Automatically converts HTML content to Markdown format

## Limitations

- **Images and Attachments**: Work item attachments and embedded images are not currently supported and will not be migrated

## Prerequisites

- Go 1.19 or later
- Azure DevOps Personal Access Token with Work Items (read) permission
- GitHub Personal Access Token or GitHub App with repository permissions
- Access to both Azure DevOps organization and target GitHub repository

## Getting Started
See the [Getting Started Guide](docs/GETTING_STARTED.md) for detailed setup instructions.

## Configuration

### Field Mapping

Configure how ADO fields map to GitHub:

```yaml
field_mapping:
  state_mapping:
    "New": "open"
    "Active": "open"
    "Done": "closed"
  
  type_mapping:
    "Bug": ["bug"]
    "User Story": ["enhancement"]
    "Task": ["task"]
  
  priority_mapping:
    "1": ["priority:critical"]
    "2": ["priority:high"]
    "3": ["priority:medium"]
    "4": ["priority:low"]
  
  # Include additional labels based on work item properties
  include_severity_label: true      # Adds severity:high, severity:critical, etc.
  include_area_path_label: true     # Adds area:frontend, area:backend, etc.
  time_zone: "America/New_York"     # Timezone for comment timestamps
```

### Migration Settings

Configure migration behavior:

```yaml
migration:
  batch_size: 50                    # Number of items to process per batch
  dry_run: false                    # Set to true for preview mode
  include_comments: true            # Migrate work item comments
  resume_from_checkpoint: false     # Resume from previous run
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
adowi2gh --help

# Show version information
adowi2gh version

# Initialize configuration
adowi2gh config init

# Validate configuration and test connections
adowi2gh validate

# Run migration
adowi2gh migrate [flags]
```

### Migration Flags

```bash
--dry-run          # Preview migration without making changes
--resume           # Resume from last checkpoint
--batch-size N     # Override batch size from config (default: 50)
--report FILE      # Specify output file for migration report
--config FILE      # Use specific configuration file
--verbose          # Enable verbose logging
```

### Examples

```bash
# Dry run to preview changes
adowi2gh migrate --dry-run

# Migrate with custom batch size
adowi2gh migrate --batch-size 25

# Resume interrupted migration
adowi2gh migrate --resume

# Use custom config file
adowi2gh migrate --config ./custom-config.yaml

# Verbose logging with custom report location
adowi2gh migrate --verbose --report ./reports/custom-migration.json

# Validate configuration with verbose output
adowi2gh validate --verbose
```

## Migration Process

1. **Connection Testing**: Validates connectivity to both Azure DevOps and GitHub
2. **Work Item Retrieval**: Queries ADO based on your configured query (WIQL, work item types, or specific IDs)
3. **Field Mapping**: Converts ADO fields to GitHub format with HTML-to-Markdown conversion
4. **Duplicate Detection**: Checks for existing GitHub issues to avoid duplicates
5. **Issue Creation**: Creates GitHub issues with mapped data and labels
6. **Comment Migration**: Migrates comments with original author attribution (if enabled)
7. **State Management**: Sets appropriate issue states (open/closed)
8. **Checkpoint Saving**: Creates resume points for large migrations
9. **Reporting**: Generates detailed migration report with mappings and errors

## Output

### Console Output
Real-time progress with structured logging:
- Informational messages about migration progress
- Work item processing status with IDs and titles
- Connection testing results
- Batch processing information

### Migration Report
JSON report saved to `reports/` directory with detailed information:
- Total items processed and timing information
- Success/failure/skipped counts
- Individual item mappings (ADO Work Item ID → GitHub Issue Number)
- Error details with specific failure reasons
- Migration metadata and configuration used

### Checkpoint Files
Automatic checkpoint creation for resume capability:
- `migration_checkpoint.json`: Current progress state with processed items
- Resume functionality to continue from interruptions
- Can resume from interruptions or failures

## Troubleshooting

### Common Issues

1. **Authentication Failures**
   - Verify PAT tokens and GitHub app (if applicable) have correct permissions
   - Check token expiration dates
   - Ensure organization/repository access

2. **Configuration Issues**
   - Verify YAML syntax in config file
   - Check required fields are populated
   - Ensure file paths and URLs are correct

3. **Field Mapping Errors**
   - Validate field mapping configuration
   - Check for invalid GitHub label names
   - Verify user mapping accuracy
   - Review HTML content conversion issues

4. **Network Issues**
   - Check connectivity to both services
   - Verify proxy/firewall settings
   - Use appropriate base URLs for enterprise instances

5. **Work Item Query Issues**
   - Validate WIQL syntax if using custom queries
   - Check work item type names and project access
   - Verify area path and iteration path permissions

6. **Duplicate Issues**
   - Tool automatically detects existing issues by work item ID
   - Check if issues already exist before re-running migration
   - Use `--resume` flag to continue from checkpoint

### Debug Mode

Enable verbose logging for detailed troubleshooting:
```bash
adowi2gh migrate --verbose
```

## API Limits and Performance

### Azure DevOps
- Rate limits vary by organization
- Batch size recommended: 25-50 items
- Default batch size: 50 items
- Monitor usage in Azure DevOps admin panel

### GitHub
- 5,000 requests per hour for authenticated requests
- Secondary rate limits apply for issue creation
- Built-in rate limiting with 2-second delays between batches

## Known Limitations

- **Attachments and Images**: Work item attachments and embedded images are not migrated
- **Work Item Links**: Relations between work items are not currently migrated
- **Rich Formatting**: Some complex HTML formatting may not convert well to Markdown

## Contributing

1. Fork the repository
2. Make your changes
3. Add tests for new functionality
4. Submit a pull request

## License

[MIT License](LICENSE)

## Support

For issues and questions:
- Create an issue in the repository
- Check existing documentation
- Review troubleshooting section
