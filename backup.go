package main

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Config holds the backup configuration
type Config struct {
	Token      string   `json:"token"`
	WebhookURL string   `json:"webhook_url"`
	Repos      []string `json:"repos"`
	MaxBackups int      `json:"max_backups"`
}

// Result represents a single backup result
type Result struct {
	Repo      string        `json:"repo"`
	Success   bool          `json:"success"`
	Error     string        `json:"error,omitempty"`
	Size      string        `json:"size,omitempty"`
	ZipSize   string        `json:"zip_size,omitempty"`
	Duration  time.Duration `json:"duration"`
}

// Summary holds the overall backup summary
type Summary struct {
	Date      string    `json:"date"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Duration  time.Duration `json:"duration"`
	Results   []Result  `json:"results"`
	Success   int        `json:"success_count"`
	Failed    int        `json:"failed_count"`
}

func main() {
	// Set up logging with timestamp
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	
	// Validate environment
	if err := validateEnvironment(); err != nil {
		log.Fatal("Environment validation failed:", err)
	}
	
	// Setup Git configuration
	if err := setupGit(); err != nil {
		log.Printf("Warning: Git setup failed: %v", err)
	}
	
	// Load config
	config, err := loadConfig()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}
	
	// Validate repositories
	if len(config.Repos) == 0 {
		log.Fatal("No repositories found in repositories.txt")
	}
	
	// Run backup
	summary := runBackup(config)
	
	// Save backup results
	if err := saveBackupResults(summary); err != nil {
		log.Printf("Warning: Failed to save backup results: %v", err)
	}
	
	// Commit and push changes
	noChanges, err := commitAndPush()
	if err != nil {
		log.Printf("Warning: Failed to commit/push: %v", err)
	}
	
	// Send notification
	if config.WebhookURL != "" {
		sendNotification(config.WebhookURL, summary, noChanges)
	}
	
	// Print summary
	printSummary(summary)
	
	// Run cleanup checks
	runCleanupChecks()
	
	// Exit with appropriate code
	if summary.Failed > 0 {
		os.Exit(1)
	}
}

func validateEnvironment() error {
	log.Printf("ℹ️  Validating environment variables...")
	
	if os.Getenv("BACKUP_TOKEN") == "" {
		return fmt.Errorf("BACKUP_TOKEN environment variable is required")
	}
	
	if os.Getenv("GITHUB_REPOSITORY") == "" {
		return fmt.Errorf("GITHUB_REPOSITORY environment variable is required")
	}
	
	// Validate Git is available
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git command not found: %w", err)
	}
	
	// Validate du command is available (for size calculation)
	if _, err := exec.LookPath("du"); err != nil {
		log.Printf("Warning: du command not found, size calculation may fail")
	}
	
	log.Printf("✅ Environment variables validated")
	return nil
}

func setupGit() error {
	log.Printf("ℹ️  Setting up Git configuration...")
	
	// Set Git user name
	cmd := exec.Command("git", "config", "--global", "user.name", "Backup Bot")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set git user name: %w", err)
	}
	
	// Set Git user email
	cmd = exec.Command("git", "config", "--global", "user.email", "ethank2222@gmail.com")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set git user email: %w", err)
	}
	
	log.Printf("✅ Git configuration completed")
	return nil
}

func loadConfig() (Config, error) {
	token := os.Getenv("BACKUP_TOKEN")
	
	// Read repositories from file
	content, err := os.ReadFile("repositories.txt")
	if err != nil {
		return Config{}, fmt.Errorf("failed to read repositories.txt: %w", err)
	}
	
	var repos []string
	for i, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Validate repository URL
		if !isValidRepoURL(line) {
			log.Printf("Warning: Invalid repository URL on line %d: %s", i+1, line)
			continue
		}
		
		repos = append(repos, line)
	}
	
	if len(repos) == 0 {
		return Config{}, fmt.Errorf("no valid repositories found in repositories.txt")
	}
	
	return Config{
		Token:      token,
		WebhookURL: os.Getenv("WEBHOOK_URL"),
		Repos:      repos,
		MaxBackups: 5,
	}, nil
}

func isValidRepoURL(url string) bool {
	url = strings.TrimSpace(url)
	return strings.HasPrefix(url, "https://github.com/") || 
		   strings.HasPrefix(url, "git@github.com:")
}

func runBackup(config Config) Summary {
	date := time.Now().Format("2006-01-02")
	summary := Summary{
		Date:      date,
		StartTime: time.Now(),
		Results:   []Result{},
	}
	
	log.Printf("Starting backup for %d repositories", len(config.Repos))
	
	// Create backup directories and backup repositories
	for _, repo := range config.Repos {
		repoName := extractRepoName(repo)
		if repoName == "" {
			log.Printf("Warning: Could not extract repo name from %s", repo)
			continue
		}
		
		backupDir := filepath.Join("backups", repoName, date)
		if err := os.MkdirAll(backupDir, 0755); err != nil {
			log.Printf("Warning: Failed to create backup directory %s: %v", backupDir, err)
			continue
		}
		
		result := backupRepo(config.Token, repo, repoName, backupDir)
		summary.Results = append(summary.Results, result)
		
		if result.Success {
			summary.Success++
		} else {
			summary.Failed++
		}
	}
	
	// Enforce retention
	enforceRetention(config.Repos, config.MaxBackups)
	
	summary.EndTime = time.Now()
	summary.Duration = summary.EndTime.Sub(summary.StartTime)
	
	return summary
}

func extractRepoName(url string) string {
	url = strings.TrimSuffix(url, ".git")
	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return ""
	}
	return parts[len(parts)-1]
}

func backupRepo(token, repoURL, repoName, backupDir string) Result {
	start := time.Now()
	result := Result{Repo: repoName}
	
	// Validate inputs
	if token == "" || repoURL == "" || repoName == "" || backupDir == "" {
		result.Error = "invalid input parameters"
		result.Duration = time.Since(start)
		return result
	}
	
	// Construct authenticated URL
	authURL := strings.Replace(repoURL, "https://", fmt.Sprintf("https://%s@", token), 1)
	
	// Clone repository
	gitPath := filepath.Join(backupDir, repoName+".git")
	zipPath := gitPath + ".zip"
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, "git", "clone", "--mirror", authURL, gitPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		result.Error = fmt.Sprintf("clone failed: %v, output: %s", err, string(output))
		result.Duration = time.Since(start)
		return result
	}
	
	// Remove credentials from config
	removeCredentials(gitPath, token)
	
	// Calculate size
	if size, err := getDirSize(gitPath); err == nil {
		result.Size = size
	}
	
	// Compress to zip
	if err := zipDir(gitPath, zipPath); err == nil {
		// Get zip size
		if zipSize, err := getFileSize(zipPath); err == nil {
			result.ZipSize = zipSize
		}
		// Remove uncompressed directory
		os.RemoveAll(gitPath)
	} else {
		log.Printf("Warning: failed to compress %s: %v", repoName, err)
	}
	
	result.Success = true
	result.Duration = time.Since(start)
	return result
}

func removeCredentials(gitPath, token string) {
	configPath := filepath.Join(gitPath, "config")
	content, err := os.ReadFile(configPath)
	if err != nil {
		return
	}
	
	newContent := strings.ReplaceAll(string(content), 
		fmt.Sprintf("https://%s@github.com", token), 
		"https://github.com")
	
	os.WriteFile(configPath, []byte(newContent), 0644)
}

func getDirSize(path string) (string, error) {
	cmd := exec.Command("du", "-sh", path)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	
	parts := strings.Fields(string(output))
	if len(parts) >= 1 {
		return parts[0], nil
	}
	return "unknown", nil
}

func getFileSize(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	return byteCountDecimal(info.Size()), nil
}

func byteCountDecimal(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "kMGTPE"[exp])
}

func zipDir(srcDir, zipPath string) error {
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()
	
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()
	
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		relPath, err := filepath.Rel(filepath.Dir(srcDir), path)
		if err != nil {
			return err
		}
		
		if info.IsDir() {
			if path == srcDir {
				return nil
			}
			header := &zip.FileHeader{
				Name:     relPath + "/",
				Method:   zip.Deflate,
				Modified: info.ModTime(),
			}
			_, err := zipWriter.CreateHeader(header)
			return err
		}
		
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = relPath
		header.Method = zip.Deflate
		
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}
		
		_, err = io.Copy(writer, file)
		return err
	})
}

func enforceRetention(repos []string, maxBackups int) {
	for _, repo := range repos {
		repoName := extractRepoName(repo)
		if repoName == "" {
			continue
		}
		
		repoPath := filepath.Join("backups", repoName)
		
		entries, err := os.ReadDir(repoPath)
		if err != nil {
			continue
		}
		
		var backupDirs []string
		for _, entry := range entries {
			if entry.IsDir() && isDateDir(entry.Name()) {
				backupDirs = append(backupDirs, entry.Name())
			}
		}
		
		if len(backupDirs) <= maxBackups {
			continue
		}
		
		// Sort and remove oldest
		sort.Strings(backupDirs)
		toDelete := backupDirs[:len(backupDirs)-maxBackups]
		
		for _, dir := range toDelete {
			dirPath := filepath.Join(repoPath, dir)
			os.RemoveAll(dirPath)
		}
	}
}

func isDateDir(name string) bool {
	if len(name) != 10 {
		return false
	}
	for i, c := range name {
		switch i {
		case 4, 7:
			if c != '-' {
				return false
			}
		default:
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return true
}

func saveBackupResults(summary Summary) error {
	jsonData, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal summary: %w", err)
	}
	
	return os.WriteFile("backup-results.json", jsonData, 0644)
}

func commitAndPush() (bool, error) {
	log.Printf("ℹ️  Checking for changes to commit...")
	
	// Add all files
	cmd := exec.Command("git", "add", ".")
	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("failed to add files: %w", err)
	}
	
	// Check if there are staged changes
	cmd = exec.Command("git", "diff", "--staged", "--quiet")
	if err := cmd.Run(); err != nil {
		// There are changes to commit
		log.Printf("ℹ️  Committing changes...")
		
		commitMsg := fmt.Sprintf("Daily mirror backup - %s", time.Now().Format("2006-01-02"))
		cmd = exec.Command("git", "commit", "-m", commitMsg)
		if err := cmd.Run(); err != nil {
			return false, fmt.Errorf("failed to commit: %w", err)
		}
		
		log.Printf("ℹ️  Pushing to repository...")
		cmd = exec.Command("git", "push", "origin", "main")
		if err := cmd.Run(); err != nil {
			return false, fmt.Errorf("failed to push: %w", err)
		}
		
		log.Printf("✅ Changes pushed successfully")
		return false, nil // false = changes were made
	}
	
	log.Printf("ℹ️  No changes to commit")
	return true, nil // true = no changes
}

func sendNotification(webhookURL string, summary Summary, noChanges bool) {
	// Validate webhook URL
	if webhookURL == "" {
		return
	}
	
	// Determine status and color
	status := "Success"
	color := "Good"
	title := "✅ GitHub Backup Successful"
	message := "Backup completed successfully"
	
	if summary.Failed > 0 {
		status = "Failure"
		color = "Attention"
		title = "❌ GitHub Backup Failed"
		message = "Backup process failed"
	} else if noChanges {
		message = "No new backups needed (repositories unchanged)"
	}

	// Build list of successful repositories
	var successfulRepos []string
	for _, result := range summary.Results {
		if result.Success {
			successfulRepos = append(successfulRepos, result.Repo)
		}
	}

	// Create workflow URL
	workflowURL := fmt.Sprintf("%s/%s/actions/runs/%s", 
		os.Getenv("GITHUB_SERVER_URL"), 
		os.Getenv("GITHUB_REPOSITORY"), 
		os.Getenv("GITHUB_RUN_ID"))

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
					"body": []map[string]interface{}{
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
						{
							"type": "TextBlock",
							"text": fmt.Sprintf("[View Workflow](%s)", workflowURL),
						},
						{
							"type": "FactSet",
							"facts": []map[string]string{
								{"title": "Repository:", "value": os.Getenv("GITHUB_REPOSITORY")},
								{"title": "Workflow Run ID:", "value": os.Getenv("GITHUB_RUN_ID")},
								{"title": "Timestamp:", "value": time.Now().UTC().Format("2006-01-02T15:04:05Z")},
								{"title": "Status:", "value": status},
								{"title": "Successful Repos:", "value": strings.Join(successfulRepos, ", ")},
							},
						},
					},
				},
			},
		},
	}

	// Send webhook with timeout
	client := &http.Client{Timeout: 30 * time.Second}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Warning: failed to marshal webhook message: %v", err)
		return
	}

	resp, err := client.Post(webhookURL, "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		log.Printf("Warning: failed to send webhook: %v", err)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		log.Printf("Warning: webhook returned status code %d", resp.StatusCode)
		return
	}
	
	log.Printf("✅ Webhook notification sent successfully")
}

func printSummary(summary Summary) {
	log.Printf("=== Backup Summary ===")
	log.Printf("Date: %s", summary.Date)
	log.Printf("Duration: %s", summary.Duration)
	log.Printf("Success: %d, Failed: %d", summary.Success, summary.Failed)
	
	for _, result := range summary.Results {
		if result.Success {
			log.Printf("✅ %s: %s", result.Repo, result.Size)
			if result.ZipSize != "" {
				log.Printf("   📦 ZIP: %s", result.ZipSize)
			}
			log.Printf("   ⏱️  Duration: %s", result.Duration)
		} else {
			log.Printf("❌ %s: %s", result.Repo, result.Error)
		}
	}
}

func runCleanupChecks() {
	log.Printf("ℹ️  Running cleanup checks...")
	
	// Check for backup results file
	if _, err := os.Stat("backup-results.json"); err == nil {
		log.Printf("ℹ️  Found backup results file")
	}
	
	// Check backup directory structure
	if _, err := os.Stat("backups"); err == nil {
		log.Printf("ℹ️  Backup directory exists")
		log.Printf("ℹ️  Backup summary:")
		
		// Count ZIP files
		zipCount := 0
		filepath.Walk("backups", func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() && strings.HasSuffix(path, ".zip") {
				zipCount++
			}
			return nil
		})
		
		log.Printf("  - ZIP files: %d", zipCount)
	} else {
		log.Printf("⚠️  No backup directory found")
	}
	
	log.Printf("✅ Cleanup checks completed")
} 