#!/bin/bash
# EXACT COPY of backup_repo function from original workflow

backup_repo() {
  local repo_url="$1"
  local repo_name=$(basename "$repo_url" .git)
  local temp_dir=$(mktemp -d)
  
  echo "📦 Backing up: $repo_name ($repo_url)"
  
  # Clone repository
  if [ -n "$GITHUB_TOKEN" ] && [[ "$repo_url" == *"github.com"* ]]; then
    # Add token for private repos
    local auth_url="https://${GITHUB_TOKEN}@${repo_url#https://}"
  else
    local auth_url="$repo_url"
  fi
  
  # Clone with stdin redirected to prevent any consumption issues
  if ! git clone --mirror "$auth_url" "$temp_dir/$repo_name" </dev/null 2>/dev/null; then
    echo "❌ Failed to clone: $repo_name"
    rm -rf "$temp_dir"
    return 1
  fi
  
  # Create archive
  local archive_name="${repo_name}_${DATE_PREFIX}.zip"
  local archive_path="$temp_dir/$archive_name"
  
  (cd "$temp_dir" && zip -qr "$archive_name" "$repo_name")
  
  if [ ! -f "$archive_path" ]; then
    echo "❌ Failed to create archive: $repo_name"
    rm -rf "$temp_dir"
    return 1
  fi
  
  # Upload to Azure with stdin redirected
  if ! az storage blob upload \
    --account-name "$AZURE_STORAGE_ACCOUNT" \
    --account-key "$AZURE_STORAGE_KEY" \
    --container-name "$CONTAINER_NAME" \
    --name "$archive_name" \
    --file "$archive_path" \
    --overwrite \
    --output none </dev/null 2>/dev/null; then
    echo "❌ Failed to upload: $repo_name"
    rm -rf "$temp_dir"
    return 1
  fi
  
  echo "✅ Successfully backed up: $repo_name"
  rm -rf "$temp_dir"
  return 0
}

# Allow function to be sourced or called directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  if [ $# -eq 0 ]; then
    echo "❌ Usage: $0 <repository_url>"
    exit 1
  fi
  backup_repo "$1"
fi 