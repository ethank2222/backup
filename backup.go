package main

import (
	"archive/zip"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Config holds application configuration
type Config struct {
	Repos []string
}

// Result represents the result of backing up a single repository
type Result struct {
	Repo     string
	Success  bool
	Error    string
	Size     int64
	ZipSize  int64
	Duration time.Duration
}

// Summary holds the overall backup summary
type Summary struct {
	Total    int
	Success  int
	Failed   int
	Results  []Result
	Duration time.Duration
}

var (
	lastWebhookTime time.Time
	webhookMutex    sync.Mutex
	logger          *slog.Logger
)

func main() {
	// Set up structured logging
	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{Key: "time", Value: slog.StringValue(a.Value.Time().Format("2006-01-02 15:04:05"))}
			}
			return a
		},
	}))
	slog.SetDefault(logger)
	
	// Track backup summary and early failure
	var summary Summary
	earlyFailure := false
	earlyFailureMessage := ""
	
	// Ensure notification is always sent, even on panic
	defer func() {
		if r := recover(); r != nil {
			logger.Error("Panic occurred", "panic", r)
			sendNotification("panic", "Backup process panicked", []string{})
			return
		}
		
		// Send appropriate notification based on backup result
		if earlyFailure {
			// Early failure occurred
			sendNotification("failure", earlyFailureMessage, []string{})
		} else if summary.Total == 0 {
			// No backup was attempted (early failure)
			sendNotification("failure", "Backup process failed before starting", []string{})
		} else if summary.Failed == summary.Total {
			// All backups failed
			sendNotification("failure", "All backups failed", []string{})
		} else if summary.Failed > 0 {
			// Some backups failed
			var successfulRepos []string
			for _, result := range summary.Results {
				if result.Success {
					successfulRepos = append(successfulRepos, extractRepoName(result.Repo))
				}
			}
			sendNotification("failure", "Some backups failed", successfulRepos)
		} else {
			// All backups succeeded
			var successfulRepos []string
			for _, result := range summary.Results {
				if result.Success {
					successfulRepos = append(successfulRepos, extractRepoName(result.Repo))
				}
			}
			sendNotification("success", "Backup completed successfully", successfulRepos)
		}
	}()
	
	// Validate environment
	if err := validateEnvironment(); err != nil {
		logger.Error("Environment validation failed", "error", err)
		earlyFailure = true
		earlyFailureMessage = fmt.Sprintf("Environment validation failed: %v", err)
		return
	}
	
	// Setup Git configuration
	if err := setupGit(); err != nil {
		logger.Warn("Git setup failed", "error", err)
	}
	
	// Load config
	config, err := loadConfig()
	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
		earlyFailure = true
		earlyFailureMessage = fmt.Sprintf("Failed to load configuration: %v", err)
		return
	}
	
	// Validate repositories
	if len(config.Repos) == 0 {
		logger.Error("No repositories found in repositories.txt")
		earlyFailure = true
		earlyFailureMessage = "No repositories found in repositories.txt"
		return
	}
	
	// Run backup
	summary = runBackup(config)
	
	// Save backup results
	if err := saveBackupResults(summary); err != nil {
		logger.Warn("Failed to save backup results", "error", err)
	}
	
	// Commit and push changes
	_, err = commitAndPush()
	if err != nil {
		logger.Warn("Failed to commit/push", "error", err)
	}
	
	// Print summary
	printSummary(summary)
	
	// Run cleanup checks
	runCleanupChecks()
	
	// Note: Exit code is handled by the defer function and webhook notifications
	// No need for os.Exit() as it would bypass the defer function
}

func validateEnvironment() error {
	// Check for required environment variables
	if os.Getenv("BACKUP_TOKEN") == "" {
		return fmt.Errorf("BACKUP_TOKEN environment variable is required")
	}
	
	// Check for required files
	if _, err := os.Stat("repositories.txt"); os.IsNotExist(err) {
		return fmt.Errorf("repositories.txt file is required")
	}
	
	// Check for required commands
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git command is required but not found")
	}
	
	// du command is optional (we have fallback)
	if _, err := exec.LookPath("du"); err != nil {
		logger.Warn("du command not found, will use fallback size calculation")
	}
	
	return nil
}

