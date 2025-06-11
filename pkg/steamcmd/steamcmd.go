package steamcmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"bufio"

	"github.com/sethvargo/go-retry"
)

// Client represents a SteamCMD client
type Client struct {
	SteamCMDPath string
	WorkingDir   string
}

// WorkshopItem represents a downloaded workshop item
type WorkshopItem struct {
	AppID      string
	WorkshopID string
	Success    bool
	PathToFile string
	SizeBytes  int64
	ErrorMsg   string
}

// NewClient creates a new SteamCMD client
func NewClient(steamcmdDir string) (*Client, error) {
	var steamcmdExe string
	if runtime.GOOS == "windows" {
		steamcmdExe = filepath.Join(steamcmdDir, "steamcmd.exe")
	} else {
		steamcmdExe = filepath.Join(steamcmdDir, "steamcmd.sh")
	}

	// Check if SteamCMD exists
	if _, err := os.Stat(steamcmdExe); os.IsNotExist(err) {
		return nil, fmt.Errorf("SteamCMD not found at %s. Run 'install' command first", steamcmdExe)
	}

	return &Client{
		SteamCMDPath: steamcmdExe,
		WorkingDir:   steamcmdDir,
	}, nil
}

// DownloadWorkshopItem downloads a workshop item using SteamCMD with retry logic
// Uses provided username with cached credentials, falls back to anonymous
func (c *Client) DownloadWorkshopItem(appID, workshopID, username string) (*WorkshopItem, error) {
	item := &WorkshopItem{
		AppID:      appID,
		WorkshopID: workshopID,
	}

	// Create a context for the retry operation
	ctx := context.Background()

	// Setup Fibonacci backoff: start at 10s, max 4 retries (5 total attempts)
	var maxRetries uint64 = 10
	backoff := retry.NewFibonacci(2 * time.Second)
	backoff = retry.WithMaxRetries(maxRetries, backoff)

	var attemptCount int
	err := retry.Do(ctx, backoff, func(ctx context.Context) error {
		attemptCount++
		if attemptCount > 1 {
			fmt.Printf("Retry attempt %d/%d...\n", attemptCount-1, maxRetries)
		}

		var args []string
		if username != "" {
			// Use provided username with cached credentials
			args = []string{
				"+@ShutdownOnFailedCommand", "1", // Exit on command failure
				"+login", username, // Use cached credentials for this user
				"+workshop_download_item", appID, workshopID,
				"+quit",
			}
		} else {
			// No username provided, try anonymous
			args = []string{
				"+@ShutdownOnFailedCommand", "1", // Exit on command failure
				"+login", "anonymous",
				"+workshop_download_item", appID, workshopID,
				"+quit",
			}
		}

		// Execute SteamCMD
		cmd := exec.Command(c.SteamCMDPath, args...)
		cmd.Dir = c.WorkingDir

		var outputBuf bytes.Buffer
		cmd.Stdout = &outputBuf
		cmd.Stderr = &outputBuf

		if err := cmd.Run(); err != nil {
			// Read the default SteamCMD console log for more details
			consoleLogPath := filepath.Join(c.WorkingDir, "logs", "console_log.txt")
			logContent := c.readLogFile(consoleLogPath)
			if logContent != "" && attemptCount == 1 {
				fmt.Printf("Recent log entries:\n%s\n", c.getRecentLogLines(logContent))
			}

			// Check for authentication issues
			if strings.Contains(logContent, "Not logged on") {
				return fmt.Errorf("not logged on to Steam. Please run 'workshop login' first to authenticate")
			}

			// Make the error retryable to trigger backoff
			return retry.RetryableError(fmt.Errorf("failed to run SteamCMD: %w\nOutput: %s", err, outputBuf.String()))
		}

		// Parse the output to determine success/failure
		if err := c.parseOutput(&outputBuf, item); err != nil {
			// Check if this is a retryable error based on the item result
			if !item.Success && c.isRetryableError(item.ErrorMsg) {
				return retry.RetryableError(fmt.Errorf("SteamCMD download failed: %s", item.ErrorMsg))
			}
			// Non-retryable error (e.g., invalid workshop ID, parsing issue)
			return fmt.Errorf("failed to parse SteamCMD output: %w", err)
		}

		// Check if download was successful
		if !item.Success {
			if c.isRetryableError(item.ErrorMsg) {
				return retry.RetryableError(fmt.Errorf("download failed: %s", item.ErrorMsg))
			}
			// Non-retryable error
			return fmt.Errorf("download failed: %s", item.ErrorMsg)
		}

		return nil
	})

	if err != nil {
		return item, err
	}

	return item, nil
}

