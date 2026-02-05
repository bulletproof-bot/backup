package analytics

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/bulletproof-bot/backup/internal/config"
	"github.com/bulletproof-bot/backup/internal/version"
)

const (
	plausibleDomain = "bulletproof.bot"
	plausibleAPI    = "https://plausible.io/api/event"
)

var (
	// userIDMutex protects concurrent UserID generation
	userIDMutex sync.Mutex
)

// Event represents an analytics event
type Event struct {
	Command string
	OS      string
	Arch    string
	Version string
	Flags   map[string]string
}

// GenerateUserID creates a new anonymous user ID
func GenerateUserID() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to generate user ID: %w", err)
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

// TrackEvent sends an analytics event asynchronously (non-blocking)
func TrackEvent(cfg *config.Config, event Event) {
	// Skip if analytics disabled
	if !cfg.Analytics.Enabled {
		return
	}

	// Generate user ID if not set (with mutex protection)
	userIDMutex.Lock()
	userID := cfg.Analytics.UserID
	if userID == "" {
		var err error
		userID, err = GenerateUserID()
		if err != nil {
			userIDMutex.Unlock()
			return // Silent failure
		}
		cfg.Analytics.UserID = userID
		if err := cfg.Save(); err != nil {
			userIDMutex.Unlock()
			return // Silent failure
		}
	}
	userIDMutex.Unlock()

	// Capture values for goroutine (avoid race condition)
	eventCopy := event
	go func() {
		if err := sendEvent(userID, eventCopy); err != nil {
			// Silent failure - analytics should never disrupt user experience
			return
		}
	}()
}

// sendEvent sends the event to Plausible Analytics
func sendEvent(userID string, event Event) error {
	// Build event payload
	payload := map[string]interface{}{
		"domain": plausibleDomain,
		"name":   "command:" + event.Command,
		"url":    "app://bulletproof/" + event.Command,
		"props": map[string]string{
			"os":      event.OS,
			"arch":    event.Arch,
			"version": event.Version,
		},
	}

	// Add flags to props (only flag names, never values)
	for flagName := range event.Flags {
		payload["props"].(map[string]string)["flag_"+flagName] = "true"
	}

	// Marshal JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", plausibleAPI, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("bulletproof/%s", version.Version))
	req.Header.Set("X-Forwarded-For", userID) // Use user ID as anonymous identifier

	// Send request with timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// ShowFirstRunNotice displays the analytics notice on first run
func ShowFirstRunNotice(cfg *config.Config) {
	if cfg.Analytics.NoticeShown {
		return
	}

	fmt.Print(`
╭─────────────────────────────────────────────────────────────╮
│ Bulletproof collects anonymous usage analytics to improve  │
│ the tool. We track:                                         │
│   • Command usage (backup, restore, etc.)                   │
│   • Operating system and version                            │
│   • Tool version                                            │
│   • Flag usage (e.g., --dry-run)                            │
│                                                              │
│ We DO NOT track:                                            │
│   • File paths or names                                     │
│   • Snapshot IDs or contents                                │
│   • Configuration values                                    │
│   • Any personally identifiable information                 │
│                                                              │
│ To disable analytics:                                       │
│   bulletproof analytics disable                             │
│                                                              │
│ For more info: https://github.com/bulletproof-bot/backup   │
╰─────────────────────────────────────────────────────────────╯
`)

	// Mark notice as shown
	cfg.Analytics.NoticeShown = true
	if err := cfg.Save(); err != nil {
		// Silent failure
		return
	}
}

// TrackCommand is a helper to track a command execution
func TrackCommand(command string, flags map[string]string) {
	cfg, err := config.Load()
	if err != nil {
		return // Silent failure
	}

	// Show first-run notice
	ShowFirstRunNotice(cfg)

	// Track event
	TrackEvent(cfg, Event{
		Command: command,
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
		Version: version.Version,
		Flags:   flags,
	})
}
