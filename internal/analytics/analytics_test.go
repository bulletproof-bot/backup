package analytics

import (
	"os"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/bulletproof-bot/backup/internal/config"
)

func TestGenerateUserID_Format(t *testing.T) {
	userID, err := GenerateUserID()
	if err != nil {
		t.Fatalf("GenerateUserID() failed: %v", err)
	}

	// UUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	pattern := `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`
	if !regexp.MustCompile(pattern).MatchString(userID) {
		t.Errorf("invalid UserID format: %s (expected UUID format)", userID)
	}
}

func TestGenerateUserID_Uniqueness(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id, err := GenerateUserID()
		if err != nil {
			t.Fatalf("GenerateUserID() failed: %v", err)
		}
		if ids[id] {
			t.Errorf("duplicate UserID generated: %s", id)
		}
		ids[id] = true
	}

	if len(ids) != 100 {
		t.Errorf("expected 100 unique IDs, got %d", len(ids))
	}
}

func TestTrackEvent_DisabledAnalytics(t *testing.T) {
	cfg := &config.Config{
		Analytics: config.AnalyticsConfig{Enabled: false},
	}

	// Should return immediately without generating UserID
	TrackEvent(cfg, Event{Command: "test"})

	if cfg.Analytics.UserID != "" {
		t.Error("UserID should not be generated when analytics disabled")
	}
}

func TestTrackEvent_GeneratesUserID(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	cfg := &config.Config{
		Analytics: config.AnalyticsConfig{
			Enabled: true,
			UserID:  "",
		},
	}

	TrackEvent(cfg, Event{Command: "test"})
	time.Sleep(50 * time.Millisecond) // Allow time for async UserID generation

	if cfg.Analytics.UserID == "" {
		t.Error("UserID should be generated on first run")
	}

	// Verify UUID format
	pattern := `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`
	if !regexp.MustCompile(pattern).MatchString(cfg.Analytics.UserID) {
		t.Errorf("invalid generated UserID format: %s", cfg.Analytics.UserID)
	}
}

func TestTrackEvent_Race(t *testing.T) {
	// This test should pass with -race flag
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	cfg := &config.Config{
		Analytics: config.AnalyticsConfig{Enabled: false},
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			TrackEvent(cfg, Event{Command: "test"})
		}(i)
	}
	wg.Wait()
}

func TestTrackEvent_ConcurrentCalls(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	cfg := &config.Config{
		Analytics: config.AnalyticsConfig{
			Enabled: true,
			UserID:  "", // Empty to trigger generation
		},
	}

	// Launch 10 concurrent TrackEvent calls
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			TrackEvent(cfg, Event{Command: "test"})
		}(i)
	}
	wg.Wait()
	time.Sleep(100 * time.Millisecond) // Allow goroutines to complete

	// Verify UserID was set and is valid format
	if cfg.Analytics.UserID == "" {
		t.Error("UserID should be set after concurrent calls")
	}

	// UUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	pattern := `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`
	if !regexp.MustCompile(pattern).MatchString(cfg.Analytics.UserID) {
		t.Errorf("invalid UserID format after concurrent calls: %s", cfg.Analytics.UserID)
	}
}

func TestShowFirstRunNotice_OnlyOnce(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	cfg := &config.Config{
		Analytics: config.AnalyticsConfig{NoticeShown: false},
	}

	ShowFirstRunNotice(cfg)
	if !cfg.Analytics.NoticeShown {
		t.Error("NoticeShown should be set to true after first call")
	}

	// Second call should not print (manual verification)
	// The function checks NoticeShown and returns early
	ShowFirstRunNotice(cfg)
	if !cfg.Analytics.NoticeShown {
		t.Error("NoticeShown should remain true after second call")
	}
}

func TestEvent_Fields(t *testing.T) {
	event := Event{
		Command: "backup",
		OS:      "linux",
		Arch:    "amd64",
		Version: "1.0.0",
		Flags:   map[string]string{"test": "value"},
	}

	// Verify fields are set correctly
	if event.Command != "backup" {
		t.Errorf("Command = %s, want backup", event.Command)
	}
	if event.OS != "linux" {
		t.Errorf("OS = %s, want linux", event.OS)
	}
	if event.Arch != "amd64" {
		t.Errorf("Arch = %s, want amd64", event.Arch)
	}
	if event.Version != "1.0.0" {
		t.Errorf("Version = %s, want 1.0.0", event.Version)
	}
	if event.Flags["test"] != "value" {
		t.Errorf("Flags[test] = %s, want value", event.Flags["test"])
	}
}
