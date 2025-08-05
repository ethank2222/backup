# Repository Backup to Azure Blob Storage

This GitHub Actions workflow automatically backs up repositories to Azure Blob Storage with a 10-day retention period and sends webhook notifications on success or failure.

## Features

-   **Manual Trigger**: Can be started manually via GitHub Actions UI
-   **Scheduled Backup**: Runs automatically at 2:00 AM UTC daily
-   **Mirror Cloning**: Creates bare git mirrors for efficient backups
-   **Azure Blob Storage**: Uploads compressed backups to Azure
-   **Automatic Cleanup**: Removes backups older than 10 days
-   **Webhook Notifications**: Sends notifications on success or failure
-   **Error Handling**: Comprehensive error handling and reporting

## Setup Instructions

### 1. Configure Azure Storage Account

1. Create an Azure Storage Account in your Azure portal
2. Create a container named `repo-backups` (or the workflow will create it automatically)
3. Get your storage account name and key from Azure portal

### 2. Configure GitHub Secrets

In your GitHub repository, go to **Settings** → **Secrets and variables** → **Actions** and add the following secrets:

-   `AZURE_STORAGE_ACCOUNT`: Your Azure Storage account name
-   `AZURE_STORAGE_KEY`: Your Azure Storage account key
-   `WEBHOOK_URL`: URL for webhook notifications (optional)
-   `BACKUP_TOKEN`: GitHub personal access token for private repositories (optional)

### 3. Configure Repository List

Edit the `repos.txt` file and add the repositories you want to backup:

```txt
# Repository URLs to backup
https://github.com/username/repo1.git
https://github.com/username/repo2.git
git@github.com:username/repo3.git
```

### 4. Private Repository Access (Optional)

If you're backing up private repositories, you'll need to set the `BACKUP_TOKEN` secret to a GitHub personal access token with `repo` scope. This allows the workflow to clone private repositories.

### 5. Webhook Configuration (Optional)

If you want webhook notifications, set the `WEBHOOK_URL` secret to your webhook endpoint. The webhook will receive JSON payloads like:

**Success:**

```json
{
    "success": true,
    "message": "All 3 repositories backed up successfully",
    "timestamp": "2024-01-15T10:30:00.000Z",
    "workflow": "repository-backup",
    "details": {
        "successful": 3,
        "backed_up_repos": [
            {
                "repo": "https://github.com/username/repo1.git",
                "blob_name": "repo1_20240115.tar.gz"
            }
        ]
    }
}
```

**Failure:**

```json
{
    "success": false,
    "message": "Backup completed with 1 failures",
    "timestamp": "2024-01-15T10:30:00.000Z",
    "workflow": "repository-backup",
    "details": {
        "successful": 2,
        "failed": 1,
        "failed_repos": [
            {
                "repo": "https://github.com/username/repo3.git",
                "error": "Clone failed"
            }
        ]
    }
}
```

## Workflow Details

### Triggers

-   **Manual**: Use the "Run workflow" button in GitHub Actions
-   **Scheduled**: Runs daily at 2:00 AM UTC

### Process

1. **Clone**: Creates bare git mirrors of each repository
2. **Compress**: Creates tar.gz archives of the repositories
3. **Upload**: Uploads to Azure Blob Storage with metadata
4. **Cleanup**: Removes backups older than 10 days
5. **Notify**: Sends webhook notifications

### Storage Structure

-   **Container**: `repo-backups`
-   **Blob Names**: `{repo-name}_{YYYYMMDD}.tar.gz` (e.g., `TrinityAI_20240115.tar.gz`)
-   **Metadata**: Includes backup date and TTL information

## Security Considerations

-   Use Azure Storage connection strings with minimal required permissions
-   Consider using Azure Key Vault for storing sensitive credentials
-   Ensure your webhook endpoint is secure and can handle the payload format
-   Review repository access permissions for private repositories

## Troubleshooting

### Common Issues

1. **Authentication Errors**: Verify Azure Storage credentials in GitHub secrets
2. **Repository Access**: Ensure the workflow has access to private repositories
3. **Webhook Failures**: Check webhook URL and endpoint availability
4. **Storage Quotas**: Monitor Azure Storage usage and quotas

### Logs

Check the GitHub Actions logs for detailed information about:

-   Repository cloning status
-   Upload progress
-   Cleanup operations
-   Webhook delivery status

## Testing Locally

You can test your configuration locally using the provided test script:

```bash
# Make the test script executable
chmod +x test_setup.sh

# Set your environment variables
export AZURE_STORAGE_ACCOUNT="your-storage-account"
export AZURE_STORAGE_KEY="your-storage-key"
export BACKUP_TOKEN="your-github-token"  # Optional

# Run the test
./test_setup.sh
```

The test script will verify:

-   Azure Storage connection and permissions
-   GitHub token validity (if provided)
-   Repository list file format

## Customization

You can modify the workflow to:

-   Change the schedule (edit the cron expression)
-   Adjust retention period (modify the cleanup logic)
-   Add additional notification methods
-   Customize the backup format or compression

## License

This project is open source and available under the MIT License.
