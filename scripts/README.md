# GitHub Terraformer Installation Scripts

This directory contains automation scripts to streamline the GitHub Terraformer installation process.

## Prerequisites

Before running the installation script, ensure you have:

1. **GitHub CLI (gh)** installed and authenticated:
   ```bash
   brew install gh
   gh auth login
   ```

2. **Organization admin access** to your GitHub organization

3. **HCP Terraform organization access**

4. **(Optional) yq** for YAML processing:
   ```bash
   brew install yq
   ```

## Installation

### Quick Start

```bash
cd /path/to/github-terraformer
chmod +x scripts/install.sh
./scripts/install.sh
```

### What Gets Automated

The script automates the following steps from the [INSTALLATION_GUIDE.md](../INSTALLATION_GUIDE.md):

✅ **Fully Automated:**
- Repository settings configuration (PR settings, merge methods)
- GitHub Actions permissions
- Deployment environments creation (plan, schedule, import, create-fork, promote)
- Environment secrets and variables
- Repository level variables
- app-list.yaml generation

⚠️ **Semi-Automated (with guidance):**
- Configuration repository creation from template
- Credentials collection and secure storage

📋 **Manual (with clear instructions):**
- GitHub App creation (Configuration App and Workflow Bot App)
  - The GitHub API doesn't support full app creation, so you'll need to use the web UI
  - The script provides step-by-step instructions and waits for you to complete each app setup
- HCP Terraform workspace setup
  - The script provides the exact values to use
- Branch protection ruleset
  - Complex ruleset configuration is easier through the web UI
  - The script provides exact settings to apply

### Installation Data

The script creates a `.install-data/` directory to store:
- App IDs and installation IDs
- Private keys (.pem files)
- HCP Terraform tokens and workspace names

**⚠️ IMPORTANT:**
- This directory contains sensitive data
- It's automatically added to `.gitignore`
- Keep it secure and do not commit it to version control
- You can safely delete it after installation completes

### Resume Installation

If the installation is interrupted, you can resume it by running the script again. It will:
- Skip steps that have already been completed
- Use previously saved data from `.install-data/`
- Continue from where you left off

### Clean Start

To start the installation from scratch:

```bash
./scripts/cleanup.sh
./scripts/install.sh
```

## Script Details

### install.sh

Main installation script that walks through all steps in the installation guide.

**Features:**
- Color-coded output for easy reading
- Progress tracking with checkmarks
- Secure credential handling
- Resume capability
- Validation at each step

**Usage:**
```bash
./scripts/install.sh
```

### cleanup.sh

Removes installation data to allow a fresh start.

**Usage:**
```bash
./scripts/cleanup.sh
```

**What it removes:**
- `.install-data/` directory and all its contents
- Any temporary files created during installation

**What it does NOT remove:**
- Created GitHub Apps (must be deleted manually from GitHub)
- Created repositories (must be deleted manually from GitHub)
- HCP Terraform workspaces (must be deleted manually from HCP)
- Deployed secrets and variables (will be overwritten on next install)

## Troubleshooting

### GitHub CLI not authenticated

**Error:** `GitHub CLI is not authenticated`

**Solution:**
```bash
gh auth login
```

### Permission denied

**Error:** `Permission denied when running script`

**Solution:**
```bash
chmod +x scripts/install.sh
```

### Repository already exists

If the configuration repository already exists, the script will ask if you want to use it or specify a different name.

### API Rate Limits

If you hit GitHub API rate limits, wait a few minutes and re-run the script. It will resume from where it left off.

## Manual Steps Reference

### Creating GitHub Apps

When the script asks you to create GitHub Apps, follow the instructions carefully:

1. Open the provided URL in your browser
2. Fill in the exact permissions listed by the script
3. Generate and download the private key
4. Note the App ID
5. Install the app to the specified repositories
6. Return to the script and provide the requested information

### HCP Terraform Setup

The script will display all the values you need to configure in HCP Terraform:

1. Log into HCP Terraform
2. Create a workspace with the provided name
3. Add the Terraform variables with the values shown
4. Create a Team API Token
5. Return to the script and provide the token

## Post-Installation

After the script completes:

1. **Update configuration files:**
   - Review and update `app-list.yaml` in the config repository
   - Update `import-config.yaml` to add your config repo to the ignore list

2. **Complete branch protection:**
   - Follow the instructions to create the branch protection ruleset
   - This is the last manual step

3. **Test the installation:**
   - Run the import or bulk-import workflow
   - This will verify GitHub App authentication and HCP connectivity

4. **Secure cleanup:**
   ```bash
   ./scripts/cleanup.sh
   ```

## Support

For issues or questions:
- Check the [INSTALLATION_GUIDE.md](../INSTALLATION_GUIDE.md) for detailed explanations
- Review the GitHub Terraformer documentation
- Open an issue in the repository

## Development

To modify the scripts:

1. Test changes in a non-production environment first
2. Ensure error handling is preserved
3. Update this README if adding new features
4. Keep the script idempotent (safe to run multiple times)