func setupGit() error {
	// Configure Git user
	cmd := exec.Command("git", "config", "--global", "user.name", "Backup Bot")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set git user name: %v", err)
	}
	
	cmd = exec.Command("git", "config", "--global", "user.email", "ethank2222@gmail.com")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set git user email: %v", err)
	}
	
	return nil
}

func loadConfig() (Config, error) {
	content, err := os.ReadFile("repositories.txt")
	if err != nil {
		return Config{}, fmt.Errorf("failed to read repositories.txt: %v", err)
	}
	
	var repos []string
	lines := strings.Split(string(content), "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			if isValidRepoURL(line) {
				repos = append(repos, line)
			} else {
				logger.Warn("Invalid repository URL", "url", line)
			}
		}
	}
	
	if len(repos) == 0 {
		return Config{}, fmt.Errorf("no valid repositories found in repositories.txt")
	}
	
	return Config{Repos: repos}, nil
}

func isValidRepoURL(url string) bool {
	// Strict GitHub URL validation - only allow valid repository paths
	pattern := `^https://github\.com/[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?/[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?$`
	matched, _ := regexp.MatchString(pattern, url)
	return matched
}

func runBackup(config Config) Summary {
	start := time.Now()
	var results []Result
	
	// Create backup directories
	for _, repo := range config.Repos {
		repoName := extractRepoName(repo)
		if repoName == "" {
			logger.Error("Failed to extract repo name from URL", "url", repo)
			results = append(results, Result{
				Repo:    repo,
				Success: false,
				Error:   "Failed to extract repository name",
			})
			continue
		}
		
		backupDir := filepath.Join("backups", repoName, time.Now().Format("2006-01-02"))
		if err := os.MkdirAll(backupDir, 0755); err != nil {
			logger.Error("Failed to create backup directory", "repo", repoName, "error", err)
			results = append(results, Result{
				Repo:    repo,
				Success: false,
				Error:   fmt.Sprintf("Failed to create backup directory: %v", err),
			})
			continue
		}
		
		// Backup the repository
		result := backupRepo(repo, repoName, backupDir)
		results = append(results, result)
	}
	
	// Calculate summary
	success := 0
	failed := 0
	for _, result := range results {
		if result.Success {
			success++
		} else {
			failed++
		}
	}
	
	return Summary{
		Total:    len(results),
		Success:  success,
		Failed:   failed,
		Results:  results,
		Duration: time.Since(start),
	}
}

func extractRepoName(repoURL string) string {
	if repoURL == "" {
		return ""
	}
	
	// Remove trailing slash and .git if present
	repoURL = strings.TrimSuffix(repoURL, "/")
	repoURL = strings.TrimSuffix(repoURL, ".git")
	
	// Split by / and get the last two parts
	parts := strings.Split(repoURL, "/")
	if len(parts) < 2 {
		return ""
	}
	
	// Sanitize owner and repo names to prevent path traversal
	owner := sanitizePathComponent(parts[len(parts)-2])
	repo := sanitizePathComponent(parts[len(parts)-1])
	
	// Return owner/repo format
	return fmt.Sprintf("%s/%s", owner, repo)
}

func sanitizePathComponent(component string) string {
	// Remove any potentially dangerous characters
	// Only allow alphanumeric, dots, hyphens, and underscores
	pattern := regexp.MustCompile(`[^a-zA-Z0-9._-]`)
	return pattern.ReplaceAllString(component, "")
}

