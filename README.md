# GitHub Repository Backup Tool

A secure, automated backup system for GitHub repositories that creates daily compressed archives and stores them with proper versioning.

## Features

-   **Automated Daily Backups**: Runs automatically via GitHub Actions
-   **Secure Authentication**: Uses GitHub Personal Access Tokens
-   **Compressed Storage**: Creates ZIP archives to save space
-   **Version Control**: Keeps last 5 backups per repository
-   **Notifications**: Sends webhook notifications on completion/failure
-   **Error Handling**: Robust error handling with retry mechanisms
-   **Security**: Token masking and input validation

## Setup

### 1. Repository Configuration

Add your repositories to `repositories.txt`:

```
https://github.com/username/repo1.git
https://github.com/username/repo2.git
```

### 2. GitHub Secrets

Set up the following secrets in your GitHub repository:

-   `BACKUP_TOKEN`: GitHub Personal Access Token with `repo` scope
-   `WEBHOOK_URL`: (Optional) Webhook URL for notifications

### 3. Token Permissions

Your GitHub Personal Access Token needs:

-   `repo` scope for private repositories
-   `public_repo` scope for public repositories

## Usage

### Manual Run

```bash
# Set environment variable
export BACKUP_TOKEN="your_github_token"

# Run backup
go run backup.go
```

### Automated Run

The system runs automatically every day at 2 AM UTC via GitHub Actions.

## Configuration

### Repository List

Edit `repositories.txt` to specify which repositories to backup:

```
https://github.com/username/repo1.git
https://github.com/username/repo2.git
# Comments start with #
```

### Backup Retention

The system automatically keeps the last 5 backups per repository and removes older ones.

## Output

-   **Backup Files**: Stored in `backups/owner/repo/YYYY-MM-DD.zip`
-   **Results**: Summary saved to `backup-results.json`
-   **Logs**: Structured logging with timestamps

## Security

-   All tokens are masked in logs
-   Input validation prevents path traversal
-   Secure file permissions (0750 for directories, 0640 for files)
-   HTTPS-only communication

## Troubleshooting

### Common Issues

1. **Authentication Failed**: Check your `BACKUP_TOKEN` and ensure it has the required permissions
2. **Repository Not Found**: Verify the repository URL and your access permissions
3. **Network Issues**: The system includes retry mechanisms for temporary failures

### Logs

Check the GitHub Actions logs for detailed error information. All sensitive data is automatically masked.

## Development

### Building

```bash
go build -o backup backup.go
```

### Testing

```bash
go test ./...
```

## License

This project is open source and available under the MIT License.
