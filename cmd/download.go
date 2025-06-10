package cmd

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/davidroman0O/steam-workshop-downloader/pkg/scraper"
	"github.com/davidroman0O/steam-workshop-downloader/pkg/steamcmd"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// downloadCmd represents the download command
var downloadCmd = &cobra.Command{
	Use:   "download [URL or ID]",
	Short: "Download Steam Workshop items",
	Long: `Download Steam Workshop items using various input formats:

Supported formats:
- Workshop URL: https://steamcommunity.com/sharedfiles/filedetails/?id=123456789
- Direct ID: 123456789 (requires --app-id)
- App ID + Workshop ID: 431960 123456789

Examples:
  workshop download https://steamcommunity.com/sharedfiles/filedetails/?id=2503622437
  workshop download 2503622437 --app-id 108600
  workshop download 108600 2503622437`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return downloadWorkshopItem(args)
	},
}

func init() {
	rootCmd.AddCommand(downloadCmd)

	downloadCmd.Flags().StringP("app-id", "a", "", "Steam App ID (required if not providing URL)")
	downloadCmd.Flags().StringP("username", "u", "", "Steam username for private items")
	downloadCmd.Flags().StringP("password", "p", "", "Steam password for private items")
	downloadCmd.Flags().BoolP("extract", "e", true, "Extract downloaded files to output directory")
	downloadCmd.Flags().StringP("output", "o", "", "Output directory (default: configured download directory)")

	viper.BindPFlag("app_id", downloadCmd.Flags().Lookup("app-id"))
	viper.BindPFlag("username", downloadCmd.Flags().Lookup("username"))
	viper.BindPFlag("password", downloadCmd.Flags().Lookup("password"))
	viper.BindPFlag("extract", downloadCmd.Flags().Lookup("extract"))
	viper.BindPFlag("output", downloadCmd.Flags().Lookup("output"))
}

func downloadWorkshopItem(args []string) error {
	// Parse input to extract app ID and workshop ID
	appID, workshopID, itemInfo, err := parseDownloadInput(args)
	if err != nil {
		return fmt.Errorf("invalid input: %w", err)
	}

	// Show what we're downloading
	if itemInfo != nil && itemInfo.Title != "" {
		fmt.Printf("Found: %s\n", itemInfo.Title)
		if itemInfo.GameName != "" {
			fmt.Printf("Game: %s\n", itemInfo.GameName)
		}
	}

	// Create SteamCMD client
	steamcmdDir := viper.GetString("steamcmd_dir")
	client, err := steamcmd.NewClient(steamcmdDir)
	if err != nil {
		return fmt.Errorf("failed to create SteamCMD client: %w", err)
	}

	fmt.Printf("Downloading workshop item %s for app %s...\n", workshopID, appID)

	// Download the workshop item
	var item *steamcmd.WorkshopItem
	username := viper.GetString("username")
	password := viper.GetString("password")

	if username != "" && password != "" {
		fmt.Println("Using Steam credentials for download...")
		item, err = client.DownloadWorkshopItemWithAuth(appID, workshopID, username, password)
	} else {
		fmt.Println("Using anonymous download...")
		item, err = client.DownloadWorkshopItem(appID, workshopID)
	}

	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	if !item.Success {
		return fmt.Errorf("download unsuccessful: %s", item.ErrorMsg)
	}

	fmt.Printf("Successfully downloaded to: %s\n", item.PathToFile)
	fmt.Printf("Size: %s\n", formatBytes(item.SizeBytes))

	// Handle extraction/copying if requested
	outputDir := viper.GetString("output")
	extract := viper.GetBool("extract")

	if extract && outputDir != "" {
		if err := handleOutput(item, outputDir, appID, workshopID); err != nil {
			fmt.Printf("Warning: Failed to handle output: %v\n", err)
		}
	}

	return nil
}

