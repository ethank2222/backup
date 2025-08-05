# Repository Backup to Azure Blob Storage

This GitHub Actions workflow automatically backs up repositories to Azure Blob Storage with a 10-day retention period and sends webhook notifications on success or failure.

## Features

-   **Manual Trigger**: Can be started manually via GitHub Actions UI
-   **Scheduled Backup**: Runs automatically at 2:00 AM UTC daily
-   **Mirror Cloning**: Creates bare git mirrors for efficient backups
-   **Azure Blob Storage**: Uploads compressed ZIP backups to Azure
-   **Automatic Cleanup**: Removes backups older than 10 days
-   **Webhook Notifications**: Sends Teams/Power Automate compatible notifications
-   **Error Handling**: Comprehensive error handling and reporting
-   **Success Tracking**: Tracks and reports successful repositories in notifications

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
https://github.com/username/repo1.git
https://github.com/username/repo2.git
https://github.com/username/repo3.git
```

### 4. Private Repository Access (Optional)

If you're backing up private repositories, you'll need to set the `BACKUP_TOKEN` secret to a GitHub personal access token with `repo` scope. This allows the workflow to clone private repositories.

### 5. Webhook Configuration (Optional)

If you want webhook notifications, set the `WEBHOOK_URL` secret to your webhook endpoint. The webhook will receive Teams/Power Automate compatible JSON payloads with:

-   **Success/Failure Status**: Clear indication of backup success or failure
-   **Successful Repositories**: List of all successfully backed up repositories
-   **Workflow Link**: Direct link to view the GitHub Actions workflow run
-   **Detailed Statistics**: Total repositories, success count, and failure count

**Example Success Payload:**

```json
{
    "@type": "MessageCard",
    "@context": "http://schema.org/extensions",
    "themeColor": "00FF00",
    "summary": "Repository Backup ✅ Success",
    "sections": [
        {
            "activityTitle": "GitHub Repository Backup",
            "activitySubtitle": "2024-01-15 10:30:00 UTC",
            "facts": [
                {
                    "name": "Status",
                    "value": "✅ Success"
                },
                {
                    "name": "Result",
                    "value": "Backup successful: All 3 repositories backed up"
                },
                {
                    "name": "Successful Repositories",
                    "value": "TrinityAI, TriniTeam, PageAI"
                },
                {
                    "name": "Workflow",
                    "value": "repository-backup"
                }
            ]
        }
    ],
    "potentialAction": [
        {
            "@type": "OpenUri",
            "name": "View Workflow Run",
            "targets": [
                {
                    "os": "default",
                    "uri": "https://github.com/username/repo/actions/runs/123456789"
                }
            ]
        }
    ]
}
```

## Workflow Details

### Triggers

-   **Manual**: Use the "Run workflow" button in GitHub Actions
-   **Scheduled**: Runs daily at 2:00 AM UTC

### Process

1. **Clone**: Creates bare git mirrors of each repository
2. **Compress**: Creates ZIP archives of the repositories
3. **Upload**: Uploads to Azure Blob Storage with metadata
4. **Cleanup**: Removes backups older than 10 days
5. **Notify**: Sends webhook notifications with success details

### Storage Structure

-   **Container**: `repo-backups`
-   **Blob Names**: `{repo-name}_{YYYYMMDD_HHMMSS}.zip` (e.g., `TrinityAI_20240115_143022.zip`)
-   **Format**: ZIP archives containing git mirror repositories

## Security Considerations

-   Use Azure Storage connection strings with minimal required permissions
-   Consider using Azure Key Vault for storing sensitive credentials
-   Ensure your webhook endpoint is secure and can handle the payload format
-   Review repository access permissions for private repositories
-   **Token Protection**: All tokens and credentials are automatically hidden from logs and output

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

## Project Structure

```
backup/
├── .github/
│   └── workflows/
│       └── backup-repos.yml    # Main workflow file
├── repos.txt                   # List of repositories to backup
└── README.md                   # This file
```

## Customization

You can modify the workflow to:

-   Change the schedule (edit the cron expression in the workflow)
-   Adjust retention period (modify the `RETENTION_DAYS` environment variable)
-   Add additional notification methods
-   Customize the backup format or compression
-   Modify webhook payload format

## License

This project is open source and available under the MIT License.
