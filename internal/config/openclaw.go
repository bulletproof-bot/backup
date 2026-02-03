package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// DetectInstallation detects the OpenClaw installation path
// Returns the path if found, empty string otherwise
func DetectInstallation() string {
	// Check default location first
	defaultRoot := DefaultRoot()
	if isInstalled(defaultRoot) {
		return defaultRoot
	}

	// Check common Docker volume mounts
	dockerPaths := []string{
		"/data/.openclaw",
		"/openclaw",
		"/app/.openclaw",
	}

	for _, path := range dockerPaths {
		if isInstalled(path) {
			return path
		}
	}

	return ""
}

// DefaultRoot returns the default OpenClaw root directory
func DefaultRoot() string {
	homeDir := homeDirectory()
	return filepath.Join(homeDir, ".openclaw")
}

// isInstalled checks if OpenClaw is installed at the given path
func isInstalled(path string) bool {
	configFile := filepath.Join(path, "openclaw.json")
	_, err := os.Stat(configFile)
	return err == nil
}

// IsDocker checks if running inside a Docker container
func IsDocker() bool {
	// Check for .dockerenv file
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// Check cgroup for docker (Linux only)
	if runtime.GOOS == "linux" {
		data, err := os.ReadFile("/proc/1/cgroup")
		if err == nil {
			content := string(data)
			if strings.Contains(content, "docker") || strings.Contains(content, "kubepods") {
				return true
			}
		}
	}

	return false
}

// Validate validates that the OpenClaw installation exists and is valid
func Validate(path string) error {
	// Check if directory exists
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("OpenClaw installation not found at %s: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", path)
	}

	// Check for openclaw.json
	configFile := filepath.Join(path, "openclaw.json")
	if _, err := os.Stat(configFile); err != nil {
		return fmt.Errorf("openclaw.json not found in %s (not a valid OpenClaw installation)", path)
	}

	return nil
}

// homeDirectory returns the user's home directory
func homeDirectory() string {
	if runtime.GOOS == "windows" {
		// Try USERPROFILE first, then HOME
		if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
			return userProfile
		}
		if home := os.Getenv("HOME"); home != "" {
			return home
		}
		return "C:\\Users\\Default"
	}

	if home := os.Getenv("HOME"); home != "" {
		return home
	}

	return "/home"
}

// BackupTarget represents a target file or directory to back up
type BackupTarget struct {
	Path        string
	Description string
	IsDirectory bool
	Critical    bool
}

// Exists checks if the backup target exists
func (bt *BackupTarget) Exists() bool {
	info, err := os.Stat(bt.Path)
	if err != nil {
		return false
	}

	if bt.IsDirectory {
		return info.IsDir()
	}
	return !info.IsDir()
}

// String returns a string representation of the backup target
func (bt *BackupTarget) String() string {
	return fmt.Sprintf("%s (%s)", bt.Description, bt.Path)
}

// GetBackupTargets returns all important files/directories to back up
func GetBackupTargets(openclawRoot string) []BackupTarget {
	return []BackupTarget{
		// Core config
		{
			Path:        filepath.Join(openclawRoot, "openclaw.json"),
			Description: "Main configuration",
			IsDirectory: false,
			Critical:    true,
		},

		// Soul and identity files
		{
			Path:        filepath.Join(openclawRoot, "workspace", "SOUL.md"),
			Description: "Soul file (personality)",
			IsDirectory: false,
			Critical:    true,
		},
		{
			Path:        filepath.Join(openclawRoot, "workspace", "AGENTS.md"),
			Description: "Agent definitions",
			IsDirectory: false,
			Critical:    true,
		},
		{
			Path:        filepath.Join(openclawRoot, "workspace", "TOOLS.md"),
			Description: "Tool configurations",
			IsDirectory: false,
			Critical:    true,
		},

		// Skills directory
		{
			Path:        filepath.Join(openclawRoot, "workspace", "skills"),
			Description: "Skills and capabilities",
			IsDirectory: true,
			Critical:    true,
		},

		// Memory and conversations
		{
			Path:        filepath.Join(openclawRoot, "workspace", "memory"),
			Description: "Conversation logs",
			IsDirectory: true,
			Critical:    true,
		},
		{
			Path:        filepath.Join(openclawRoot, "workspace", "MEMORY.md"),
			Description: "Long-term memory",
			IsDirectory: false,
			Critical:    true,
		},

		// Per-agent configs
		{
			Path:        filepath.Join(openclawRoot, "agents"),
			Description: "Per-agent configurations",
			IsDirectory: true,
			Critical:    false,
		},
	}
}

// SensitivePatterns returns patterns for sensitive files that should be backed up with caution
func SensitivePatterns() []string {
	return []string{
		"auth-profiles.json",
		"oauth.json",
		"**/secrets/**",
		"**/*.key",
		"**/*.pem",
	}
}
