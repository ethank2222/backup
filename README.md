# GitHub Repository Backup System

A comprehensive backup solution for GitHub repositories that creates compressed ZIP archives and stores them in Azure Blob Storage with Teams/Power Automate compatible notifications.

## Features

-   **Compressed ZIP backups** of complete repository mirrors
-   **Teams/Power Automate compatible notifications** with success/failure status
-   **Success tracking** with detailed repository lists
-   **Modular architecture** for easy local testing and development
-   **Line-for-line identical** code extraction from original workflow

## Project Structure

```
backup/
├── .github/workflows/
│   └── backup-repos-modular.yml      # New modular workflow
├── scripts/                          # Modular script components
│   ├── setup.sh                      # Environment setup
│   ├── backup-repo.sh                # Single repository backup
│   ├── send-webhook.sh               # Webhook notifications
│   ├── process-repos.sh              # Repository processing
│   ├── main.sh                       # Main orchestration
│   └── run-workflow.sh               # GitHub Actions entry point
├── repos.txt                         # Repository list
└── README.md                         # This file
```

## Quick Start

### 1. Configure Repository List

Edit `repos.txt` to include the repositories you want to backup:

```
https://github.com/username/repo1.git
https://github.com/username/repo2.git
https://github.com/username/private-repo.git
```

### 2. Set Up GitHub Secrets

Configure these secrets in your GitHub repository:

-   `AZURE_STORAGE_ACCOUNT`: Your Azure storage account name
-   `AZURE_STORAGE_KEY`: Your Azure storage account key
-   `BACKUP_TOKEN`: GitHub Personal Access Token (for private repos)
-   `WEBHOOK_URL`: Teams/Power Automate webhook URL (optional)

### 3. Run the Workflow

The workflow runs automatically daily at 2 AM UTC, or you can trigger it manually via GitHub Actions.

## Local Testing and Development

The backup system has been modularized for easy local testing and development. Each component can be tested independently.

### Prerequisites for Local Testing

1. **Bash shell** (Linux, macOS, or Windows with WSL/Git Bash)
2. **Azure CLI** installed and configured
3. **Git** installed
4. **zip** utility available

### Setting Up Local Environment

#### 1. Install Dependencies

**On Ubuntu/Debian:**

```bash
sudo apt-get update
sudo apt-get install -y git zip jq curl
curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash
```

**On macOS:**

```bash
brew install git zip jq curl azure-cli
```

**On Windows (with Chocolatey):**

```powershell
choco install git zip jq curl azure-cli
```

#### 2. Configure Environment Variables

Copy the environment template and fill in your values:

```bash
cp env.example .env
```

Edit `.env` with your actual credentials:

```bash
# Required
AZURE_STORAGE_ACCOUNT=your-storage-account-name
AZURE_STORAGE_KEY=your-storage-account-key
GITHUB_TOKEN=your-github-personal-access-token

# Optional
WEBHOOK_URL=https://your-webhook-url
CONTAINER_NAME=repo-backups
GITHUB_REPOSITORY=username/repository-name
```

#### 3. Set Script Permissions

```bash
chmod +x scripts/*.sh
```

### Testing Individual Components

The modular architecture allows you to test each component independently:

#### Test Webhook Functionality

```bash
# Test webhook with success message
scripts/send-webhook.sh true "Test success message" "test-repo"

# Test webhook with failure message
scripts/send-webhook.sh false "Test failure message" "test-repo"
```

#### Test Single Repository Backup

```bash
# Test backup of a specific repository
scripts/backup-repo.sh https://github.com/username/repo.git
```

#### Test Full Workflow

```bash
# Test the complete backup process
scripts/run-workflow.sh
```

### Manual Testing Commands

You can also test components directly without the test script:

```bash
# Test environment setup
scripts/setup.sh

# Test repository processing
scripts/process-repos.sh

# Test main orchestration
scripts/main.sh
```

### Testing with Different Scenarios

#### Test with Public Repositories

```bash
export GITHUB_TOKEN=""  # No token needed for public repos
scripts/backup-repo.sh https://github.com/microsoft/vscode.git
```

#### Test with Private Repositories

```bash
export GITHUB_TOKEN="your-personal-access-token"
scripts/backup-repo.sh https://github.com/username/private-repo.git
```

#### Test Webhook Notifications

```bash
export WEBHOOK_URL="https://your-teams-webhook-url"
scripts/send-webhook.sh true "Test message" "test-repo"
```

### Debugging and Troubleshooting

#### Check Environment Variables

```bash
# Verify all required variables are set
echo "AZURE_STORAGE_ACCOUNT: $AZURE_STORAGE_ACCOUNT"
echo "AZURE_STORAGE_KEY: $AZURE_STORAGE_KEY"
echo "GITHUB_TOKEN: $GITHUB_TOKEN"
echo "WEBHOOK_URL: $WEBHOOK_URL"
```

#### Test Azure Connection

```bash
# Test Azure storage connectivity
az storage account show --name "$AZURE_STORAGE_ACCOUNT"
az storage container list --account-name "$AZURE_STORAGE_ACCOUNT" --account-key "$AZURE_STORAGE_KEY"
```

#### Test GitHub Token

