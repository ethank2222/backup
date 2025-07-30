package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestRepositoryConfig tests the repository configuration structure
func TestRepositoryConfig(t *testing.T) {
	config := RepositoryConfig{
		Name: "TestRepo",
		URL:  "https://github.com/test/repo.git",
	}

	if config.Name != "TestRepo" {
		t.Errorf("Expected name TestRepo, got %s", config.Name)
	}

	if config.URL != "https://github.com/test/repo.git" {
		t.Errorf("Expected URL https://github.com/test/repo.git, got %s", config.URL)
	}
}

// TestBackupResult tests the backup result structure
func TestBackupResult(t *testing.T) {
	result := BackupResult{
		Name:      "TestRepo",
		Success:   true,
		Size:      "1.2M",
		Duration:  time.Minute,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(time.Minute),
	}

	if result.Name != "TestRepo" {
		t.Errorf("Expected name TestRepo, got %s", result.Name)
	}

	if !result.Success {
		t.Error("Expected success to be true")
	}

	if result.Size != "1.2M" {
		t.Errorf("Expected size 1.2M, got %s", result.Size)
	}
}

// TestBackupSummary tests the backup summary structure
func TestBackupSummary(t *testing.T) {
	summary := BackupSummary{
		Date:         "2024-01-01",
		StartTime:    time.Now(),
		EndTime:      time.Now().Add(time.Minute),
		Duration:     time.Minute,
		SuccessCount: 2,
		FailureCount: 1,
		Results: []BackupResult{
			{Name: "Repo1", Success: true},
			{Name: "Repo2", Success: false},
		},
	}

	if summary.Date != "2024-01-01" {
		t.Errorf("Expected date 2024-01-01, got %s", summary.Date)
	}

	if summary.SuccessCount != 2 {
		t.Errorf("Expected success count 2, got %d", summary.SuccessCount)
	}

	if summary.FailureCount != 1 {
		t.Errorf("Expected failure count 1, got %d", summary.FailureCount)
	}

	if len(summary.Results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(summary.Results))
	}
}

// TestCreateBackupDirectories tests directory creation
func TestCreateBackupDirectories(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "backup-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	repositories := []RepositoryConfig{
		{Name: "TestRepo1", URL: "https://github.com/test/repo1.git"},
		{Name: "TestRepo2", URL: "https://github.com/test/repo2.git"},
	}

	err = createBackupDirectories(repositories, "2024-01-01")
	if err != nil {
		t.Fatalf("Failed to create backup directories: %v", err)
	}

	// Verify directories were created
	for _, repo := range repositories {
		expectedPath := filepath.Join("backups", repo.Name, "2024-01-01")
		if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
			t.Errorf("Expected directory %s was not created", expectedPath)
		}
	}
}

// TestRemoveCredentialsFromConfig tests credential removal
func TestRemoveCredentialsFromConfig(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "git-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create mock git config
	configPath := filepath.Join(tempDir, "config")
	configContent := `[remote "origin"]
	url = https://test-token@github.com/test/repo.git
	fetch = +refs/heads/*:refs/remotes/origin/*`

	err = os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Set environment variable
	os.Setenv("BACKUP_TOKEN", "test-token")
	defer os.Unsetenv("BACKUP_TOKEN")

	// Test credential removal
	err = removeCredentialsFromConfig(tempDir)
	if err != nil {
		t.Fatalf("Failed to remove credentials: %v", err)
	}

	// Verify credentials were removed
	updatedContent, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read updated config: %v", err)
	}

	if strings.Contains(string(updatedContent), "test-token") {
		t.Error("Credentials were not properly removed from config")
	}

	if !strings.Contains(string(updatedContent), "https://github.com/test/repo.git") {
		t.Error("URL was not properly sanitized")
	}
}

// TestLoadRepositoriesFromFile tests loading repositories from file
func TestLoadRepositoriesFromFile(t *testing.T) {
	// Create temporary file
	tempFile, err := os.CreateTemp("", "repos-test")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write test repositories
	testContent := `https://github.com/ethank2222/TrinityAI.git
https://github.com/ethank2222/TriniTeam.git
# This is a comment
https://github.com/ethank2222/PageAI.git

https://github.com/ethank2222/TestRepo.git`
	
	if _, err := tempFile.WriteString(testContent); err != nil {
		t.Fatalf("Failed to write test content: %v", err)
	}
	tempFile.Close()

	// Test loading repositories
	repos, err := loadRepositoriesFromFile(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to load repositories: %v", err)
	}

	// Verify results
	expectedRepos := []string{"TrinityAI", "TriniTeam", "PageAI", "TestRepo"}
	if len(repos) != len(expectedRepos) {
		t.Errorf("Expected %d repositories, got %d", len(expectedRepos), len(repos))
	}

	for i, expectedName := range expectedRepos {
		if repos[i].Name != expectedName {
			t.Errorf("Expected repository name %s, got %s", expectedName, repos[i].Name)
		}
	}
}

