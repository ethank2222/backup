#!/bin/bash
# EXACT COPY of send_webhook function from original workflow

send_webhook() {
  if [ -z "$WEBHOOK_URL" ]; then
    return 0
  fi
  
  local success="$1"
  local message="$2"
  local successful_repos="$3"
  local color=$([ "$success" = "true" ] && echo "00FF00" || echo "FF0000")
  local status=$([ "$success" = "true" ] && echo "✅ Success" || echo "❌ Failed")
  local workflow_url="https://github.com/${GITHUB_REPOSITORY:-unknown}/actions/runs/${GITHUB_RUN_ID:-}"
  
  # Create adaptive card format for Teams/Power Automate
  local payload=$(cat <<EOF
{
  "@type": "MessageCard",
  "@context": "http://schema.org/extensions",
  "themeColor": "$color",
  "summary": "Repository Backup $status",
  "sections": [{
    "activityTitle": "GitHub Repository Backup",
    "activitySubtitle": "$(date -u '+%Y-%m-%d %H:%M:%S UTC')",
    "activityImage": "https://github.githubassets.com/images/modules/logos_page/GitHub-Mark.png",
    "facts": [
      {
        "name": "Status",
        "value": "$status"
      },
      {
        "name": "Result",
        "value": "$message"
      },
      {
        "name": "Successful Repositories",
        "value": "$successful_repos"
      },
      {
        "name": "Workflow",
        "value": "repository-backup"
      },
      {
        "name": "Run ID",
        "value": "${GITHUB_RUN_ID:-N/A}"
      }
    ],
    "markdown": true
  }],
  "potentialAction": [{
    "@type": "OpenUri",
    "name": "View Workflow Run",
    "targets": [{
      "os": "default",
      "uri": "$workflow_url"
    }]
  }],
  "attachments": [{
    "contentType": "application/vnd.microsoft.card.adaptive",
    "content": {
      "type": "AdaptiveCard",
      "version": "1.0",
      "body": [
        {
          "type": "TextBlock",
          "size": "Medium",
          "weight": "Bolder",
          "text": "Repository Backup $status"
        },
        {
          "type": "TextBlock",
          "text": "$message",
          "wrap": true
        },
        {
          "type": "TextBlock",
          "text": "**Successful Repositories:** $successful_repos",
          "wrap": true
        },
        {
          "type": "TextBlock",
          "text": "[View Workflow Run]($workflow_url)",
          "wrap": true
        }
      ]
    }
  }]
}
EOF
)
  
  curl -X POST "$WEBHOOK_URL" \
    -H "Content-Type: application/json" \
    -d "$payload" \
    --max-time 10 || true
}

# Allow function to be sourced or called directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  if [ $# -lt 2 ]; then
    echo "❌ Usage: $0 <success> <message> [successful_repos]"
    exit 1
  fi
  send_webhook "$1" "$2" "${3:-}"
fi 