// DownloadWorkshopItemWithAuth downloads a workshop item using Steam credentials with retry logic
func (c *Client) DownloadWorkshopItemWithAuth(appID, workshopID, username, password, guardCode string) (*WorkshopItem, error) {
	item := &WorkshopItem{
		AppID:      appID,
		WorkshopID: workshopID,
	}

	// Create a context for the retry operation
	ctx := context.Background()

	// Setup Fibonacci backoff: start at 10s, max 4 retries (5 total attempts)
	var maxRetries uint64 = 10
	backoff := retry.NewFibonacci(2 * time.Second)
	backoff = retry.WithMaxRetries(maxRetries, backoff)

	var attemptCount int
	err := retry.Do(ctx, backoff, func(ctx context.Context) error {
		attemptCount++
		if attemptCount > 1 {
			fmt.Printf("Retry attempt %d/%d...\n", attemptCount-1, maxRetries)
		}

		// Build SteamCMD arguments with authentication
		args := []string{
			"+@ShutdownOnFailedCommand", "1", // Exit on command failure
			"+@NoPromptForPassword", "1", // Don't prompt for passwords
			"+login", username, password,
		}

		// Add Steam Guard code if provided
		if guardCode != "" {
			args = append(args, "+set_steam_guard_code", guardCode)
		}

		// Add download command
		args = append(args, "+workshop_download_item", appID, workshopID, "+quit")

		// Execute SteamCMD
		cmd := exec.Command(c.SteamCMDPath, args...)
		cmd.Dir = c.WorkingDir

		var outputBuf bytes.Buffer
		cmd.Stdout = &outputBuf
		cmd.Stderr = &outputBuf

		if err := cmd.Run(); err != nil {
			// Read the default SteamCMD console log for more details
			consoleLogPath := filepath.Join(c.WorkingDir, "logs", "console_log.txt")
			logContent := c.readLogFile(consoleLogPath)
			fmt.Printf("SteamCMD failed, check console log: %s\n", consoleLogPath)
			if logContent != "" {
				fmt.Printf("Recent log entries:\n%s\n", c.getRecentLogLines(logContent))
			}
			// Check if this is a Steam Guard error
			if strings.Contains(logContent, "steam_guard_code") || strings.Contains(logContent, "Account Logon Denied") {
				if guardCode == "" {
					return fmt.Errorf("Steam Guard authentication required. Please provide --guard-code flag with the code from your email")
				}
			}
			// Check for authentication issues
			if strings.Contains(logContent, "Not logged on") {
				return fmt.Errorf("not logged on to Steam. Please run 'workshop login' first to authenticate")
			}
			// Make the error retryable to trigger backoff
			return retry.RetryableError(fmt.Errorf("failed to run SteamCMD: %w\nOutput: %s", err, outputBuf.String()))
		}

		// Parse the output to determine success/failure
		if err := c.parseOutput(&outputBuf, item); err != nil {
			// Check if this is a retryable error based on the item result
			if !item.Success && c.isRetryableError(item.ErrorMsg) {
				consoleLogPath := filepath.Join(c.WorkingDir, "logs", "console_log.txt")
				logContent := c.readLogFile(consoleLogPath)
				if logContent != "" {
					fmt.Printf("Download failed, recent log entries:\n%s\n", c.getRecentLogLines(logContent))
				}
				return retry.RetryableError(fmt.Errorf("SteamCMD download failed: %s", item.ErrorMsg))
			}
			// Non-retryable error (e.g., invalid workshop ID, parsing issue)
			return fmt.Errorf("failed to parse SteamCMD output: %w", err)
		}

		// Check if download was successful
		if !item.Success {
			if c.isRetryableError(item.ErrorMsg) {
				consoleLogPath := filepath.Join(c.WorkingDir, "logs", "console_log.txt")
				logContent := c.readLogFile(consoleLogPath)
				if logContent != "" {
					fmt.Printf("Download failed, recent log entries:\n%s\n", c.getRecentLogLines(logContent))
				}
				return retry.RetryableError(fmt.Errorf("download failed: %s", item.ErrorMsg))
			}
			// Non-retryable error
			return fmt.Errorf("download failed: %s", item.ErrorMsg)
		}

		return nil
	})

	if err != nil {
		return item, err
	}

	return item, nil
}