func parseDownloadInput(args []string) (appID, workshopID string, itemInfo *scraper.WorkshopInfo, err error) {
	if len(args) == 0 {
		return "", "", nil, fmt.Errorf("no input provided")
	}

	// Case 1: Two arguments (app ID and workshop ID)
	if len(args) == 2 {
		appID = args[0]
		workshopID = args[1]

		// Validate both are numeric
		if !isNumeric(appID) || !isNumeric(workshopID) {
			return "", "", nil, fmt.Errorf("both app ID and workshop ID must be numeric")
		}

		return appID, workshopID, nil, nil
	}

	// Case 2: Single argument (URL or workshop ID)
	input := args[0]

	// Try to parse as URL
	if strings.HasPrefix(input, "http") {
		parsedURL, err := url.Parse(input)
		if err != nil {
			return "", "", nil, fmt.Errorf("invalid URL: %w", err)
		}

		if parsedURL.Host != "steamcommunity.com" {
			return "", "", nil, fmt.Errorf("unsupported URL host: %s", parsedURL.Host)
		}

		fmt.Println("Extracting information from workshop page...")

		// Use scraper to get App ID and other info
		itemInfo, err := scraper.ScrapeWorkshopPage(input)
		if err != nil {
			return "", "", nil, fmt.Errorf("failed to scrape workshop page: %w", err)
		}

		return itemInfo.AppID, itemInfo.WorkshopID, itemInfo, nil
	}

	// Case 3: Single numeric input (workshop ID only)
	if isNumeric(input) {
		workshopID = input
		appID = viper.GetString("app_id")

		if appID == "" {
			return "", "", nil, fmt.Errorf("app ID is required when providing only workshop ID. Use --app-id flag or provide both app ID and workshop ID")
		}

		return appID, workshopID, nil, nil
	}

	return "", "", nil, fmt.Errorf("invalid input format")
}

func isNumeric(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func handleOutput(item *steamcmd.WorkshopItem, outputDir, appID, workshopID string) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create a structured directory for the workshop item
	itemOutputDir := filepath.Join(outputDir, fmt.Sprintf("app_%s_workshop_%s", appID, workshopID))
	if err := os.MkdirAll(itemOutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create item output directory: %w", err)
	}

	// Copy the workshop item directory to the output location
	if err := copyDirectory(item.PathToFile, itemOutputDir); err != nil {
		return fmt.Errorf("failed to copy workshop item: %w", err)
	}

	fmt.Printf("Workshop item extracted to: %s\n", itemOutputDir)
	return nil
}

// copyDirectory recursively copies a directory from src to dst
func copyDirectory(src, dst string) error {
	// Get the source directory info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Create the destination directory
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	// Read the source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	// Copy each entry
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectories
			if err := copyDirectory(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy files
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file from src to dst
func copyFile(src, dst string) error {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Get source file info
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	// Create destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Copy the file contents
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	// Set the file permissions to match the source
	if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
		return err
	}

	return nil
}

// Additional helper functions for URL parsing and validation
func parseWorkshopURL(rawURL string) (workshopID string, err error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	// Check for steamcommunity.com host
	if !strings.Contains(parsedURL.Host, "steamcommunity.com") {
		return "", fmt.Errorf("not a Steam Community URL")
	}

	// Extract ID from query parameters
	workshopID = parsedURL.Query().Get("id")
	if workshopID == "" {
		return "", fmt.Errorf("no workshop ID found in URL")
	}

	// Validate that ID is numeric
	if !isNumeric(workshopID) {
		return "", fmt.Errorf("workshop ID must be numeric")
	}

	return workshopID, nil
}

// ValidateWorkshopID validates a workshop ID
func ValidateWorkshopID(id string) error {
	if id == "" {
		return fmt.Errorf("workshop ID cannot be empty")
	}

	// Check if numeric
	if !isNumeric(id) {
		return fmt.Errorf("workshop ID must be numeric")
	}

	// Check reasonable length (Steam IDs are typically 8-10 digits)
	if len(id) < 1 || len(id) > 20 {
		return fmt.Errorf("workshop ID has invalid length")
	}

	return nil
}

// ValidateAppID validates a Steam app ID
func ValidateAppID(id string) error {
	if id == "" {
		return fmt.Errorf("app ID cannot be empty")
	}

	// Check if numeric
	if !isNumeric(id) {
		return fmt.Errorf("app ID must be numeric")
	}

	// Check reasonable length
	if len(id) < 1 || len(id) > 10 {
		return fmt.Errorf("app ID has invalid length")
	}

	return nil
}