func backupRepo(repoURL, repoName, backupDir string) Result {
	if repoURL == "" || repoName == "" || backupDir == "" {
		return Result{
			Repo:    repoURL,
			Success: false,
			Error:   "Invalid parameters provided",
		}
	}
	
	start := time.Now()
	
	// Construct authenticated URL
	authURL := constructAuthenticatedURL(repoURL)
	
	// Clone repository
	if err := cloneRepository(authURL, backupDir); err != nil {
		return Result{
			Repo:    repoURL,
			Success: false,
			Error:   fmt.Sprintf("Clone failed: %v", err),
		}
	}
	
	// Get directory size
	size, err := getDirectorySize(backupDir)
	if err != nil {
		logger.Warn("Failed to get directory size", "repo", repoName, "error", err)
	}
	
	// Create ZIP file
	zipPath := backupDir + ".zip"
	if err := zipDirectory(backupDir, zipPath); err != nil {
		// Clean up backup directory on ZIP failure
		if cleanupErr := os.RemoveAll(backupDir); cleanupErr != nil {
			logger.Warn("Failed to cleanup backup directory after ZIP failure", "repo", repoName, "error", cleanupErr)
		}
		return Result{
			Repo:    repoURL,
			Success: false,
			Error:   fmt.Sprintf("ZIP creation failed: %v", err),
		}
	}
	
	// Get ZIP file size
	zipSize, err := getFileSize(zipPath)
	if err != nil {
		logger.Warn("Failed to get ZIP size", "repo", repoName, "error", err)
	}
	
	// Remove original directory
	if err := os.RemoveAll(backupDir); err != nil {
		logger.Warn("Failed to remove backup directory", "repo", repoName, "error", err)
	}
	
	duration := time.Since(start)
	logger.Info("Backed up repository", "repo", repoName, "size", byteCountDecimal(size), "zipSize", byteCountDecimal(zipSize), "duration", duration)
	
	return Result{
		Repo:     repoURL,
		Success:  true,
		Size:     size,
		ZipSize:  zipSize,
		Duration: duration,
	}
}

func constructAuthenticatedURL(repoURL string) string {
	token := os.Getenv("BACKUP_TOKEN")
	if token == "" {
		return repoURL
	}
	
	// Replace https:// with https://token@
	return strings.Replace(repoURL, "https://", fmt.Sprintf("https://%s@", token), 1)
}

func cloneRepository(repoURL, backupDir string) error {
	cmd := exec.Command("git", "clone", "--mirror", repoURL, backupDir)
	// Suppress output to prevent token exposure in logs
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

func getDirectorySize(dir string) (int64, error) {
	// Try du command first (Unix/Linux)
	cmd := exec.Command("du", "-sb", dir)
	output, err := cmd.Output()
	if err == nil {
		// Parse output: "123456\t/path/to/dir"
		parts := strings.Fields(string(output))
		if len(parts) >= 1 {
			size, err := strconv.ParseInt(parts[0], 10, 64)
			if err == nil {
				return size, nil
			}
		}
	}
	
	// Fallback: calculate size manually
	var totalSize int64
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	
	if err != nil {
		return 0, fmt.Errorf("failed to calculate directory size: %v", err)
	}
	
	return totalSize, nil
}

func zipDirectory(sourceDir, zipPath string) error {
	zipfile, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer zipfile.Close()
	
	archive := zip.NewWriter(zipfile)
	defer archive.Close()
	
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		
		// Get relative path
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		header.Name = relPath
		
		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}
		
		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}
		
		if info.IsDir() {
			return nil
		}
		
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		
		_, err = io.Copy(writer, file)
		return err
	})
}

func getFileSize(filePath string) (int64, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func byteCountDecimal(bytes int64) string {
	const unit = 1000
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "kMGTPE"[exp])
}

func saveBackupResults(summary Summary) error {
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal summary: %v", err)
	}
	
	return os.WriteFile("backup-results.json", data, 0644)
}