// isRetryableError determines if an error should trigger a retry
func (c *Client) isRetryableError(errorMsg string) bool {
	// Define retryable error patterns (network issues, temporary Steam server problems)
	retryablePatterns := []string{
		"timeout",
		"connection",
		"network",
		"server",
		"unavailable",
		"busy",
		"rate limit",
		"throttle",
		"no connection",
		"steam servers",
		"failure",    // Generic SteamCMD failure
		"failed",     // Generic failures
		"error",      // Generic errors
		"temporary",  // Temporary issues
		"retry",      // Explicit retry suggestions
		"please try", // Steam's "please try again" messages
	}

	errorLower := strings.ToLower(errorMsg)
	for _, pattern := range retryablePatterns {
		if strings.Contains(errorLower, pattern) {
			return true
		}
	}

	// Non-retryable errors (invalid workshop ID, authentication issues, etc.)
	return false
}

// parseOutput parses SteamCMD output to determine download status
func (c *Client) parseOutput(outputBuf *bytes.Buffer, item *WorkshopItem) error {
	output := outputBuf.String()

	// Define regular expressions for different scenarios
	successRegex := regexp.MustCompile(`Success\. Downloaded item (\d+) to "([^"]+)" \((\d+) bytes\)`)
	downloadFailureRegex := regexp.MustCompile(`ERROR! Download item (\d+) failed \(([^)]+)\)`)
	loginFailureRegex := regexp.MustCompile(`FAILED \(([^)]+)\)`)

	// Check for success case
	if matches := successRegex.FindStringSubmatch(output); matches != nil {
		item.Success = true
		item.PathToFile = matches[2]

		if bytes, err := strconv.ParseInt(matches[3], 10, 64); err == nil {
			item.SizeBytes = bytes
		}

		return nil
	}

	// Check for download failure
	if matches := downloadFailureRegex.FindStringSubmatch(output); matches != nil {
		item.Success = false
		item.ErrorMsg = fmt.Sprintf("Download failed: %s", matches[2])
		return nil
	}

	// Check for login failure
	if matches := loginFailureRegex.FindStringSubmatch(output); matches != nil {
		item.Success = false
		item.ErrorMsg = fmt.Sprintf("Login failed: %s", matches[1])
		return nil
	}

	// If we can't parse the output, it's an unknown error
	item.Success = false
	item.ErrorMsg = "Unknown error occurred"

	return fmt.Errorf("unhandled SteamCMD output: %s", output)
}

// TestConnection tests if SteamCMD can connect to Steam
func (c *Client) TestConnection() error {
	args := []string{"+login", "anonymous", "+quit"}

	cmd := exec.Command(c.SteamCMDPath, args...)
	cmd.Dir = c.WorkingDir

	var outputBuf bytes.Buffer
	cmd.Stdout = &outputBuf
	cmd.Stderr = &outputBuf

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("SteamCMD connection test failed: %w\nOutput: %s", err, outputBuf.String())
	}

	output := outputBuf.String()

	// Check for successful login
	if strings.Contains(output, "Waiting for user info...OK") {
		return nil
	}

	// Check for specific connection errors
	if strings.Contains(output, "No connection") {
		return fmt.Errorf("no internet connection or Steam servers unreachable")
	}

	return fmt.Errorf("connection test inconclusive: %s", output)
}

// GetWorkshopPath returns the path where workshop content is downloaded
func (c *Client) GetWorkshopPath() string {
	return filepath.Join(c.WorkingDir, "steamapps", "workshop", "content")
}