```bash
# Test GitHub token validity
curl -H "Authorization: token $GITHUB_TOKEN" https://api.github.com/user
```

#### View Script Output

```bash
# Run with verbose output
bash -x scripts/backup-repo.sh https://github.com/username/repo.git
```

### Development Workflow

1. **Make changes** to individual scripts in the `scripts/` directory
2. **Test locally** using the individual script commands
3. **Verify functionality** matches the original workflow
4. **Commit changes** and push to trigger GitHub Actions

## Webhook Configuration

The system sends Teams/Power Automate compatible notifications with:

-   **Success/Failure status** with color coding
-   **Successful repositories list**
-   **Direct link to workflow run**
-   **Detailed statistics** (total, succeeded, failed)
-   **Timestamp and repository information**

### Example Success Payload

```json
{
    "@type": "MessageCard",
    "@context": "http://schema.org/extensions",
    "themeColor": "00FF00",
    "summary": "Repository Backup ✅ Success",
    "sections": [
        {
            "activityTitle": "GitHub Repository Backup",
            "activitySubtitle": "2024-01-15 14:30:00 UTC",
            "facts": [
                { "name": "Status", "value": "✅ Success" },
                {
                    "name": "Result",
                    "value": "Backup successful: All 3 repositories backed up"
                },
                {
                    "name": "Successful Repositories",
                    "value": "repo1, repo2, repo3"
                },
                { "name": "Workflow", "value": "repository-backup" }
            ]
        }
    ],
    "potentialAction": [
        {
            "@type": "OpenUri",
            "name": "View Workflow Run",
            "targets": [
                {
                    "uri": "https://github.com/username/repo/actions/runs/123456789"
                }
            ]
        }
    ]
}
```

## Workflow Details

### Process

1. **Environment Setup**: Install Azure CLI and create storage container
2. **Repository Reading**: Parse `repos.txt` and load repositories into array
3. **Backup Processing**: For each repository:
    - Clone with mirror option
    - Create ZIP archive with timestamp
    - Upload to Azure Blob Storage
    - Track success/failure
4. **ZIP archives** stored as `{repo-name}_{YYYYMMDD_HHMMSS}.zip`
5. **Webhook notifications** with success details and workflow link

### Storage Structure

```
Azure Blob Storage Container: repo-backups/
├── repo1_20240115_143000.zip
├── repo2_20240115_143000.zip
└── repo3_20240115_143000.zip
```

### Retention Policy

**No retention policy** - backed-up data stays forever. This reduces complexity and eliminates the risk of accidental data loss.

## Customization

### Modify Schedule

Edit the cron expression in `.github/workflows/backup-repos-modular.yml`:

```yaml
schedule:
    - cron: "0 2 * * *" # Daily at 2 AM UTC
    - cron: "0 */6 * * *" # Every 6 hours
    - cron: "0 9 * * 1-5" # Weekdays at 9 AM UTC
```

### Add New Features

The modular architecture makes it easy to add new features:

1. **Add new scripts** to the `scripts/` directory
2. **Test locally** using the test script
3. **Integrate** into the main workflow

### Environment Variables

| Variable                | Required | Description                                  |
| ----------------------- | -------- | -------------------------------------------- |
| `AZURE_STORAGE_ACCOUNT` | Yes      | Azure storage account name                   |
| `AZURE_STORAGE_KEY`     | Yes      | Azure storage account key                    |
| `GITHUB_TOKEN`          | Yes      | GitHub Personal Access Token                 |
| `WEBHOOK_URL`           | No       | Teams/Power Automate webhook URL             |
| `CONTAINER_NAME`        | No       | Azure container name (default: repo-backups) |

## Troubleshooting

### Common Issues

#### "Failed to clone" errors

-   Check if repository is private and `GITHUB_TOKEN` is set
-   Verify repository URL is correct
-   Ensure token has appropriate permissions

#### "Failed to upload" errors

-   Verify Azure storage credentials
-   Check network connectivity
-   Ensure storage account has sufficient space

#### Webhook not sending

-   Check `WEBHOOK_URL` is set correctly
-   Verify webhook URL is accessible
-   Check Teams/Power Automate configuration

#### Script permission errors

```bash
chmod +x scripts/*.sh
```

### Getting Help

1. **Check logs** in GitHub Actions
2. **Test locally** using the individual script commands
3. **Verify environment** variables are set correctly
4. **Check Azure** and GitHub connectivity

## Migration from Original Workflow

The modular version is **functionally identical** to the original workflow:

-   **Same code** - extracted line-for-line
-   **Same behavior** - identical output and results
-   **Same configuration** - uses same secrets and settings
-   **Same schedule** - runs at same time

To migrate:

1. **Deploy** the new modular workflow
2. **Test** alongside the original workflow
3. **Verify** results match
4. **Switch** to modular workflow
5. **Remove** original workflow

## Benefits of Modular Architecture

1. **Local Testing**: Test components independently
2. **Faster Development**: Iterate quickly with local testing
3. **Better Debugging**: Isolate issues to specific components
4. **Easier Maintenance**: Modify individual components
5. **Code Reuse**: Use components in other projects
6. **Zero Risk**: Identical functionality to original workflow
