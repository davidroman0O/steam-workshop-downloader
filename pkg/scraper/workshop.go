package scraper

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// WorkshopInfo contains information scraped from a workshop page
type WorkshopInfo struct {
	AppID      string
	WorkshopID string
	Title      string
	GameName   string
}

// ScrapeWorkshopPage extracts App ID and other info from a Steam Workshop URL
func ScrapeWorkshopPage(url string) (*WorkshopInfo, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Make request to workshop page
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch workshop page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("workshop page returned status: %s", resp.Status)
	}

	// Read the page content
	buf := make([]byte, 1024*1024) // Read up to 1MB
	n, err := resp.Body.Read(buf)
	if err != nil && n == 0 {
		return nil, fmt.Errorf("failed to read workshop page content: %w", err)
	}

	content := string(buf[:n])

	// Extract workshop ID from URL
	workshopIDRegex := regexp.MustCompile(`id=(\d+)`)
	workshopIDMatches := workshopIDRegex.FindStringSubmatch(url)
	if len(workshopIDMatches) < 2 {
		return nil, fmt.Errorf("could not extract workshop ID from URL")
	}

	info := &WorkshopInfo{
		WorkshopID: workshopIDMatches[1],
	}

	// Extract App ID from the page content
	// Look for various patterns where App ID appears
	appIDPatterns := []string{
		`"appid"\s*:\s*"?(\d+)"?`,            // JSON format
		`appid=(\d+)`,                        // URL parameter
		`data-appid="(\d+)"`,                 // HTML data attribute
		`/app/(\d+)/`,                        // App URL pattern
		`store\.steampowered\.com/app/(\d+)`, // Store URL
		`steam://nav/games/details/(\d+)`,    // Steam protocol
	}

	for _, pattern := range appIDPatterns {
		regex := regexp.MustCompile(pattern)
		matches := regex.FindStringSubmatch(content)
		if len(matches) > 1 {
			info.AppID = matches[1]
			break
		}
	}

	if info.AppID == "" {
		return nil, fmt.Errorf("could not extract App ID from workshop page")
	}

	// Extract title if possible
	titleRegex := regexp.MustCompile(`<title>([^<]+)</title>`)
	titleMatches := titleRegex.FindStringSubmatch(content)
	if len(titleMatches) > 1 {
		info.Title = strings.TrimSpace(titleMatches[1])
		// Remove "Steam Workshop::" prefix if present
		info.Title = strings.TrimPrefix(info.Title, "Steam Workshop::")
		info.Title = strings.TrimSpace(info.Title)
	}

	// Try to extract game name
	gameNamePatterns := []string{
		`Steam Workshop::\s*([^>]+)`,
		`<h1[^>]*class="apphub_AppName"[^>]*>([^<]+)</h1>`,
		`data-panel="\{\\"appName\\":\\"([^"]+)\\"`,
	}

	for _, pattern := range gameNamePatterns {
		regex := regexp.MustCompile(pattern)
		matches := regex.FindStringSubmatch(content)
		if len(matches) > 1 {
			info.GameName = strings.TrimSpace(matches[1])
			break
		}
	}

	return info, nil
}

// GetAppIDFromWorkshopURL is a convenience function to just get the App ID
func GetAppIDFromWorkshopURL(url string) (string, error) {
	info, err := ScrapeWorkshopPage(url)
	if err != nil {
		return "", err
	}
	return info.AppID, nil
}
