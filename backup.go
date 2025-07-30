package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"archive/zip"
	"io"
)

// RepositoryConfig defines a repository to backup
type RepositoryConfig struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// BackupResult represents the result of backing up a single repository
type BackupResult struct {
	Name      string        `json:"name"`
	Success   bool          `json:"success"`
	Error     string        `json:"error,omitempty"`
	Size      string        `json:"size"`
	ZipSize   string        `json:"zip_size,omitempty"`
	Duration  time.Duration `json:"duration"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
}

// BackupSummary represents the overall backup summary
type BackupSummary struct {
	Date           string          `json:"date"`
	StartTime      time.Time       `json:"start_time"`
	EndTime        time.Time       `json:"end_time"`
	Duration       time.Duration   `json:"duration"`
	Results        []BackupResult  `json:"results"`
	SuccessCount   int             `json:"success_count"`
	FailureCount   int             `json:"failure_count"`
}

// WebhookLogger handles sending log messages to webhook
type WebhookLogger struct {
	webhookURL string
	client     *http.Client
}

// LogMessage represents a structured log message for webhook
type LogMessage struct {
	Type        string `json:"type"`
	Attachments []struct {
		ContentType string `json:"contentType"`
		Content     struct {
			Schema  string `json:"$schema"`
			Type    string `json:"type"`
			Version string `json:"version"`
			Body    []struct {
				Type string `json:"type"`
				Text string `json:"text,omitempty"`
				Weight string `json:"weight,omitempty"`
				Size string `json:"size,omitempty"`
				Color string `json:"color,omitempty"`
				Wrap bool `json:"wrap,omitempty"`
				Facts []struct {
					Title string `json:"title"`
					Value string `json:"value"`
				} `json:"facts,omitempty"`
			} `json:"body"`
		} `json:"content"`
	} `json:"attachments"`
}

// BackupManager handles the backup process
type BackupManager struct {
	logger      *slog.Logger
	webhookLog  *WebhookLogger
	backupToken string
	backupDate  string
}

// NewBackupManager creates a new backup manager
func NewBackupManager(backupToken, webhookURL string) *BackupManager {
	// Setup structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	webhookLog := &WebhookLogger{
		webhookURL: webhookURL,
		client:     &http.Client{Timeout: 30 * time.Second},
	}

	return &BackupManager{
		logger:      logger,
		webhookLog:  webhookLog,
		backupToken: backupToken,
		backupDate:  time.Now().Format("2006-01-02"),
	}
}

// Log sends a structured log message and optionally to webhook
func (bm *BackupManager) Log(level slog.Level, msg string, fields ...interface{}) {
	// Log to stdout
	bm.logger.Log(level, msg, fields...)

	// Send to webhook if URL is provided
	if bm.webhookLog.webhookURL != "" {
		// Create adaptive card webhook message
		webhookMsg := LogMessage{
			Type: "message",
			Attachments: []struct {
				ContentType string `json:"contentType"`
				Content     struct {
					Schema  string `json:"$schema"`
					Type    string `json:"type"`
					Version string `json:"version"`
					Body    []struct {
						Type string `json:"type"`
						Text string `json:"text,omitempty"`
						Weight string `json:"weight,omitempty"`
						Size string `json:"size,omitempty"`
						Color string `json:"color,omitempty"`
						Wrap bool `json:"wrap,omitempty"`
						Facts []struct {
							Title string `json:"title"`
							Value string `json:"value"`
						} `json:"facts,omitempty"`
					} `json:"body"`
				} `json:"content"`
			}{
				{
					ContentType: "application/vnd.microsoft.card.adaptive",
					Content: struct {
						Schema  string `json:"$schema"`
						Type    string `json:"type"`
						Version string `json:"version"`
						Body    []struct {
							Type string `json:"type"`
							Text string `json:"text,omitempty"`
							Weight string `json:"weight,omitempty"`
							Size string `json:"size,omitempty"`
							Color string `json:"color,omitempty"`
							Wrap bool `json:"wrap,omitempty"`
							Facts []struct {
								Title string `json:"title"`
								Value string `json:"value"`
							} `json:"facts,omitempty"`
						} `json:"body"`
					}{
						Schema:  "http://adaptivecards.io/schemas/adaptive-card.json",
						Type:    "AdaptiveCard",
						Version: "1.3",
						Body: []struct {
							Type string `json:"type"`
							Text string `json:"text,omitempty"`
							Weight string `json:"weight,omitempty"`
							Size string `json:"size,omitempty"`
							Color string `json:"color,omitempty"`
							Wrap bool `json:"wrap,omitempty"`
							Facts []struct {
								Title string `json:"title"`
								Value string `json:"value"`
							} `json:"facts,omitempty"`
						}{
							{
								Type:   "TextBlock",
								Text:   msg,
								Weight: "Bolder",
								Size:   "Medium",
								Color:  getColorForLevel(level),
							},
						},
					},
				},
			},
		}

		// Add fields as facts if present
		if len(fields) > 0 {
			facts := []struct {
				Title string `json:"title"`
				Value string `json:"value"`
			}{}
			
			for i := 0; i < len(fields); i += 2 {
				if i+1 < len(fields) {
					if key, ok := fields[i].(string); ok {
						facts = append(facts, struct {
							Title string `json:"title"`
							Value string `json:"value"`
						}{
							Title: key,
							Value: fmt.Sprintf("%v", fields[i+1]),
						})
					}
				}
			}
			
			if len(facts) > 0 {
				webhookMsg.Attachments[0].Content.Body = append(webhookMsg.Attachments[0].Content.Body, struct {
					Type string `json:"type"`
					Text string `json:"text,omitempty"`
					Weight string `json:"weight,omitempty"`
					Size string `json:"size,omitempty"`
					Color string `json:"color,omitempty"`
					Wrap bool `json:"wrap,omitempty"`
					Facts []struct {
						Title string `json:"title"`
						Value string `json:"value"`
					} `json:"facts,omitempty"`
				}{
					Type:  "FactSet",
					Facts: facts,
				})
			}
		}

		bm.webhookLog.Send(webhookMsg)
	}
}

// Send sends a log message to the webhook
func (wl *WebhookLogger) Send(msg LogMessage) {
	if wl.webhookURL == "" {
		return
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		return
	}

	resp, err := wl.client.Post(wl.webhookURL, "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

func main() {
	// Get required environment variables
	backupToken := os.Getenv("BACKUP_TOKEN")
	if backupToken == "" {
		fmt.Fprintf(os.Stderr, "BACKUP_TOKEN environment variable is required\n")
		os.Exit(1)
	}

	webhookURL := os.Getenv("WEBHOOK_URL")

	// Create backup manager
	bm := NewBackupManager(backupToken, webhookURL)

	// Start backup process
	if err := bm.RunBackup(); err != nil {
		bm.Log(slog.LevelError, "Backup process failed", "error", err.Error())
		os.Exit(1)
	}
}

// RunBackup executes the complete backup process
func (bm *BackupManager) RunBackup() error {
	bm.Log(slog.LevelInfo, "Starting backup process", "date", bm.backupDate)

	// Load repositories
	repositories, err := bm.loadRepositoriesFromFile("repositories.txt")
	if err != nil {
		return fmt.Errorf("failed to load repositories: %w", err)
	}

	bm.Log(slog.LevelInfo, "Loaded repositories", "count", len(repositories))

	// Create backup directories
	if err := bm.createBackupDirectories(repositories); err != nil {
		return fmt.Errorf("failed to create backup directories: %w", err)
	}

	// Perform backups
	summary := bm.performBackups(repositories)

	// Enforce retention
	bm.enforceRetention(repositories)

	// Create summary
	if err := bm.createBackupSummary(summary); err != nil {
		bm.Log(slog.LevelWarn, "Failed to create backup summary", "error", err.Error())
	}

	// Log final results
	bm.Log(slog.LevelInfo, "Backup completed", 
		"success_count", summary.SuccessCount,
		"failure_count", summary.FailureCount,
		"duration", summary.Duration.String(),
		"success_rate", fmt.Sprintf("%.1f%%", float64(summary.SuccessCount)/float64(len(summary.Results))*100))

	return nil
}

func (bm *BackupManager) loadRepositoriesFromFile(filename string) ([]RepositoryConfig, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read repositories file: %w", err)
	}

	var repositories []RepositoryConfig
	lines := strings.Split(string(content), "\n")

	for i, line := range lines {
		line = strings.TrimSpace(line)
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Extract repository name from URL
		repoName, err := bm.extractRepoNameFromURL(line)
		if err != nil {
			return nil, fmt.Errorf("invalid repository URL on line %d: %w", i+1, err)
		}

		repositories = append(repositories, RepositoryConfig{
			Name: repoName,
			URL:  line,
		})
	}

	if len(repositories) == 0 {
		return nil, fmt.Errorf("no valid repositories found in %s", filename)
	}

	return repositories, nil
}

func (bm *BackupManager) extractRepoNameFromURL(url string) (string, error) {
	url = strings.TrimSpace(url)
	
	// Remove .git suffix if present
	if strings.HasSuffix(url, ".git") {
		url = url[:len(url)-4]
	}
	
	// Handle HTTPS URLs
	if strings.HasPrefix(url, "https://") {
		parts := strings.Split(url, "/")
		if len(parts) >= 5 {
			return parts[4], nil
		}
	}
	
	// Handle SSH URLs
	if strings.HasPrefix(url, "git@") {
		parts := strings.Split(url, ":")
		if len(parts) >= 2 {
			repoPart := parts[1]
			if strings.HasSuffix(repoPart, ".git") {
				repoPart = repoPart[:len(repoPart)-4]
			}
			repoParts := strings.Split(repoPart, "/")
			if len(repoParts) >= 2 {
				return repoParts[1], nil
			}
		}
	}
	
	return "", fmt.Errorf("unable to extract repository name from URL: %s", url)
}

func (bm *BackupManager) createBackupDirectories(repositories []RepositoryConfig) error {
	for _, repo := range repositories {
		backupPath := filepath.Join("backups", repo.Name, bm.backupDate)
		
		// Remove existing backup for today if it exists
		if err := os.RemoveAll(backupPath); err != nil {
			bm.Log(slog.LevelWarn, "Could not remove existing backup directory", 
				"path", backupPath, "error", err.Error())
		}

		// Create fresh backup directory
		if err := os.MkdirAll(backupPath, 0755); err != nil {
			return fmt.Errorf("failed to create backup directory %s: %w", backupPath, err)
		}

		bm.Log(slog.LevelInfo, "Created backup directory", "path", backupPath)
	}
	return nil
}

func (bm *BackupManager) performBackups(repositories []RepositoryConfig) BackupSummary {
	summary := BackupSummary{
		Date:      bm.backupDate,
		StartTime: time.Now(),
		Results:   []BackupResult{},
	}

	for _, repo := range repositories {
		bm.Log(slog.LevelInfo, "Starting backup", "repository", repo.Name)
		
		result := bm.backupRepository(repo)
		summary.Results = append(summary.Results, result)

		if result.Success {
			summary.SuccessCount++
			bm.Log(slog.LevelInfo, "Repository backup completed", 
				"repository", repo.Name, 
				"size", result.Size,
				"zip_size", result.ZipSize,
				"duration", result.Duration.String())
		} else {
			summary.FailureCount++
			bm.Log(slog.LevelError, "Repository backup failed", 
				"repository", repo.Name, 
				"error", result.Error,
				"duration", result.Duration.String())
		}
	}

	summary.EndTime = time.Now()
	summary.Duration = summary.EndTime.Sub(summary.StartTime)

	return summary
}

func (bm *BackupManager) backupRepository(repo RepositoryConfig) BackupResult {
	result := BackupResult{
		Name:      repo.Name,
		StartTime: time.Now(),
	}

	// Clone repository with retry
	backupPath := filepath.Join("backups", repo.Name, bm.backupDate, repo.Name+".git")
	zipPath := backupPath + ".zip"

	// Construct authenticated URL
	repoURL := bm.constructAuthenticatedURL(repo.URL)

	// Retry logic
	maxRetries := 3
	var lastError error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if err := bm.cloneRepository(repoURL, backupPath); err != nil {
			lastError = err
			if attempt < maxRetries {
				bm.Log(slog.LevelWarn, "Git clone failed, retrying", 
					"repository", repo.Name, 
					"attempt", attempt, 
					"max_attempts", maxRetries, 
					"error", err.Error())
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}
		} else {
			result.Success = true
			break
		}
	}

	if !result.Success {
		result.Error = lastError.Error()
	} else {
		// Remove credentials from config
		if err := bm.removeCredentialsFromConfig(backupPath); err != nil {
			bm.Log(slog.LevelWarn, "Could not remove credentials from config", 
				"repository", repo.Name, "error", err.Error())
		}

		// Calculate size
		if size, err := bm.getDirectorySize(backupPath); err == nil {
			result.Size = size
		}

		// Compress to zip
		if err := bm.zipDirectory(backupPath, zipPath); err == nil {
			if zipSize, err := bm.getFileSize(zipPath); err == nil {
				result.ZipSize = zipSize
			}
			// Delete the uncompressed .git folder
			os.RemoveAll(backupPath)
		} else {
			bm.Log(slog.LevelWarn, "Could not compress backup", 
				"repository", repo.Name, "error", err.Error())
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result
}

func (bm *BackupManager) constructAuthenticatedURL(url string) string {
	if strings.HasPrefix(url, "https://") {
		return strings.Replace(url, "https://", fmt.Sprintf("https://%s@", bm.backupToken), 1)
	} else if strings.HasPrefix(url, "git@") {
		return strings.Replace(url, "git@github.com:", fmt.Sprintf("https://%s@github.com/", bm.backupToken), 1)
	}
	return url
}

func (bm *BackupManager) cloneRepository(repoURL, backupPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "clone", "--mirror", repoURL, backupPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone failed: %w, output: %s", err, string(output))
	}
	return nil
}

func (bm *BackupManager) removeCredentialsFromConfig(gitPath string) error {
	configPath := filepath.Join(gitPath, "config")
	
	content, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	
	newContent := strings.ReplaceAll(string(content), 
		fmt.Sprintf("https://%s@github.com", bm.backupToken), 
		"https://github.com")
	
	return os.WriteFile(configPath, []byte(newContent), 0644)
}

func (bm *BackupManager) getDirectorySize(path string) (string, error) {
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

func (bm *BackupManager) createBackupSummary(summary BackupSummary) error {
	summaryDir := filepath.Join("backups", "summary", bm.backupDate)
	if err := os.MkdirAll(summaryDir, 0755); err != nil {
		return fmt.Errorf("failed to create summary directory: %w", err)
	}

	// Create markdown summary
	summaryPath := filepath.Join(summaryDir, "README.md")
	if err := bm.createMarkdownSummary(summaryPath, summary); err != nil {
		return fmt.Errorf("failed to create markdown summary: %w", err)
	}

	// Create JSON summary
	jsonPath := filepath.Join(summaryDir, "summary.json")
	if err := bm.createJSONSummary(jsonPath, summary); err != nil {
		return fmt.Errorf("failed to create JSON summary: %w", err)
	}

	bm.Log(slog.LevelInfo, "Created backup summaries", 
		"markdown_path", summaryPath, 
		"json_path", jsonPath)

	return nil
}

func (bm *BackupManager) createMarkdownSummary(path string, summary BackupSummary) error {
	var content strings.Builder
	content.WriteString(fmt.Sprintf("# Backup Summary - %s\n\n", summary.Date))
	content.WriteString(fmt.Sprintf("**Start Time:** %s\n", summary.StartTime.Format("2006-01-02 15:04:05")))
	content.WriteString(fmt.Sprintf("**End Time:** %s\n", summary.EndTime.Format("2006-01-02 15:04:05")))
	content.WriteString(fmt.Sprintf("**Duration:** %s\n", summary.Duration.String()))
	
	// Calculate success rate safely
	successRate := 0.0
	if len(summary.Results) > 0 {
		successRate = float64(summary.SuccessCount) / float64(len(summary.Results)) * 100
	}
	content.WriteString(fmt.Sprintf("**Success Rate:** %d/%d (%.1f%%)\n\n", 
		summary.SuccessCount, len(summary.Results), successRate))

	content.WriteString("## Backup Results:\n")
	for _, result := range summary.Results {
		if result.Success {
			content.WriteString(fmt.Sprintf("- ✅ **%s**: %s", result.Name, result.Size))
			if result.ZipSize != "" {
				content.WriteString(fmt.Sprintf(" (zip: %s)", result.ZipSize))
			}
			content.WriteString(fmt.Sprintf(" - %s\n", result.Duration.String()))
		} else {
			content.WriteString(fmt.Sprintf("- ❌ **%s**: %s\n", result.Name, result.Error))
		}
	}

	content.WriteString("\n## Backup Log:\n")
	if logContent, err := os.ReadFile("backup-log.txt"); err == nil {
		content.WriteString("```\n")
		content.WriteString(string(logContent))
		content.WriteString("\n```\n")
	}

	return os.WriteFile(path, []byte(content.String()), 0644)
}