// TestExtractRepoNameFromURL tests URL parsing
func TestExtractRepoNameFromURL(t *testing.T) {
	testCases := []struct {
		url      string
		expected string
		hasError bool
	}{
		{"https://github.com/ethank2222/TrinityAI.git", "TrinityAI", false},
		{"https://github.com/ethank2222/TriniTeam.git", "TriniTeam", false},
		{"https://github.com/ethank2222/PageAI", "PageAI", false},
		{"git@github.com:ethank2222/TestRepo.git", "TestRepo", false},
		{"git@github.com:ethank2222/AnotherRepo", "AnotherRepo", false},
		{"invalid-url", "", true},
		{"https://github.com/", "", true},
		{"", "", true},
	}

	for _, tc := range testCases {
		result, err := extractRepoNameFromURL(tc.url)
		
		if tc.hasError {
			if err == nil {
				t.Errorf("Expected error for URL %s, but got none", tc.url)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for URL %s: %v", tc.url, err)
			}
			if result != tc.expected {
				t.Errorf("For URL %s, expected %s, got %s", tc.url, tc.expected, result)
			}
		}
	}
}

// TestLoadRepositoriesFromFileError tests error handling
func TestLoadRepositoriesFromFileError(t *testing.T) {
	// Test with non-existent file
	_, err := loadRepositoriesFromFile("non-existent-file.txt")
	if err == nil {
		t.Error("Expected error for non-existent file, but got none")
	}

	// Test with empty file
	tempFile, err := os.CreateTemp("", "empty-repos")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	_, err = loadRepositoriesFromFile(tempFile.Name())
	if err == nil {
		t.Error("Expected error for empty file, but got none")
	}

	// Test with file containing only comments and empty lines
	tempFile2, err := os.CreateTemp("", "comments-only")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile2.Name())

	if _, err := tempFile2.WriteString("# This is a comment\n\n# Another comment\n"); err != nil {
		t.Fatalf("Failed to write test content: %v", err)
	}
	tempFile2.Close()

	_, err = loadRepositoriesFromFile(tempFile2.Name())
	if err == nil {
		t.Error("Expected error for file with only comments, but got none")
	}
}

// TestCreateMarkdownSummary tests markdown summary creation
func TestCreateMarkdownSummary(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "summary-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	summary := BackupSummary{
		Date:         "2024-01-01",
		StartTime:    time.Now(),
		EndTime:      time.Now().Add(time.Minute),
		Duration:     time.Minute,
		SuccessCount: 1,
		FailureCount: 1,
		Results: []BackupResult{
			{
				Name:     "TestRepo1",
				Success:  true,
				Size:     "1.2M",
				Duration: time.Minute,
			},
			{
				Name:    "TestRepo2",
				Success: false,
				Error:   "test error",
				Duration: time.Second * 30,
			},
		},
	}

	summaryPath := filepath.Join(tempDir, "summary.md")
	createMarkdownSummary(summaryPath, summary)

	// Verify file was created
	if _, err := os.Stat(summaryPath); os.IsNotExist(err) {
		t.Error("Summary file was not created")
	}

	// Read and verify content
	content, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("Failed to read summary file: %v", err)
	}

	contentStr := string(content)

	// Verify key elements are present
	if !strings.Contains(contentStr, "Backup Summary - 2024-01-01") {
		t.Error("Summary title not found")
	}

	if !strings.Contains(contentStr, "TestRepo1") {
		t.Error("Repository name not found in summary")
	}

	if !strings.Contains(contentStr, "1.2M") {
		t.Error("Repository size not found in summary")
	}

	if !strings.Contains(contentStr, "test error") {
		t.Error("Error message not found in summary")
	}
}

// TestCreateJSONSummary tests JSON summary creation
func TestCreateJSONSummary(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "json-summary-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	summary := BackupSummary{
		Date:         "2024-01-01",
		StartTime:    time.Now(),
		EndTime:      time.Now().Add(time.Minute),
		Duration:     time.Minute,
		SuccessCount: 1,
		FailureCount: 1,
		Results: []BackupResult{
			{Name: "TestRepo", Success: true, Size: "1.2M"},
		},
	}

	jsonPath := filepath.Join(tempDir, "summary.json")
	createJSONSummary(jsonPath, summary)

	// Verify file was created
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Error("JSON summary file was not created")
	}

	// Read and verify content
	content, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("Failed to read JSON summary file: %v", err)
	}

	// Verify it's valid JSON
	var unmarshaledSummary BackupSummary
	err = json.Unmarshal(content, &unmarshaledSummary)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON summary: %v", err)
	}

	// Verify key fields
	if unmarshaledSummary.Date != summary.Date {
		t.Errorf("Date mismatch: expected %s, got %s", summary.Date, unmarshaledSummary.Date)
	}

	if unmarshaledSummary.SuccessCount != summary.SuccessCount {
		t.Errorf("SuccessCount mismatch: expected %d, got %d", summary.SuccessCount, unmarshaledSummary.SuccessCount)
	}
} 