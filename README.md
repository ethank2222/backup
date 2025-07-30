# GitHub Repository Backup Tool

A production-ready Go application that creates secure, compressed backups of GitHub repositories with automated retention management and webhook notifications.

## Features

-   ✅ **Secure Git mirror cloning** - Complete repository backups with credential removal
-   ✅ **ZIP compression** - Reduces storage space by ~50-70%
-   ✅ **Automated retention** - Keeps only the 5 most recent backups per repository
-   ✅ **Rich webhook notifications** - Teams/Adaptive Card integration with detailed status
-   ✅ **Production logging** - Structured logging with timestamps and file locations
-   ✅ **Comprehensive error handling** - Graceful failure recovery and detailed error reporting
-   ✅ **Input validation** - URL validation and environment checks
-   ✅ **Cross-platform** - Works on Windows, Linux, and macOS
-   ✅ **Single binary** - No external dependencies beyond Git

## Requirements

-   **Go 1.21+** for building
-   **Git** for repository cloning
-   **du command** (optional, for size calculation)
-   **GitHub Personal Access Token** with repo access
-   **Teams Webhook URL** (optional, for notifications)

## Quick Start

### 1. Build the Application

```bash
go build -o backup backup.go
```

### 2. Create Configuration

**repositories.txt:**

```
https://github.com/username/repo1.git
https://github.com/username/repo2.git
# Comments start with #
```

### 3. Set Environment Variables

```bash
export BACKUP_TOKEN="ghp_your_github_token_here"
export GITHUB_REPOSITORY="username/backup-repo"
export WEBHOOK_URL="https://webhook.office.com/your_webhook_url"  # Optional
```

### 4. Run Backup

```bash
./backup
```

## Production Deployment

### GitHub Actions Workflow

```yaml
name: Daily Repository Backup

on:
    schedule:
        - cron: "0 2 * * *" # Daily at 2 AM UTC
    workflow_dispatch: # Manual trigger

jobs:
    backup:
        runs-on: ubuntu-latest

        steps:
            - uses: actions/checkout@v4

            - name: Setup Go
              uses: actions/setup-go@v4
              with:
                  go-version: "1.21"

            - name: Build backup tool
              run: go build -o backup backup.go

            - name: Run backup
              env:
                  BACKUP_TOKEN: ${{ secrets.GITHUB_TOKEN }}
                  GITHUB_REPOSITORY: ${{ github.repository }}
                  WEBHOOK_URL: ${{ secrets.WEBHOOK_URL }}
              run: ./backup
```

### Docker Deployment

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o backup backup.go

FROM alpine:latest
RUN apk add --no-cache git
WORKDIR /app
COPY --from=builder /app/backup .
COPY repositories.txt .
CMD ["./backup"]
```

### Kubernetes CronJob

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
    name: github-backup
spec:
    schedule: "0 2 * * *"
    jobTemplate:
        spec:
            template:
                spec:
                    containers:
                        - name: backup
                          image: your-registry/backup:latest
                          env:
                              - name: BACKUP_TOKEN
                                valueFrom:
                                    secretKeyRef:
                                        name: github-secrets
                                        key: token
                              - name: GITHUB_REPOSITORY
                                value: "username/backup-repo"
                              - name: WEBHOOK_URL
                                valueFrom:
                                    secretKeyRef:
                                        name: webhook-secrets
                                        key: url
                    restartPolicy: OnFailure
```

## Security Considerations

### Token Security

-   Use GitHub Personal Access Tokens with minimal required permissions
-   Store tokens in environment variables or secrets management systems
-   Never commit tokens to version control
-   Rotate tokens regularly

### Repository Access

-   Ensure the backup token has access to all repositories in `repositories.txt`
-   Use organization-level tokens for multiple repositories
-   Consider using GitHub Apps for fine-grained permissions

### File Permissions

-   Backup files are created with `0755` permissions
-   Git credentials are automatically removed from backup files
-   ZIP files are compressed and uncompressed directories are deleted

## Configuration

### Environment Variables

| Variable            | Required | Description                                     |
| ------------------- | -------- | ----------------------------------------------- |
| `BACKUP_TOKEN`      | Yes      | GitHub Personal Access Token                    |
| `GITHUB_REPOSITORY` | Yes      | Repository name (username/repo)                 |
| `WEBHOOK_URL`       | No       | Teams webhook URL for notifications             |
| `GITHUB_SERVER_URL` | No       | GitHub server URL (default: https://github.com) |
| `GITHUB_RUN_ID`     | No       | Workflow run ID for notifications               |

### Repository File Format

```
# One repository per line
https://github.com/username/repo1.git
https://github.com/username/repo2.git
git@github.com:username/repo3.git

# Comments start with #
# Empty lines are ignored
```

## Output Structure

```
backups/
├── repo1/
│   ├── 2025-01-15/
│   │   └── repo1.git.zip
│   └── 2025-01-16/
│       └── repo1.git.zip
├── repo2/
│   └── 2025-01-16/
│       └── repo2.git.zip
└── backup-results.json
```

## Monitoring and Logging

### Log Levels

-   **Info**: Normal operation messages
-   **Warning**: Non-critical issues (missing commands, failed compression)
-   **Error**: Critical failures (clone failures, missing tokens)

### Exit Codes

-   **0**: Success (all repositories backed up successfully)
-   **1**: Failure (one or more repositories failed)

### Webhook Notifications

-   **Success**: Green card with successful repository list
-   **Failure**: Red card with error details
-   **No Changes**: Special message when repositories are unchanged

## Troubleshooting

### Common Issues

**"git command not found"**

```bash
# Install Git
sudo apt-get install git  # Ubuntu/Debian
brew install git          # macOS
```

**"BACKUP_TOKEN environment variable is required"**

```bash
# Set the token
export BACKUP_TOKEN="ghp_your_token_here"
```

**"Failed to read repositories.txt"**

```bash
# Create the file
echo "https://github.com/username/repo.git" > repositories.txt
```

**"clone failed: authentication required"**

-   Verify the token has repository access
-   Check token expiration
-   Ensure repository URLs are correct

**"du command not found"**

-   Size calculation will use "unknown" instead
-   Install du: `sudo apt-get install coreutils`

### Debug Mode

Add verbose logging:

```bash
export DEBUG=1
./backup
```

### Manual Testing

Test individual components:

```bash
# Test Git access
git clone --mirror https://github.com/username/test-repo.git

# Test webhook
curl -H "Content-Type: application/json" -d '{"test": "message"}' $WEBHOOK_URL
```

## Performance Considerations

### Large Repositories

-   Default timeout is 10 minutes per repository
-   Consider increasing timeout for very large repositories
-   Monitor disk space usage

### Network Optimization

-   Use GitHub's closest mirror
-   Consider running during off-peak hours
-   Monitor bandwidth usage

### Storage Optimization

-   ZIP compression reduces size by 50-70%
-   Retention policy keeps only 5 most recent backups
-   Monitor backup directory size

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

MIT License - see LICENSE file for details.

## Support

For issues and questions:

1. Check the troubleshooting section
2. Review the logs for error details
3. Open an issue with detailed information
4. Include relevant log output and configuration

---

**Production Status**: ✅ Ready for production deployment
**Last Updated**: 2025-01-16
**Version**: 1.0.0