func (bm *BackupManager) createJSONSummary(path string, summary BackupSummary) error {
	jsonData, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal summary: %w", err)
	}
	return os.WriteFile(path, jsonData, 0644)
}

// enforceRetention keeps only the 5 most recent backup folders per repo, deleting the oldest.
func (bm *BackupManager) enforceRetention(repositories []RepositoryConfig) {
	const maxBackups = 5
	for _, repo := range repositories {
		repoPath := filepath.Join("backups", repo.Name)
		entries, err := os.ReadDir(repoPath)
		if err != nil {
			bm.Log(slog.LevelWarn, "Could not read repository directory", 
				"repository", repo.Name, "error", err.Error())
			continue
		}
		
		var backupDirs []string
		for _, entry := range entries {
			if entry.IsDir() && bm.isDateDir(entry.Name()) {
				backupDirs = append(backupDirs, entry.Name())
			}
		}
		
		if len(backupDirs) <= maxBackups {
			continue
		}
		
		// Sort oldest to newest
		sort.Strings(backupDirs)
		toDelete := backupDirs[:len(backupDirs)-maxBackups]
		
		for _, dir := range toDelete {
			dirPath := filepath.Join(repoPath, dir)
			if err := os.RemoveAll(dirPath); err != nil {
				bm.Log(slog.LevelWarn, "Could not remove old backup directory", 
					"path", dirPath, "error", err.Error())
			} else {
				bm.Log(slog.LevelInfo, "Removed old backup directory", "path", dirPath)
			}
		}
	}
}

// isDateDir returns true if the directory name looks like a date (YYYY-MM-DD)
func (bm *BackupManager) isDateDir(name string) bool {
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

// zipDirectory compresses the given directory into a zip file at zipPath
func (bm *BackupManager) zipDirectory(srcDir, zipPath string) error {
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
				return nil // don't add the root dir itself
			}
			// Add directory entry
			header := &zip.FileHeader{
				Name:     relPath + "/",
				Method:   zip.Deflate,
				Modified: info.ModTime(),
			}
			_, err := zipWriter.CreateHeader(header)
			return err
		}
		// Add file entry
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

// getFileSize returns the size of the file at path as a human-readable string
func (bm *BackupManager) getFileSize(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	return bm.byteCountDecimal(info.Size()), nil
}

// byteCountDecimal returns a human-readable file size
func (bm *BackupManager) byteCountDecimal(b int64) string {
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

// getColorForLevel returns the appropriate color for a log level
func getColorForLevel(level slog.Level) string {
	switch level {
	case slog.LevelError:
		return "Attention"
	case slog.LevelWarn:
		return "Warning"
	case slog.LevelInfo:
		return "Good"
	default:
		return "Default"
	}
}