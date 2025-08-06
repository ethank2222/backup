#!/bin/bash
# EXACT COPY of repository processing logic from original workflow

# Source the backup function
source "$(dirname "$0")/backup-repo.sh"

# Initialize counters (EXACT COPY from original workflow)
SUCCESS_COUNT=0
FAIL_COUNT=0
FAILED_REPOS=""
SUCCESSFUL_REPOS=""
DATE_PREFIX=$(date +%Y%m%d_%H%M%S)

# Read all repositories into an array first (EXACT COPY from original workflow)
echo "📋 Reading repository list..."
declare -a REPOS_ARRAY

while IFS= read -r line; do
  # Skip comments and empty lines
  if [[ ! "$line" =~ ^[[:space:]]*# ]] && [[ -n "${line// }" ]]; then
    REPOS_ARRAY+=("$line")
  fi
done < repos.txt

TOTAL_REPOS=${#REPOS_ARRAY[@]}
echo "📋 Found $TOTAL_REPOS repositories to backup"
echo ""

# Process each repository from the array (EXACT COPY from original workflow)
for i in "${!REPOS_ARRAY[@]}"; do
  repo_url="${REPOS_ARRAY[$i]}"
  echo "[$(($i + 1))/$TOTAL_REPOS] Processing..."
  
  if backup_repo "$repo_url"; then
    SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
    SUCCESSFUL_REPOS="${SUCCESSFUL_REPOS}$(basename "$repo_url" .git), "
  else
    FAIL_COUNT=$((FAIL_COUNT + 1))
    FAILED_REPOS="${FAILED_REPOS}$(basename "$repo_url" .git), "
  fi
  echo ""
done 