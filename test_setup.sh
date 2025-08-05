#!/bin/bash
"""
Simple test script to verify Azure Storage and GitHub token setup.
Run this locally to test your configuration.
"""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
PASSED=0
FAILED=0

print_result() {
    local test_name="$1"
    local result="$2"
    local message="$3"
    
    if [ "$result" = "true" ]; then
        echo -e "${GREEN}✅ PASS${NC}: $test_name"
        ((PASSED++))
    else
        echo -e "${RED}❌ FAIL${NC}: $test_name - $message"
        ((FAILED++))
    fi
}

test_azure_connection() {
    echo "Testing Azure Storage Connection..."
    
    # Check if environment variables are set
    if [ -z "$AZURE_STORAGE_ACCOUNT" ] || [ -z "$AZURE_STORAGE_KEY" ]; then
        print_result "Azure Storage Connection" false "AZURE_STORAGE_ACCOUNT or AZURE_STORAGE_KEY not found"
        return
    fi
    
    # Test container access
    CONTAINER_NAME="repo-backups"
    
    # Try to create container (will fail if it exists, which is fine)
    if az storage container create \
        --account-name "$AZURE_STORAGE_ACCOUNT" \
        --account-key "$AZURE_STORAGE_KEY" \
        --name "$CONTAINER_NAME" 2>/dev/null; then
        echo "  Container '$CONTAINER_NAME' created successfully"
    else
        echo "  Container '$CONTAINER_NAME' already exists"
    fi
    
    # Test blob upload
    TEST_BLOB_NAME="test_$(date +%Y%m%d_%H%M%S).txt"
    TEST_CONTENT="This is a test backup verification file"
    
    # Create temporary file
    TEMP_FILE=$(mktemp)
    echo "$TEST_CONTENT" > "$TEMP_FILE"
    
    if az storage blob upload \
        --account-name "$AZURE_STORAGE_ACCOUNT" \
        --account-key "$AZURE_STORAGE_KEY" \
        --container-name "$CONTAINER_NAME" \
        --name "$TEST_BLOB_NAME" \
        --file "$TEMP_FILE" \
        --overwrite 2>/dev/null; then
        echo "  Test blob '$TEST_BLOB_NAME' uploaded successfully"
        
        # Clean up test blob
        if az storage blob delete \
            --account-name "$AZURE_STORAGE_ACCOUNT" \
            --account-key "$AZURE_STORAGE_KEY" \
            --container-name "$CONTAINER_NAME" \
            --name "$TEST_BLOB_NAME" 2>/dev/null; then
            echo "  Test blob '$TEST_BLOB_NAME' deleted successfully"
            print_result "Azure Storage Connection" true ""
        else
            print_result "Azure Storage Connection" false "Failed to delete test blob"
        fi
    else
        print_result "Azure Storage Connection" false "Failed to upload test blob"
    fi
    
    # Clean up temp file
    rm -f "$TEMP_FILE"
}

test_github_token() {
    echo "Testing GitHub Token..."
    
    if [ -z "$BACKUP_TOKEN" ]; then
        echo -e "${YELLOW}⚠️  BACKUP_TOKEN not found (private repos may fail)${NC}"
        print_result "GitHub Token" true "Optional - not configured"
        return
    fi
    
    # Test GitHub API access
    if curl -s -H "Authorization: token $BACKUP_TOKEN" \
        "https://api.github.com/user" | grep -q '"login"'; then
        USER_LOGIN=$(curl -s -H "Authorization: token $BACKUP_TOKEN" \
            "https://api.github.com/user" | grep '"login"' | cut -d'"' -f4)
        echo "  GitHub token valid - authenticated as: $USER_LOGIN"
        print_result "GitHub Token" true ""
    else
        print_result "GitHub Token" false "Invalid token or API access failed"
    fi
}

test_repos_file() {
    echo "Testing Repository List File..."
    
    if [ ! -f "repos.txt" ]; then
        print_result "Repository List File" false "repos.txt file not found"
        return
    fi
    
    # Filter out comments and empty lines
    REPO_COUNT=$(grep -v '^#' repos.txt | grep -v '^$' | wc -l)
    
    if [ "$REPO_COUNT" -eq 0 ]; then
        echo -e "${YELLOW}⚠️  No repositories found in repos.txt${NC}"
        print_result "Repository List File" true "File exists but is empty"
    else
        echo "  Found $REPO_COUNT repositories in repos.txt"
        echo "  First few repositories:"
        grep -v '^#' repos.txt | grep -v '^$' | head -3 | while read -r repo; do
            echo "    - $repo"
        done
        if [ "$REPO_COUNT" -gt 3 ]; then
            echo "    ... and $((REPO_COUNT - 3)) more"
        fi
        print_result "Repository List File" true ""
    fi
}

main() {
    echo "🔍 Testing backup configuration..."
    echo ""
    
    # Run tests
    test_azure_connection
    echo ""
    
    test_github_token
    echo ""
    
    test_repos_file
    echo ""
    
    # Print summary
    echo "📊 Test Results:"
    echo "=================="
    echo "Total tests: $((PASSED + FAILED))"
    echo "Passed: $PASSED"
    echo "Failed: $FAILED"
    echo "=================="
    
    if [ $FAILED -eq 0 ]; then
        echo -e "${GREEN}🎉 All tests passed! Your backup configuration is ready.${NC}"
        echo "You can now run the GitHub Actions workflow."
    else
        echo -e "${RED}⚠️  Some tests failed. Please fix the issues before running the workflow.${NC}"
        echo ""
        echo "Common fixes:"
        echo "- Set AZURE_STORAGE_ACCOUNT and AZURE_STORAGE_KEY environment variables"
        echo "- Verify Azure Storage account permissions"
        echo "- Check GitHub token if using private repositories"
        echo "- Add repositories to repos.txt file"
    fi
}

# Check if Azure CLI is installed
if ! command -v az &> /dev/null; then
    echo -e "${RED}❌ Azure CLI not found. Please install it first.${NC}"
    echo "Install with: curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash"
    exit 1
fi

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo -e "${YELLOW}⚠️  jq not found. Installing...${NC}"
    sudo apt-get update && sudo apt-get install -y jq
fi

main 