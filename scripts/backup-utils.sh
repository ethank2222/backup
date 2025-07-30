#!/bin/bash

# Backup Utilities Script
# Provides modular functions for the backup process with clear error handling

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Error handling
handle_error() {
    local exit_code=$?
    log_error "Script failed with exit code $exit_code"
    exit $exit_code
}

trap handle_error ERR

# Validate environment variables
validate_environment() {
    log_info "Validating environment variables..."
    
    if [ -z "${BACKUP_TOKEN:-}" ]; then
        log_error "BACKUP_TOKEN environment variable is required"
        exit 1
    fi
    
    if [ -z "${GITHUB_REPOSITORY:-}" ]; then
        log_error "GITHUB_REPOSITORY environment variable is required"
        exit 1
    fi
    
    log_success "Environment variables validated"
}

# Setup Git configuration
setup_git() {
    log_info "Setting up Git configuration..."
    
    git config --global user.name "Backup Bot"
    git config --global user.email "ethank2222@gmail.com"
    
    log_success "Git configuration completed"
}

# Validate backup results
validate_backup_results() {
    log_info "Validating backup results..."
    
    if [ ! -f "backup-results.json" ]; then
        log_warning "No backup results file found"
        return 1
    fi
    
    # Check if the JSON is valid
    if ! jq empty backup-results.json 2>/dev/null; then
        log_error "Invalid JSON in backup results file"
        return 1
    fi
    
    log_success "Backup results validated"
    return 0
}

# Commit and push changes
commit_and_push() {
    log_info "Checking for changes to commit..."
    
    git add .
    
    if git diff --staged --quiet; then
        echo "NO_CHANGES=true" >> $GITHUB_ENV
        log_info "No changes to commit"
        return 0
    fi
    
    log_info "Committing changes..."
    git commit -m "Daily mirror backup - $(date +%Y-%m-%d)"
    
    log_info "Pushing to repository..."
    if git push origin main; then
        echo "BACKUP_COMMITTED=true" >> $GITHUB_ENV
        log_success "Changes pushed successfully"
        return 0
    else
        log_error "Failed to push changes"
        return 1
    fi
}

# Send webhook notification
send_webhook_notification() {
    local webhook_url="${WEBHOOK_URL:-}"
    local job_status="${1:-unknown}"
    local workflow_run_id="${GITHUB_RUN_ID:-}"
    local repository="${GITHUB_REPOSITORY:-}"
    
    log_info "Preparing webhook notification..."
    
    # Read backup results
    local results='{"success":false,"error":"No backup results found"}'
    if [ -f "backup-results.json" ]; then
        results=$(cat backup-results.json)
        log_info "Found backup results"
    else
        log_warning "No backup results file found"
    fi
    
    # Determine notification content
    local title=""
    local message=""
    
    case "$job_status" in
        "success")
            title="✅ GitHub Backup Successful"
            if [ "${NO_CHANGES:-}" = "true" ]; then
                message="No new backups needed (repositories unchanged)"
                log_info "No changes detected"
            else
                message="Backup completed successfully"
                log_success "Backup completed with changes"
            fi
            ;;
        "failure")
            title="❌ GitHub Backup Failed"
            message="Backup process failed"
            log_error "Backup process failed"
            ;;
        *)
            title="⚠️ GitHub Backup Cancelled"
            message="Backup process was cancelled"
            log_warning "Backup process was cancelled"
            ;;
    esac
    
    # Send webhook if URL is provided
    if [ -n "$webhook_url" ]; then
        log_info "Sending webhook notification..."
        
        # Create the webhook payload according to the specified schema
        local payload=$(cat <<EOF
{
  "type": "message",
  "attachments": [
    {
      "contentType": "application/vnd.microsoft.card.adaptive",
      "content": {
        "\$schema": "http://adaptivecards.io/schemas/adaptive-card.json",
        "type": "AdaptiveCard",
        "version": "1.3",
        "body": [
          {
            "type": "TextBlock",
            "text": "$title",
            "weight": "Bolder",
            "size": "Large",
            "color": "$(if [ "$job_status" = "success" ]; then echo "Good"; elif [ "$job_status" = "failure" ]; then echo "Attention"; else echo "Default"; fi)"
          },
          {
            "type": "TextBlock",
            "text": "$message on $(date +%Y-%m-%d)",
            "wrap": true
          },
          {
            "type": "TextBlock",
            "text": "[View Workflow](${GITHUB_SERVER_URL:-https://github.com}/$repository/actions/runs/$workflow_run_id)",
            "type": "TextBlock"
          },
          {
            "type": "FactSet",
            "facts": [
              {
                "title": "Repository:",
                "value": "$repository"
              },
              {
                "title": "Workflow Run ID:",
                "value": "$workflow_run_id"
              },
              {
                "title": "Timestamp:",
                "value": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
              },
              {
                "title": "Status:",
                "value": "$job_status"
              }
            ]
          }
        ]
      }
    }
  ]
}
EOF
)
        
        if curl -H 'Content-Type: application/json' \
               -d "$payload" \
               "$webhook_url" >/dev/null 2>&1; then
            log_success "Webhook notification sent successfully"
        else
            log_error "Failed to send webhook notification"
            return 1
        fi
    else
        log_info "No webhook URL configured, skipping notification"
    fi
}

# Run cleanup checks
run_cleanup_checks() {
    log_info "Running cleanup checks..."
    
    # Check for temporary files
    if [ -f "backup-results.json" ]; then
        log_info "Found backup results file"
    fi
    
    # Check backup directory structure
    if [ -d "backups" ]; then
        log_info "Backup directory exists"
        log_info "Backup summary:"
        
        local zip_count=$(find backups -name "*.zip" 2>/dev/null | wc -l)
        local summary_count=$(find backups -name "summary.json" 2>/dev/null | wc -l)
        
        echo "  - ZIP files: $zip_count"
        echo "  - Summary files: $summary_count"
    else
        log_warning "No backup directory found"
    fi
    
    log_success "Cleanup checks completed"
}

# Main function
main() {
    local action="${1:-}"
    
    case "$action" in
        "validate")
            validate_environment
            ;;
        "setup-git")
            setup_git
            ;;
        "validate-results")
            validate_backup_results
            ;;
        "commit-push")
            commit_and_push
            ;;
        "send-notification")
            local job_status="${2:-unknown}"
            send_webhook_notification "$job_status"
            ;;
        "cleanup")
            run_cleanup_checks
            ;;
        *)
            log_error "Unknown action: $action"
            echo "Usage: $0 {validate|setup-git|validate-results|commit-push|send-notification|cleanup}"
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@" 