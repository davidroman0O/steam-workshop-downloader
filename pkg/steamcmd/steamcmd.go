package steamcmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
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

// DownloadWorkshopItem downloads a workshop item using SteamCMD
func (c *Client) DownloadWorkshopItem(appID, workshopID string) (*WorkshopItem, error) {
	item := &WorkshopItem{
		AppID:      appID,
		WorkshopID: workshopID,
	}

	// Build SteamCMD arguments
	args := []string{
		"+login", "anonymous",
		"+workshop_download_item", appID, workshopID,
		"+quit",
	}

	// Execute SteamCMD
	cmd := exec.Command(c.SteamCMDPath, args...)
	cmd.Dir = c.WorkingDir

	var outputBuf bytes.Buffer
	cmd.Stdout = &outputBuf
	cmd.Stderr = &outputBuf

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to run SteamCMD: %w\nOutput: %s", err, outputBuf.String())
	}

	// Parse the output to determine success/failure
	if err := c.parseOutput(&outputBuf, item); err != nil {
		return nil, fmt.Errorf("failed to parse SteamCMD output: %w", err)
	}

	return item, nil
}

// DownloadWorkshopItemWithAuth downloads a workshop item using Steam credentials
func (c *Client) DownloadWorkshopItemWithAuth(appID, workshopID, username, password string) (*WorkshopItem, error) {
	item := &WorkshopItem{
		AppID:      appID,
		WorkshopID: workshopID,
	}

	// Build SteamCMD arguments with authentication
	args := []string{
		"+login", username, password,
		"+workshop_download_item", appID, workshopID,
		"+quit",
	}

	// Execute SteamCMD
	cmd := exec.Command(c.SteamCMDPath, args...)
	cmd.Dir = c.WorkingDir

	var outputBuf bytes.Buffer
	cmd.Stdout = &outputBuf
	cmd.Stderr = &outputBuf

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to run SteamCMD: %w\nOutput: %s", err, outputBuf.String())
	}

	// Parse the output to determine success/failure
	if err := c.parseOutput(&outputBuf, item); err != nil {
		return nil, fmt.Errorf("failed to parse SteamCMD output: %w", err)
	}

	return item, nil
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
