package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfig_Validate_Success(t *testing.T) {
	// Create temp directory for testing
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	destDir := filepath.Join(tmpDir, "dest")

	// Create source directory
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	// Create test file in source
	testFile := filepath.Join(sourceDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cfg := &Config{
		OpenclawPath: sourceDir,
		Destination: &DestinationConfig{
			Type: "local",
			Path: destDir,
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Validation should succeed: %v", err)
	}

	// Verify destination was created
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		t.Errorf("Destination directory should be created")
	}
}

func TestConfig_Validate_NoDestination(t *testing.T) {
	cfg := &Config{
		OpenclawPath: "/tmp/test",
	}

	err := cfg.Validate()
	if err == nil {
		t.Errorf("Expected error when destination is nil")
	}
}

func TestConfig_Validate_NoSources(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		Destination: &DestinationConfig{
			Type: "local",
			Path: tmpDir,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Errorf("Expected error when no sources configured")
	}
}

func TestConfig_Validate_SourceDoesNotExist(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		OpenclawPath: "/nonexistent/path",
		Destination: &DestinationConfig{
			Type: "local",
			Path: tmpDir,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Errorf("Expected error when source does not exist")
	}
}

func TestConfig_Validate_SourceIsFile(t *testing.T) {
	tmpDir := t.TempDir()
	destDir := filepath.Join(tmpDir, "dest")
	sourceFile := filepath.Join(tmpDir, "file.txt")

	// Create a file instead of directory
	if err := os.WriteFile(sourceFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cfg := &Config{
		OpenclawPath: sourceFile,
		Destination: &DestinationConfig{
			Type: "local",
			Path: destDir,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Errorf("Expected error when source is a file, not a directory")
	}
}

func TestConfig_Validate_DestinationNotWritable(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root (can write to any directory)")
	}

	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	destDir := "/root/bulletproof_test" // Typically not writable by non-root users

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	cfg := &Config{
		OpenclawPath: sourceDir,
		Destination: &DestinationConfig{
			Type: "local",
			Path: destDir,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Errorf("Expected error when destination is not writable")
	}
}

func TestConfig_Validate_RetentionPolicy(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	destDir := filepath.Join(tmpDir, "dest")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	tests := []struct {
		name      string
		retention RetentionPolicy
		wantError bool
	}{
		{
			name: "Valid retention policy",
			retention: RetentionPolicy{
				Enabled:  true,
				KeepLast: 10,
			},
			wantError: false,
		},
		{
			name: "Negative keep_last",
			retention: RetentionPolicy{
				Enabled:  true,
				KeepLast: -1,
			},
			wantError: true,
		},
		{
			name: "Enabled but no rules",
			retention: RetentionPolicy{
				Enabled: true,
			},
			wantError: true,
		},
		{
			name: "Disabled policy",
			retention: RetentionPolicy{
				Enabled: false,
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				OpenclawPath: sourceDir,
				Destination: &DestinationConfig{
					Type: "local",
					Path: destDir,
				},
				Retention: tt.retention,
			}

			err := cfg.Validate()
			if tt.wantError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestConfig_Validate_GlobPatterns(t *testing.T) {
	tmpDir := t.TempDir()
	destDir := filepath.Join(tmpDir, "dest")

	// Create multiple source directories
	source1 := filepath.Join(tmpDir, "source1")
	source2 := filepath.Join(tmpDir, "source2")
	if err := os.MkdirAll(source1, 0755); err != nil {
		t.Fatalf("Failed to create source1: %v", err)
	}
	if err := os.MkdirAll(source2, 0755); err != nil {
		t.Fatalf("Failed to create source2: %v", err)
	}

	// Test with glob pattern
	cfg := &Config{
		Sources: []string{filepath.Join(tmpDir, "source*")},
		Destination: &DestinationConfig{
			Type: "local",
			Path: destDir,
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Validation with glob pattern should succeed: %v", err)
	}
}

func TestConfig_Validate_GlobNoMatches(t *testing.T) {
	tmpDir := t.TempDir()
	destDir := filepath.Join(tmpDir, "dest")

	// Glob pattern that matches nothing
	cfg := &Config{
		Sources: []string{filepath.Join(tmpDir, "nonexistent*")},
		Destination: &DestinationConfig{
			Type: "local",
			Path: destDir,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Errorf("Expected error when glob pattern matches no paths")
	}
}

func TestConfig_GetSources(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *Config
		wantCount   int
		description string
	}{
		{
			name: "Sources configured",
			cfg: &Config{
				Sources: []string{"/path1", "/path2"},
			},
			wantCount:   2,
			description: "Should return Sources when configured",
		},
		{
			name: "OpenclawPath only",
			cfg: &Config{
				OpenclawPath: "/openclaw",
			},
			wantCount:   1,
			description: "Should return OpenclawPath as single source for backward compatibility",
		},
		{
			name: "Both configured - Sources takes precedence",
			cfg: &Config{
				Sources:      []string{"/path1", "/path2"},
				OpenclawPath: "/openclaw",
			},
			wantCount:   2,
			description: "Should return Sources when both are configured",
		},
		{
			name:        "Neither configured",
			cfg:         &Config{},
			wantCount:   0,
			description: "Should return empty slice when neither configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sources := tt.cfg.GetSources()
			if len(sources) != tt.wantCount {
				t.Errorf("%s: expected %d sources, got %d", tt.description, tt.wantCount, len(sources))
			}
		})
	}
}

func TestExpandGlobPattern(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test directories
	dir1 := filepath.Join(tmpDir, "test1")
	dir2 := filepath.Join(tmpDir, "test2")
	if err := os.MkdirAll(dir1, 0755); err != nil {
		t.Fatalf("Failed to create dir1: %v", err)
	}
	if err := os.MkdirAll(dir2, 0755); err != nil {
		t.Fatalf("Failed to create dir2: %v", err)
	}

	tests := []struct {
		name      string
		pattern   string
		wantCount int
		wantError bool
	}{
		{
			name:      "Glob with wildcard",
			pattern:   filepath.Join(tmpDir, "test*"),
			wantCount: 2,
			wantError: false,
		},
		{
			name:      "Exact path",
			pattern:   dir1,
			wantCount: 1,
			wantError: false,
		},
		{
			name:      "Tilde expansion",
			pattern:   "~/test",
			wantCount: 1, // Should expand to home directory
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths, err := expandGlobPattern(tt.pattern)
			if tt.wantError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if !tt.wantError && len(paths) != tt.wantCount {
				t.Errorf("Expected %d paths, got %d", tt.wantCount, len(paths))
			}
		})
	}
}
