# Backup System

A simple and efficient backup system for GitHub repositories built in Go.

## Features

- **Repository Mirroring**: Creates git mirrors of specified repositories
- **Credential Sanitization**: Removes authentication tokens from backup configs
- **Directory Management**: Creates and manages backup directory structure
- **Size Calculation**: Calculates and logs backup sizes
- **Retry Logic**: Automatic retry with exponential backoff for failed operations
- **Simple Logging**: Clear console output and log files
- **Summary Reports**: Markdown and JSON summaries of backup results

## Requirements

- Go 1.21 or later
- Git installed and accessible
- `du` command available (for size calculation)

## Environment Variables

- `BACKUP_TOKEN`: GitHub personal access token with repository access (required)

## Configuration

### Repository Setup

The system uses a simple text file (`repositories.txt`) to configure which repositories to backup. Each line should contain a repository URL:

```
https://github.com/username/repo1.git
https://github.com/username/repo2.git
git@github.com:username/repo3.git
```

**Supported URL formats:**

- HTTPS: `https://github.com/username/repo.git`
- SSH: `git@github.com:username/repo.git`

**File features:**

- Comments: Lines starting with `#` are ignored
- Empty lines: Are automatically skipped
- Repository names: Extracted automatically from URLs

## Usage

### Quick Start

```bash
# Set your GitHub token
export BACKUP_TOKEN="your_github_token_here"

# Run the backup system
go run backup.go
```

### Using Docker

```bash
# Build the Docker image
docker build -t backup-system .

# Run with environment variables
docker run --rm \
  -e BACKUP_TOKEN="your_github_token_here" \
  -v $(pwd)/backups:/app/backups \
  backup-system
```

### Using Makefile

```bash
# Install dependencies
make install

# Run tests
make test

# Build binary
make build

# Run the system
make run

# Validate configuration
make validate-config
```

## Output

The system creates the following structure:

```
backups/
├── repo1/
│   └── 2024-01-01/
│       └── repo1.git/
├── repo2/
│   └── 2024-01-01/
│       └── repo2.git/
└── summary/
    └── 2024-01-01/
        ├── README.md
        └── summary.json
```

### Log Files

- `backup-log.txt`: Simple log with success/failure messages
- Console output: Real-time progress and results

### Summary Reports

- `README.md`: Human-readable summary with success rates and details
- `summary.json`: Machine-readable JSON with complete backup metadata

## Error Handling

- **Retry Logic**: Failed git clones are retried up to 3 times with exponential backoff
- **Credential Sanitization**: Tokens are automatically removed from backup configs
- **Directory Management**: Existing backups are safely replaced
- **Error Reporting**: Clear error messages for debugging

## 🗓️ Daily Backup Logic

- If a backup for today already exists, it is **deleted before the new backup is taken**. This ensures you always have only the latest backup for each day, and never duplicate backups for the same date.
- The system will automatically keep only the 5 most recent daily backups per repository, deleting the oldest ones.

## Development

```bash
# Setup development environment
make dev

# Run all checks
make check

# Create release
make release
```

## Testing

```bash
# Run tests
make test

# Run tests with coverage
make test-coverage
```

## Security

- Non-root Docker containers
- Credential sanitization in backup configs
- Minimal attack surface with simple codebase