// GetWorkshopCachePaths returns all paths that should be cleaned to fix CWorkThreadPool errors
func (c *Client) GetWorkshopCachePaths() []string {
	var paths []string

	// Local steamcmd workshop directories
	localWorkshopBase := filepath.Join(c.WorkingDir, "steamapps", "workshop")
	if _, err := os.Stat(localWorkshopBase); err == nil {
		paths = append(paths,
			filepath.Join(localWorkshopBase, "downloads"),
			filepath.Join(localWorkshopBase, "temp"),
			filepath.Join(localWorkshopBase, "content"),
		)
	}

	// System Steam workshop directories (where content often actually goes)
	homeDir, err := os.UserHomeDir()
	if err == nil {
		var systemSteamBase string
		switch runtime.GOOS {
		case "darwin":
			systemSteamBase = filepath.Join(homeDir, "Library", "Application Support", "Steam", "steamapps", "workshop")
		case "windows":
			systemSteamBase = filepath.Join(homeDir, "AppData", "Local", "Steam", "steamapps", "workshop")
		case "linux":
			systemSteamBase = filepath.Join(homeDir, ".steam", "steam", "steamapps", "workshop")
		}

		if systemSteamBase != "" {
			if _, err := os.Stat(systemSteamBase); err == nil {
				paths = append(paths,
					filepath.Join(systemSteamBase, "downloads"),
					filepath.Join(systemSteamBase, "temp"),
				)
			}
		}
	}

	return paths
}

// CheckWorkshopItemExists checks if a workshop item is already downloaded
func (c *Client) CheckWorkshopItemExists(appID, workshopID string) (bool, string, error) {
	// Check both local steamcmd path and system Steam path
	possiblePaths := []string{
		// Local steamcmd path
		filepath.Join(c.WorkingDir, "steamapps", "workshop", "content", appID, workshopID),
	}

	// System Steam path
	homeDir, err := os.UserHomeDir()
	if err == nil {
		var systemSteamBase string
		switch runtime.GOOS {
		case "darwin":
			systemSteamBase = filepath.Join(homeDir, "Library", "Application Support", "Steam", "steamapps", "workshop", "content")
		case "windows":
			systemSteamBase = filepath.Join(homeDir, "AppData", "Local", "Steam", "steamapps", "workshop", "content")
		case "linux":
			systemSteamBase = filepath.Join(homeDir, ".steam", "steam", "steamapps", "workshop", "content")
		}

		if systemSteamBase != "" {
			possiblePaths = append(possiblePaths, filepath.Join(systemSteamBase, appID, workshopID))
		}
	}

	// Check each possible path
	for _, path := range possiblePaths {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return true, path, nil
		}
	}

	return false, "", nil
}

// ListDownloadedItems lists all downloaded workshop items
func (c *Client) ListDownloadedItems() (map[string][]string, error) {
	workshopPath := c.GetWorkshopPath()
	items := make(map[string][]string)

	if _, err := os.Stat(workshopPath); os.IsNotExist(err) {
		return items, nil // No workshop content downloaded yet
	}

	// List app directories
	appDirs, err := os.ReadDir(workshopPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read workshop directory: %w", err)
	}

	for _, appDir := range appDirs {
		if !appDir.IsDir() {
			continue
		}

		appID := appDir.Name()
		appPath := filepath.Join(workshopPath, appID)

		// List workshop items for this app
		itemDirs, err := os.ReadDir(appPath)
		if err != nil {
			continue // Skip if we can't read this app directory
		}

		var itemIDs []string
		for _, itemDir := range itemDirs {
			if itemDir.IsDir() {
				itemIDs = append(itemIDs, itemDir.Name())
			}
		}

		if len(itemIDs) > 0 {
			items[appID] = itemIDs
		}
	}

	return items, nil
}

// readLogFile reads the content of a log file
func (c *Client) readLogFile(logFile string) string {
	content, err := os.ReadFile(logFile)
	if err != nil {
		return ""
	}
	return string(content)
}