func commitAndPush() (bool, error) {
	// Check if there are changes
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check git status: %v", err)
	}
	
	// Check if output is empty (no changes)
	if len(strings.TrimSpace(string(output))) == 0 {
		logger.Info("No changes to commit")
		return true, nil
	}
	
	// Add all changes
	cmd = exec.Command("git", "add", ".")
	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("failed to add changes: %v", err)
	}
	
	// Check if there are staged changes after adding
	cmd = exec.Command("git", "diff", "--staged", "--quiet")
	if err := cmd.Run(); err != nil {
		// There are staged changes, proceed with commit
		commitMsg := fmt.Sprintf("Daily mirror backup - %s", time.Now().Format("2006-01-02"))
		cmd = exec.Command("git", "commit", "-m", commitMsg)
		if err := cmd.Run(); err != nil {
			return false, fmt.Errorf("failed to commit changes: %v", err)
		}
		
		// Push changes
		cmd = exec.Command("git", "push", "origin", "main")
		// Suppress output to prevent token exposure in logs
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Run(); err != nil {
			return false, fmt.Errorf("failed to push changes: %v", err)
		}
		
		logger.Info("Changes committed and pushed")
		return false, nil
	}
	
	// No staged changes after adding
	logger.Info("No changes to commit after adding")
	return true, nil
}

func sendNotification(status, message string, successfulRepos []string) {
	webhookURL := os.Getenv("WEBHOOK_URL")
	if webhookURL == "" {
		logger.Warn("No webhook URL configured, skipping notification")
		return
	}
	
	// Simple rate limiting - prevent webhook spam
	webhookMutex.Lock()
	if time.Since(lastWebhookTime) < 5*time.Second {
		logger.Warn("Rate limiting webhook notifications")
		webhookMutex.Unlock()
		return
	}
	lastWebhookTime = time.Now()
	webhookMutex.Unlock()
	
	// Determine notification details based on status
	var title, color string
	switch status {
	case "success":
		title = "✅ GitHub Backup Successful"
		color = "Good"
	case "failure":
		title = "❌ GitHub Backup Failed"
		color = "Attention"
	case "panic":
		title = "💥 GitHub Backup Crashed"
		color = "Attention"
	default:
		title = "⚠️ GitHub Backup Status"
		color = "Warning"
	}
	
	// Get workflow information
	repo := os.Getenv("GITHUB_REPOSITORY")
	runID := os.Getenv("GITHUB_RUN_ID")
	serverURL := os.Getenv("GITHUB_SERVER_URL")
	
	// Create workflow URL
	workflowURL := ""
	if repo != "" && runID != "" {
		if serverURL == "" {
			serverURL = "https://github.com"
		}
		workflowURL = fmt.Sprintf("%s/%s/actions/runs/%s", serverURL, repo, runID)
	}
	
	// Build body elements
	body := []map[string]interface{}{
		{
			"type":   "TextBlock",
			"text":   title,
			"weight": "Bolder",
			"size":   "Large",
			"color":  color,
		},
		{
			"type": "TextBlock",
			"text": fmt.Sprintf("%s on %s", message, time.Now().Format("2006-01-02")),
			"wrap":  true,
		},
	}
	
	// Add workflow link if available
	if workflowURL != "" {
		body = append(body, map[string]interface{}{
			"type": "TextBlock",
			"text": fmt.Sprintf("[View Workflow](%s)", workflowURL),
		})
	}
	
	// Build facts section
	facts := []map[string]string{
		{"title": "Timestamp:", "value": time.Now().UTC().Format("2006-01-02T15:04:05Z")},
		{"title": "Status:", "value": strings.Title(status)},
	}
	
	if repo != "" {
		facts = append(facts, map[string]string{"title": "Repository:", "value": repo})
	}
	
	if runID != "" {
		facts = append(facts, map[string]string{"title": "Workflow Run ID:", "value": runID})
	}
	
	if len(successfulRepos) > 0 {
		facts = append(facts, map[string]string{"title": "Successful Repos:", "value": strings.Join(successfulRepos, ", ")})
	}
	
	// Add facts to body
	body = append(body, map[string]interface{}{
		"type": "FactSet",
		"facts": facts,
	})
	
	// Create webhook payload
	payload := map[string]interface{}{
		"type": "message",
		"attachments": []map[string]interface{}{
			{
				"contentType": "application/vnd.microsoft.card.adaptive",
				"content": map[string]interface{}{
					"$schema": "http://adaptivecards.io/schemas/adaptive-card.json",
					"type":    "AdaptiveCard",
					"version": "1.3",
					"body":    body,
				},
			},
		},
	}
	
	// Send webhook with timeout and proper TLS configuration
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		},
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		logger.Warn("failed to marshal webhook message", "error", err)
		return
	}
	
	resp, err := client.Post(webhookURL, "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		logger.Warn("failed to send webhook", "error", err)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		logger.Warn("webhook returned status code", "statusCode", resp.StatusCode)
		return
	}
	
	logger.Info("Webhook notification sent successfully", "status", status)
}

