package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_EmptyConfig(t *testing.T) {
	// Use a non-existent path to test loading empty config
	originalPath := os.Getenv("HOME")
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalPath)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.OpenclawPath != "" {
		t.Errorf("expected empty OpenclawPath, got %s", cfg.OpenclawPath)
	}

	if cfg.Destination != nil {
		t.Error("expected nil Destination")
	}

	if cfg.Schedule.Time != "03:00" {
		t.Errorf("expected default time 03:00, got %s", cfg.Schedule.Time)
	}

	if len(cfg.Options.Exclude) != 3 {
		t.Errorf("expected 3 default exclusions, got %d", len(cfg.Options.Exclude))
	}
}

func TestSave_Load_RoundTrip(t *testing.T) {
	tempDir := t.TempDir()
	originalPath := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalPath)

	// Create a config
	cfg := &Config{
		OpenclawPath: "/test/openclaw",
		Destination: &DestinationConfig{
			Type: "local",
			Path: "/test/backup",
		},
		Schedule: ScheduleConfig{
			Enabled: true,
			Time:    "02:30",
		},
		Options: BackupOptions{
			IncludeAuth: true,
			Exclude:     []string{"*.tmp", "cache/"},
		},
	}

	// Save it
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Load it back
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify fields
	if loaded.OpenclawPath != cfg.OpenclawPath {
		t.Errorf("OpenclawPath: got %s, want %s", loaded.OpenclawPath, cfg.OpenclawPath)
	}

	if loaded.Destination.Type != cfg.Destination.Type {
		t.Errorf("Destination.Type: got %s, want %s", loaded.Destination.Type, cfg.Destination.Type)
	}

	if loaded.Destination.Path != cfg.Destination.Path {
		t.Errorf("Destination.Path: got %s, want %s", loaded.Destination.Path, cfg.Destination.Path)
	}

	if loaded.Schedule.Enabled != cfg.Schedule.Enabled {
		t.Errorf("Schedule.Enabled: got %v, want %v", loaded.Schedule.Enabled, cfg.Schedule.Enabled)
	}

	if loaded.Schedule.Time != cfg.Schedule.Time {
		t.Errorf("Schedule.Time: got %s, want %s", loaded.Schedule.Time, cfg.Schedule.Time)
	}

	if loaded.Options.IncludeAuth != cfg.Options.IncludeAuth {
		t.Errorf("Options.IncludeAuth: got %v, want %v", loaded.Options.IncludeAuth, cfg.Options.IncludeAuth)
	}
}

func TestSave_Load_RoundTrip_SpecialChars(t *testing.T) {
	tempDir := t.TempDir()
	originalPath := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalPath)

	cfg := &Config{
		OpenclawPath: `/test/path with spaces/openclaw: "quoted"`,
		Destination: &DestinationConfig{
			Type: "local",
			Path: `/test/backup/special chars: {glob}`,
		},
		Schedule: ScheduleConfig{
			Enabled: true,
			Time:    "02:30",
		},
		Options: BackupOptions{
			IncludeAuth: false,
			Exclude:     []string{"*.tmp", `path with "quotes"`, "colon: value"},
		},
		Scripts: ScriptsConfig{
			PreBackup: []ScriptConfig{
				{Name: "export db", Command: `echo "hello: world"`, Timeout: 30},
			},
		},
		Retention: RetentionPolicy{
			Enabled:  true,
			KeepLast: 5,
		},
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if loaded.OpenclawPath != cfg.OpenclawPath {
		t.Errorf("OpenclawPath: got %q, want %q", loaded.OpenclawPath, cfg.OpenclawPath)
	}
	if loaded.Destination.Path != cfg.Destination.Path {
		t.Errorf("Destination.Path: got %q, want %q", loaded.Destination.Path, cfg.Destination.Path)
	}
	if len(loaded.Options.Exclude) != len(cfg.Options.Exclude) {
		t.Fatalf("Exclude count: got %d, want %d", len(loaded.Options.Exclude), len(cfg.Options.Exclude))
	}
	for i, want := range cfg.Options.Exclude {
		if loaded.Options.Exclude[i] != want {
			t.Errorf("Exclude[%d]: got %q, want %q", i, loaded.Options.Exclude[i], want)
		}
	}
	if len(loaded.Scripts.PreBackup) != 1 {
		t.Fatalf("PreBackup script count: got %d, want 1", len(loaded.Scripts.PreBackup))
	}
	if loaded.Scripts.PreBackup[0].Command != cfg.Scripts.PreBackup[0].Command {
		t.Errorf("PreBackup command: got %q, want %q", loaded.Scripts.PreBackup[0].Command, cfg.Scripts.PreBackup[0].Command)
	}
	if !loaded.Retention.Enabled || loaded.Retention.KeepLast != 5 {
		t.Errorf("Retention: got enabled=%v keep_last=%d, want enabled=true keep_last=5",
			loaded.Retention.Enabled, loaded.Retention.KeepLast)
	}
}

func TestScheduleConfig_HourMinute(t *testing.T) {
	tests := []struct {
		name       string
		time       string
		wantHour   int
		wantMinute int
		wantErr    bool
	}{
		{
			name:       "valid time",
			time:       "14:30",
			wantHour:   14,
			wantMinute: 30,
			wantErr:    false,
		},
		{
			name:       "midnight",
			time:       "00:00",
			wantHour:   0,
			wantMinute: 0,
			wantErr:    false,
		},
		{
			name:    "invalid format",
			time:    "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ScheduleConfig{Time: tt.time}

			hour, err := cfg.Hour()
			if (err != nil) != tt.wantErr {
				t.Errorf("Hour() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && hour != tt.wantHour {
				t.Errorf("Hour() = %d, want %d", hour, tt.wantHour)
			}

			minute, err := cfg.Minute()
			if (err != nil) != tt.wantErr {
				t.Errorf("Minute() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && minute != tt.wantMinute {
				t.Errorf("Minute() = %d, want %d", minute, tt.wantMinute)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tempDir := t.TempDir()

	// Create a valid OpenClaw directory
	openclawDir := filepath.Join(tempDir, ".openclaw")
	if err := os.MkdirAll(openclawDir, 0755); err != nil {
		t.Fatal(err)
	}

	configFile := filepath.Join(openclawDir, "openclaw.json")
	if err := os.WriteFile(configFile, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test valid path
	if err := Validate(openclawDir); err != nil {
		t.Errorf("Validate() on valid path failed: %v", err)
	}

	// Test invalid path (no openclaw.json)
	invalidDir := filepath.Join(tempDir, "invalid")
	if err := os.MkdirAll(invalidDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := Validate(invalidDir); err == nil {
		t.Error("Validate() should fail on directory without openclaw.json")
	}

	// Test non-existent path
	if err := Validate(filepath.Join(tempDir, "nonexistent")); err == nil {
		t.Error("Validate() should fail on non-existent directory")
	}
}

func TestConfigPath_NoHomeDir(t *testing.T) {
	// Save original HOME
	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		}
	}()

	// Unset HOME to simulate container environment where HOME is not set
	os.Unsetenv("HOME")

	// ConfigPath should return an error, not panic
	_, err := ConfigPath()
	if err == nil {
		t.Error("ConfigPath() should return error when HOME is not set")
	}

	// Verify the error message mentions home directory
	if err != nil && err.Error() == "" {
		t.Error("ConfigPath() should return a descriptive error message")
	}
}
