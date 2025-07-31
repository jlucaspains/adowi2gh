# Getting Started with ado-gh-wi-migrator

This guide will help you set up and run your first migration from Azure DevOps to GitHub.

## Prerequisites

1. **Azure DevOps Access**
   - Access to your Azure DevOps organization
   - Personal Access Token with Work Items (read) permission
   - Project name you want to migrate from

2. **GitHub Access**
   - GitHub account with access to target repository
   - Personal Access Token with repo permissions
   - Target repository created and accessible

3. **Go Environment** (if building from source)
   - Go 1.19 or later installed
   - Basic knowledge of command line

## Step 1: Installation

### Option A: Build from Source
```bash
git clone <repository-url>
cd ado-gh-wi-migrator
go build -o build/ado-gh-wi-migrator.exe ./cmd/migrate
```

### Option B: Use PowerShell Script
```powershell
.\scripts\build.ps1 build
```

### Option C: Use Makefile (requires make)
```bash
make build
```

## Step 2: Create Configuration

Initialize a new configuration file:
```bash
.\build\ado-gh-wi-migrator.exe config init
```

This creates `configs/config.yaml` with default settings.

## Step 3: Configure Credentials

Edit `configs/config.yaml` and update the following sections:

### Azure DevOps Settings
```yaml
azure_devops:
  organization_url: "https://dev.azure.com/your-organization"
  personal_access_token: "your-ado-pat-token"
  project: "your-project-name"
```

### GitHub Settings
```yaml
github:
  token: "your-github-token"
  owner: "your-github-username-or-org"
  repository: "your-repository-name"
```

## Step 4: Set Up Personal Access Tokens

### Azure DevOps PAT
1. Go to Azure DevOps → User Settings → Personal Access Tokens
2. Click "New Token"
3. Give it a name like "GitHub Migration"
4. Select appropriate expiration date
5. Under Scopes, select "Work Items (read)"
6. Click "Create" and copy the token

### GitHub PAT
1. Go to GitHub → Settings → Developer settings → Personal access tokens
2. Click "Generate new token"
3. Give it a name like "ADO Migration"
4. Select appropriate expiration date
5. Under Scopes, select "repo" (Full control of private repositories)
6. Click "Generate token" and copy the token

## Step 5: Configure Work Item Query

Choose one of these approaches:

### Option A: Simple Filters
```yaml
query:
  work_item_types:
    - "Bug"
    - "User Story"
    - "Task"
  states:
    - "New"
    - "Active"
    - "Done"
```

### Option B: Custom WIQL Query
```yaml
query:
  wiql: |
    SELECT [System.Id] 
    FROM WorkItems 
    WHERE [System.TeamProject] = 'YourProject'
      AND [System.WorkItemType] IN ('Bug', 'User Story')
      AND [System.State] NOT IN ('Removed')
      AND [System.CreatedDate] >= '2024-01-01'
```

### Option C: Specific Work Item IDs
```yaml
query:
  ids: [123, 124, 125, 126]
```

## Step 6: Configure Field Mapping

Customize how ADO fields map to GitHub:

```yaml
field_mapping:
  state_mapping:
    "New": "open"
    "Active": "open"
    "Done": "closed"
  
  type_mapping:
    "Bug": ["bug"]
    "User Story": ["enhancement", "user-story"]
    "Task": ["task"]
  
  priority_mapping:
    "1": ["priority:critical"]
    "2": ["priority:high"]
    "3": ["priority:medium"]
    "4": ["priority:low"]
```

## Step 7: Configure User Mapping

Map ADO users to GitHub usernames:

```yaml
user_mapping:
  "john.doe@company.com": "johndoe"
  "jane.smith@company.com": "janesmith"
```

## Step 8: Validate Configuration

Test your configuration and connections:
```bash
.\build\ado-gh-wi-migrator.exe validate
```

This will verify:
- Configuration file is valid
- Azure DevOps connection works
- GitHub connection works
- Permissions are sufficient

## Step 9: Run Dry Run

Preview the migration without making changes:
```bash
.\build\ado-gh-wi-migrator.exe migrate --dry-run --verbose
```

This shows you:
- How many work items will be migrated
- What GitHub issues would be created
- Any potential issues or conflicts

## Step 10: Execute Migration

If the dry run looks good, run the actual migration:
```bash
.\build\ado-gh-wi-migrator.exe migrate --verbose
```

## Step 11: Review Results

After migration:
- Check the console output for summary
- Review the migration report (saved automatically)
- Verify issues in GitHub repository
- Check for any errors or warnings

## Common Configuration Examples

### Enterprise GitHub
```yaml
github:
  base_url: "https://github.company.com/api/v3"
```

### Include Comments and Attachments
```yaml
migration:
  include_comments: true
  include_attachments: true
```

### Custom Batch Size
```yaml
migration:
  batch_size: 25
```

## Troubleshooting

### Connection Issues
- Verify tokens are correct and not expired
- Check organization/repository names
- Ensure proper permissions

### Rate Limiting
- Reduce batch size
- Increase rate limiting delays
- Monitor GitHub rate limit status

### Field Mapping Issues
- Check label names (no spaces, special characters)
- Verify user mappings are accurate
- Test with dry run first

## Next Steps

- Set up regular incremental migrations
- Customize field mappings for your workflow
- Create team-specific configurations
- Set up CI/CD for automated migrations

## Support

For help:
- Check the main README.md file
- Review error messages carefully
- Use verbose logging for troubleshooting
- Create issues in the repository