func printSummary(summary Summary) {
	logger.Info("Backup Summary", "totalRepos", summary.Total, "successful", summary.Success, "failed", summary.Failed, "duration", summary.Duration)
	
	if summary.Failed > 0 {
		logger.Error("Failed backups", "results", summary.Results)
	}
	
	if summary.Success > 0 {
		logger.Info("Successful backups", "results", summary.Results)
	}
}

func runCleanupChecks() {
	logger.Info("Running cleanup checks...")
	
	// Check for backup results file
	if _, err := os.Stat("backup-results.json"); err == nil {
		logger.Info("Found backup results file")
	}
	
	// Check backup directory
	if _, err := os.Stat("backups"); err == nil {
		logger.Info("Backup directory exists")
		
		// Count ZIP files
		zipFiles, err := filepath.Glob("backups/**/*.zip")
		if err == nil {
			logger.Info("ZIP files", "count", len(zipFiles))
		}
		
		// Clean up old backups (keep only last 5)
		if err := cleanupOldBackups(); err != nil {
			logger.Warn("Failed to cleanup old backups", "error", err)
		}
	} else {
		logger.Warn("No backup directory found")
	}
	
	logger.Info("Cleanup checks completed")
}

func cleanupOldBackups() error {
	// Find all backup directories
	backupDirs, err := filepath.Glob("backups/*/*")
	if err != nil {
		return fmt.Errorf("failed to find backup directories: %v", err)
	}
	
	for _, dir := range backupDirs {
		// Check if it's a directory
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			continue
		}
		
		// Find all ZIP files in this directory
		zipFiles, err := filepath.Glob(filepath.Join(dir, "*.zip"))
		if err != nil {
			logger.Warn("Failed to find ZIP files in directory", "dir", dir, "error", err)
			continue
		}
		
		// If we have more than 5 backups, remove the oldest ones
		if len(zipFiles) > 5 {
			// Sort files by modification time (oldest first)
			type fileInfo struct {
				path    string
				modTime time.Time
			}
			
			var files []fileInfo
			for _, file := range zipFiles {
				info, err := os.Stat(file)
				if err != nil {
					logger.Warn("Failed to stat file", "file", file, "error", err)
					continue
				}
				files = append(files, fileInfo{
					path:    file,
					modTime: info.ModTime(),
				})
			}
			
			// Sort by modification time (oldest first)
			for i := 0; i < len(files)-1; i++ {
				for j := i + 1; j < len(files); j++ {
					if files[i].modTime.After(files[j].modTime) {
						files[i], files[j] = files[j], files[i]
					}
				}
			}
			
			// Remove oldest files (keep last 5)
			filesToRemove := len(files) - 5
			for i := 0; i < filesToRemove; i++ {
				if err := os.Remove(files[i].path); err != nil {
					logger.Warn("Failed to remove old backup", "file", files[i].path, "error", err)
				} else {
					logger.Info("Removed old backup", "file", filepath.Base(files[i].path))
				}
			}
			
			logger.Info("Cleaned up directory", "dir", filepath.Base(dir), "keptBackups", 5, "removedBackups", filesToRemove)
		}
	}
	
	return nil
} 