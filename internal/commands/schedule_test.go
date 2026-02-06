package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bulletproof-bot/backup/internal/config"
)

// setupTestConfig creates a temporary HOME directory for testing
func setupTestConfig(t *testing.T) (cleanup func()) {
	t.Helper()

	tempDir := t.TempDir()

	// Set HOME to temp directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)

	// Create .config/bulletproof directory
	configDir := filepath.Join(tempDir, ".config", "bulletproof")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	return func() {
		os.Setenv("HOME", oldHome)
	}
}

func TestScheduleEnable(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	// Create initial config
	cfg := &config.Config{
		OpenclawPath: "/test/.openclaw",
		Destination: &config.DestinationConfig{
			Type: "local",
			Path: "/test/backups",
		},
		Schedule: config.ScheduleConfig{
			Enabled: false,
			Time:    "03:00",
		},
		Options: config.BackupOptions{
			Exclude: []string{"*.log"},
		},
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Failed to save initial config: %v", err)
	}

	// Test enable with default time
	if err := runScheduleEnable("03:00"); err != nil {
		t.Errorf("runScheduleEnable failed: %v", err)
	}

	// Verify config was updated
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if !cfg.Schedule.Enabled {
		t.Error("Schedule should be enabled")
	}

	if cfg.Schedule.Time != "03:00" {
		t.Errorf("Expected time 03:00, got %s", cfg.Schedule.Time)
	}
}

func TestScheduleDisable(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	// Create initial config with schedule enabled
	cfg := &config.Config{
		OpenclawPath: "/test/.openclaw",
		Destination: &config.DestinationConfig{
			Type: "local",
			Path: "/test/backups",
		},
		Schedule: config.ScheduleConfig{
			Enabled: true,
			Time:    "03:00",
		},
		Options: config.BackupOptions{
			Exclude: []string{"*.log"},
		},
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Failed to save initial config: %v", err)
	}

	// Test disable
	if err := runScheduleDisable(); err != nil {
		t.Errorf("runScheduleDisable failed: %v", err)
	}

	// Verify config was updated
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Schedule.Enabled {
		t.Error("Schedule should be disabled")
	}
}

func TestScheduleEnableCustomTime(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	// Create initial config
	cfg := &config.Config{
		OpenclawPath: "/test/.openclaw",
		Destination: &config.DestinationConfig{
			Type: "local",
			Path: "/test/backups",
		},
		Schedule: config.ScheduleConfig{
			Enabled: false,
			Time:    "03:00",
		},
		Options: config.BackupOptions{
			Exclude: []string{"*.log"},
		},
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Failed to save initial config: %v", err)
	}

	// Test enable with custom time
	if err := runScheduleEnable("14:30"); err != nil {
		t.Errorf("runScheduleEnable with custom time failed: %v", err)
	}

	// Verify config was updated
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if !cfg.Schedule.Enabled {
		t.Error("Schedule should be enabled")
	}

	if cfg.Schedule.Time != "14:30" {
		t.Errorf("Expected time 14:30, got %s", cfg.Schedule.Time)
	}
}

func TestIsValidTime(t *testing.T) {
	tests := []struct {
		time  string
		valid bool
	}{
		{"00:00", true},
		{"03:00", true},
		{"12:30", true},
		{"23:59", true},
		{"24:00", false},    // Invalid hour
		{"12:60", false},    // Invalid minute
		{"3:00", false},     // Missing leading zero
		{"12:3", false},     // Missing trailing zero
		{"12-30", false},    // Wrong separator
		{"12:00:00", false}, // Too long
		{"", false},         // Empty
	}

	for _, tt := range tests {
		result := isValidTime(tt.time)
		if result != tt.valid {
			t.Errorf("isValidTime(%q) = %v, expected %v", tt.time, result, tt.valid)
		}
	}
}

func TestScheduleEnableInvalidTime(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	// Create initial config
	cfg := &config.Config{
		OpenclawPath: "/test/.openclaw",
		Destination: &config.DestinationConfig{
			Type: "local",
			Path: "/test/backups",
		},
		Schedule: config.ScheduleConfig{
			Enabled: false,
			Time:    "03:00",
		},
		Options: config.BackupOptions{
			Exclude: []string{"*.log"},
		},
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Failed to save initial config: %v", err)
	}

	// Test enable with invalid time
	err := runScheduleEnable("25:00") // Invalid hour
	if err == nil {
		t.Error("Expected error for invalid time, got nil")
	}

	// Verify config was not changed
	cfg, err = config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Schedule.Enabled {
		t.Error("Schedule should still be disabled after invalid enable attempt")
	}
}

func TestScheduleStatus(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	// Create initial config
	cfg := &config.Config{
		OpenclawPath: "/test/.openclaw",
		Destination: &config.DestinationConfig{
			Type: "local",
			Path: "/test/backups",
		},
		Schedule: config.ScheduleConfig{
			Enabled: true,
			Time:    "14:00",
		},
		Options: config.BackupOptions{
			Exclude: []string{"*.log"},
		},
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Failed to save initial config: %v", err)
	}

	// Test status (should not error)
	if err := runScheduleStatus(); err != nil {
		t.Errorf("runScheduleStatus failed: %v", err)
	}
}
