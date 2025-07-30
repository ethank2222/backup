# Simple Backup System

A clean, efficient backup system for GitHub repositories built in Go. This system provides reliable repository mirroring with minimal complexity and maximum reliability.

## 🚀 Features

### Core Functionality

- **Repository Mirroring**: Creates git mirrors of specified repositories
- **Credential Sanitization**: Removes authentication tokens from backup configs
- **Directory Management**: Creates and manages backup directory structure
- **Size Calculation**: Calculates and logs backup sizes
- **Retry Logic**: Automatic retry with exponential backoff for failed operations

### Simple & Reliable

- **Minimal Dependencies**: Only requires Go, Git, and `du` command
- **Clear Logging**: Simple console output and log files
- **Error Handling**: Comprehensive error handling with retry logic
- **Summary Reports**: Markdown and JSON summaries of backup results

## 📋 Requirements

### Prerequisites

- Go 1.21 or later
- Git installed and accessible
- `du` command available (for size calculation)

### Environment Variables

- `BACKUP_TOKEN`: GitHub personal access token with repository access (required)

## 🛠️ Installation

### Quick Start

```bash
# Clone the repository
git clone <repository-url>
cd backup

# Install dependencies
go mod download

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
# Setup development environment
make dev

# Run tests
make test

# Build binary
make build

# Run the system
make run
```

## 📖 Configuration

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

## 📁 Output Structure

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

## 🔧 Development

### Setup

```bash
# Install dependencies
make install

# Run all checks
make check

# Development workflow
make dev
```

### Testing

```bash
# Run tests
make test

# Run tests with coverage
make test-coverage

# Validate configuration
make validate-config
```

### Building

```bash
# Build binary
make build

# Create release
make release
```

## 🐳 Docker

```bash
# Build image
make docker-build

# Run container
make docker-run
```

## 🔒 Security

- Non-root Docker containers
- Credential sanitization in backup configs
- Minimal attack surface with simple codebase
- No unnecessary dependencies

## 📊 Error Handling

- **Retry Logic**: Failed git clones are retried up to 3 times with exponential backoff
- **Credential Sanitization**: Tokens are automatically removed from backup configs
- **Directory Management**: Existing backups are safely replaced
- **Error Reporting**: Clear error messages for debugging

## 🚀 GitHub Actions Integration

The system integrates seamlessly with GitHub Actions. The workflow at `.github/workflows/daily-backup.yml`:

1. Runs tests to ensure code quality
2. Sets up Go environment
3. Runs the backup script
4. Commits and pushes changes
5. Sends notifications

## 📈 Monitoring

- Console output shows real-time progress
- Log files provide detailed operation history
- Summary reports give overview of backup results
- GitHub Actions provides workflow monitoring

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests: `make test`
5. Submit a pull request

## 📝 License

This project is open source and available under the [MIT License](LICENSE).

## 🆘 Troubleshooting

### Common Issues

**Missing BACKUP_TOKEN**

```
BACKUP_TOKEN environment variable is required
```

Solution: Set the environment variable with your GitHub token.

**Git clone failures**

```
git clone failed: authentication failed
```

Solution: Ensure your token has the necessary repository permissions.

**Permission denied**

```
failed to create backup directory
```

Solution: Check directory permissions and ensure write access.

### Getting Help

- Check the logs in `backup-log.txt`
- Review the summary reports in `backups/summary/`
- Ensure all prerequisites are installed
- Verify your GitHub token has correct permissions
