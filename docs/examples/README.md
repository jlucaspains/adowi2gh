# Example Configurations

This directory contains example configuration files for different scenarios.

## Files

- `basic-config.yaml` - Basic configuration with minimal settings
- `enterprise-config.yaml` - Configuration for GitHub Enterprise
- `github-app-config.yaml` - Configuration using GitHub app instead of PAT for authentication

## Usage

Copy one of these files to `configs/config.yaml` and modify it with your specific settings:

```bash
cp docs/examples/basic-config.yaml configs/config.yaml
```

Then edit the file with your Azure DevOps and GitHub credentials and settings.

## Security Note

Never commit configuration files containing real tokens or credentials to version control. Use environment variables or secure configuration management for production deployments.
