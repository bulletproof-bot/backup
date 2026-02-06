package version

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Version information - set by build flags
var (
	Version   = "dev"
	GitCommit = "none"
	BuildDate = "unknown"
)

// Info returns formatted version information
func Info() string {
	return fmt.Sprintf("bulletproof version %s (commit: %s, built: %s)", Version, GitCommit, BuildDate)
}

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// CheckForUpdate checks if a newer version is available on GitHub
// Returns the latest version and URL if newer, empty strings otherwise
func CheckForUpdate() (latestVersion, downloadURL string, err error) {
	// Skip check if running dev version
	if Version == "dev" {
		return "", "", nil
	}

	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	req, err := http.NewRequest("GET", "https://api.github.com/repos/bulletproof-bot/backup/releases/latest", nil)
	if err != nil {
		return "", "", err
	}

	// Set user agent to avoid rate limiting
	req.Header.Set("User-Agent", "bulletproof-cli")

	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("github API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", "", err
	}

	// Compare versions (simple string comparison works for semver)
	latestVersion = release.TagName
	if latestVersion > Version && latestVersion != "" {
		return latestVersion, release.HTMLURL, nil
	}

	return "", "", nil
}

// PrintUpdateNotice prints an update notice if a newer version is available
func PrintUpdateNotice() {
	latestVersion, downloadURL, err := CheckForUpdate()
	if err != nil {
		// Silently ignore errors - don't interrupt user workflow
		return
	}

	if latestVersion != "" {
		fmt.Printf("\n💡 New version available: %s\n", latestVersion)
		fmt.Printf("   Download: %s\n\n", downloadURL)
	}
}
