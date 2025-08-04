#!/bin/bash

# Test script to debug GitHub authentication issues

echo "🔍 Testing GitHub Authentication..."

# Check if BACKUP_TOKEN is set
if [ -z "$BACKUP_TOKEN" ]; then
    echo "❌ BACKUP_TOKEN is not set"
    exit 1
fi

echo "✅ BACKUP_TOKEN is set (length: ${#BACKUP_TOKEN})"

# Test GitHub API authentication
echo "🔍 Testing GitHub API authentication..."
if curl -H "Authorization: token $BACKUP_TOKEN" https://api.github.com/user | grep -q "login"; then
    echo "✅ GitHub API authentication successful"
else
    echo "❌ GitHub API authentication failed"
    exit 1
fi

# Test Git configuration
echo "🔍 Testing Git configuration..."
git config --global user.name "Backup Bot"
git config --global user.email "ethank2222@gmail.com"
git config --global credential.helper store
git config --global core.autocrlf false

echo "✅ Git configuration set"

# Test repository access
echo "🔍 Testing repository access..."
TEST_REPO="https://github.com/ethank2222/TrinityAI.git"

echo "Testing access to: $TEST_REPO"
echo "Authenticated URL format: https://[TOKEN]@github.com/ethank2222/TrinityAI.git"

# Try to clone a small test (using environment variable directly)
TEMP_DIR=$(mktemp -d)
if git clone --mirror "https://$BACKUP_TOKEN@github.com/ethank2222/TrinityAI.git" "$TEMP_DIR" 2>/dev/null; then
    echo "✅ Repository access successful"
    rm -rf "$TEMP_DIR"
else
    echo "❌ Repository access failed"
    rm -rf "$TEMP_DIR"
    exit 1
fi

echo "🎉 All authentication tests passed!" 