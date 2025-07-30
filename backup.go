package main

import (
	"encoding/json"
	"fmt"
	"log"
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

func main() {
	// Get required environment variable
	backupToken := os.Getenv("BACKUP_TOKEN")
	if backupToken == "" {
		log.Fatal("BACKUP_TOKEN environment variable is required")
	}

	// Set backup date
	backupDate := time.Now().Format("2006-01-02")

	// Load repositories from file
	repositories, err := loadRepositoriesFromFile("repositories.txt")
	if err != nil {
		log.Fatalf("Failed to load repositories: %v", err)
	}

	log.Printf("Starting backup process for %d repositories", len(repositories))

	// Create backup directories
	if err := createBackupDirectories(repositories, backupDate); err != nil {
		log.Fatalf("Failed to create backup directories: %v", err)
	}

	// Perform backups
	summary := performBackups(repositories, backupToken, backupDate)

	// Enforce retention: keep only 5 most recent backups per repo
	enforceRetention(repositories)

	// Create summary
	createBackupSummary(summary, backupDate)

	// Log final results
	log.Printf("Backup completed. Success: %d, Failures: %d, Duration: %s", 
		summary.SuccessCount, summary.FailureCount, summary.Duration)
}

func loadRepositoriesFromFile(filename string) ([]RepositoryConfig, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read repositories file: %v", err)
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
		repoName, err := extractRepoNameFromURL(line)
		if err != nil {
			return nil, fmt.Errorf("invalid repository URL on line %d: %v", i+1, err)
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

func extractRepoNameFromURL(url string) (string, error) {
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

func createBackupDirectories(repositories []RepositoryConfig, backupDate string) error {
	for _, repo := range repositories {
		backupPath := filepath.Join("backups", repo.Name, backupDate)
		
		// Remove existing backup for today if it exists
		if err := os.RemoveAll(backupPath); err != nil {
			log.Printf("Warning: Could not remove existing backup directory %s: %v", backupPath, err)
		}

		// Create fresh backup directory
		if err := os.MkdirAll(backupPath, 0755); err != nil {
			return fmt.Errorf("failed to create backup directory %s: %v", backupPath, err)
		}

		log.Printf("Created backup directory: %s", backupPath)
	}
	return nil
}

func performBackups(repositories []RepositoryConfig, backupToken, backupDate string) BackupSummary {
	summary := BackupSummary{
		Date:      backupDate,
		StartTime: time.Now(),
		Results:   []BackupResult{},
	}

	for _, repo := range repositories {
		log.Printf("Starting backup of %s", repo.Name)

		result := BackupResult{
			Name:      repo.Name,
			StartTime: time.Now(),
		}

		// Clone repository with retry
		backupPath := filepath.Join("backups", repo.Name, backupDate, repo.Name+".git")
		zipPath := backupPath + ".zip"

		// Construct authenticated URL
		repoURL := repo.URL
		if strings.HasPrefix(repoURL, "https://") {
			repoURL = strings.Replace(repoURL, "https://", fmt.Sprintf("https://%s@", backupToken), 1)
		} else if strings.HasPrefix(repoURL, "git@") {
			repoURL = strings.Replace(repoURL, "git@github.com:", fmt.Sprintf("https://%s@github.com/", backupToken), 1)
		}

		// Retry logic
		maxRetries := 3
		var lastError error

		for attempt := 1; attempt <= maxRetries; attempt++ {
			if err := cloneRepository(repoURL, backupPath); err != nil {
				lastError = err
				if attempt < maxRetries {
					log.Printf("Git clone failed for %s (attempt %d/%d), retrying: %v", 
						repo.Name, attempt, maxRetries, err)
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
			if err := removeCredentialsFromConfig(backupPath); err != nil {
				log.Printf("Warning: Could not remove credentials from config for %s: %v", repo.Name, err)
			}

			// Calculate size
			if size, err := getDirectorySize(backupPath); err == nil {
				result.Size = size
			}

			// Compress to zip
			if err := zipDirectory(backupPath, zipPath); err == nil {
				if zipSize, err := getFileSize(zipPath); err == nil {
					result.ZipSize = zipSize
				}
				// Delete the uncompressed .git folder
				os.RemoveAll(backupPath)
			} else {
				log.Printf("Warning: Could not compress backup for %s: %v", repo.Name, err)
			}

			// Log success
			logMessage := fmt.Sprintf("✅ %s mirrored and zipped successfully - %s\n", repo.Name, time.Now().Format("2006-01-02 15:04:05"))
			appendToLog("backup-log.txt", logMessage)
		}

		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)

		if result.Success {
			summary.SuccessCount++
			log.Printf("✅ %s backed up and zipped successfully (%s) in %s", repo.Name, result.ZipSize, result.Duration)
		} else {
			summary.FailureCount++
			log.Printf("❌ %s backup failed: %s", repo.Name, result.Error)
		}

		summary.Results = append(summary.Results, result)
	}

	summary.EndTime = time.Now()
	summary.Duration = summary.EndTime.Sub(summary.StartTime)

	return summary
}

func cloneRepository(repoURL, backupPath string) error {
	cmd := exec.Command("git", "clone", "--mirror", repoURL, backupPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone failed: %v, output: %s", err, string(output))
	}
	return nil
}

func removeCredentialsFromConfig(gitPath string) error {
	configPath := filepath.Join(gitPath, "config")
	
	content, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	
	token := os.Getenv("BACKUP_TOKEN")
	newContent := strings.ReplaceAll(string(content), 
		fmt.Sprintf("https://%s@github.com", token), 
		"https://github.com")
	
	return os.WriteFile(configPath, []byte(newContent), 0644)
}

func getDirectorySize(path string) (string, error) {
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

func createBackupSummary(summary BackupSummary, backupDate string) {
	summaryDir := filepath.Join("backups", "summary", backupDate)
	if err := os.MkdirAll(summaryDir, 0755); err != nil {
		log.Printf("Error creating summary directory: %v", err)
		return
	}

	// Create markdown summary
	summaryPath := filepath.Join(summaryDir, "README.md")
	createMarkdownSummary(summaryPath, summary)

	// Create JSON summary
	jsonPath := filepath.Join(summaryDir, "summary.json")
	createJSONSummary(jsonPath, summary)

	log.Printf("Created backup summaries: %s, %s", summaryPath, jsonPath)
}

func createMarkdownSummary(path string, summary BackupSummary) {
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

	os.WriteFile(path, []byte(content.String()), 0644)
}

func createJSONSummary(path string, summary BackupSummary) {
	jsonData, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal summary: %v", err)
		return
	}
	os.WriteFile(path, jsonData, 0644)
}

func appendToLog(logFile, message string) {
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	
	f.WriteString(message)
} 

// enforceRetention keeps only the 5 most recent backup folders per repo, deleting the oldest.
func enforceRetention(repositories []RepositoryConfig) {
	const maxBackups = 5
	for _, repo := range repositories {
		repoPath := filepath.Join("backups", repo.Name)
		entries, err := os.ReadDir(repoPath)
		if err != nil {
			continue // skip if can't read dir
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
		// Sort oldest to newest
		sort.Strings(backupDirs)
		toDelete := backupDirs[:len(backupDirs)-maxBackups]
		for _, dir := range toDelete {
			os.RemoveAll(filepath.Join(repoPath, dir))
		}
	}
}

// isDateDir returns true if the directory name looks like a date (YYYY-MM-DD)
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

// zipDirectory compresses the given directory into a zip file at zipPath
func zipDirectory(srcDir, zipPath string) error {
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
func getFileSize(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	return byteCountDecimal(info.Size()), nil
}

// byteCountDecimal returns a human-readable file size
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