// getRecentLogLines returns the most recent lines from the log content
func (c *Client) getRecentLogLines(logContent string) string {
	lines := strings.Split(logContent, "\n")
	if len(lines) > 5 {
		return strings.Join(lines[len(lines)-5:], "\n")
	}
	return logContent
}

// GetDebugCommand returns the exact SteamCMD command that would be executed for debugging
func (c *Client) GetDebugCommand(appID, workshopID string) string {
	args := []string{
		c.SteamCMDPath,
		"+@ShutdownOnFailedCommand", "1",
		"+workshop_download_item", appID, workshopID,
		"+quit",
	}
	return strings.Join(args, " ")
}

// GetDebugCommandWithAuth returns the exact SteamCMD command with auth for debugging
func (c *Client) GetDebugCommandWithAuth(appID, workshopID, username, password string) string {
	args := []string{
		c.SteamCMDPath,
		"+@ShutdownOnFailedCommand", "1",
		"+@NoPromptForPassword", "1",
		"+login", username, "****", // Hide password in debug output
		"+workshop_download_item", appID, workshopID,
		"+quit",
	}
	return strings.Join(args, " ")
}

// InteractiveLogin logs into Steam interactively, handling Steam Guard codes
func (c *Client) InteractiveLogin(username, password string) error {
	fmt.Println("Starting Steam login process...")

	// Build SteamCMD arguments for login
	args := []string{
		"+@ShutdownOnFailedCommand", "0", // Don't exit on failed commands
		"+@NoPromptForPassword", "1", // Don't prompt for passwords
		"+login", username, password,
		"+quit",
	}

	// Execute SteamCMD
	cmd := exec.Command(c.SteamCMDPath, args...)
	cmd.Dir = c.WorkingDir

	var outputBuf bytes.Buffer
	cmd.Stdout = &outputBuf
	cmd.Stderr = &outputBuf

	err := cmd.Run()
	output := outputBuf.String()

	// Check if Steam Guard is required
	if strings.Contains(output, "steam_guard_code") || strings.Contains(output, "Please check your email") {
		fmt.Println("📧 Steam Guard authentication required!")
		fmt.Println("Please check your email for the Steam Guard code.")
		fmt.Print("Enter Steam Guard code: ")

		// Read Steam Guard code from user
		reader := bufio.NewReader(os.Stdin)
		guardCode, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read Steam Guard code: %w", err)
		}
		guardCode = strings.TrimSpace(guardCode)

		if guardCode == "" {
			return fmt.Errorf("Steam Guard code is required")
		}

		// Login with Steam Guard code
		fmt.Println("Authenticating with Steam Guard code...")
		args = []string{
			"+@ShutdownOnFailedCommand", "0",
			"+@NoPromptForPassword", "1",
			"+login", username, password,
			"+set_steam_guard_code", guardCode,
			"+quit",
		}

		cmd = exec.Command(c.SteamCMDPath, args...)
		cmd.Dir = c.WorkingDir

		var finalOutputBuf bytes.Buffer
		cmd.Stdout = &finalOutputBuf
		cmd.Stderr = &finalOutputBuf

		err = cmd.Run()
		finalOutput := finalOutputBuf.String()

		// Check for successful login
		if strings.Contains(finalOutput, "Waiting for user info...OK") || strings.Contains(finalOutput, "OK") {
			return nil
		}

		// Check for login errors
		if strings.Contains(finalOutput, "FAILED") || strings.Contains(finalOutput, "Logon Denied") {
			return fmt.Errorf("authentication failed - check your credentials or Steam Guard code")
		}

		return fmt.Errorf("authentication result unclear: %s", finalOutput)
	}

	// Check for successful login without Steam Guard
	if strings.Contains(output, "Waiting for user info...OK") || strings.Contains(output, "OK") {
		return nil
	}

	// Check for login errors
	if strings.Contains(output, "FAILED") || strings.Contains(output, "Logon Denied") {
		return fmt.Errorf("authentication failed - check your credentials")
	}

	if err != nil {
		return fmt.Errorf("SteamCMD execution failed: %w\nOutput: %s", err, output)
	}

	return fmt.Errorf("authentication result unclear: %s", output)
}
