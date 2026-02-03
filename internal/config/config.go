package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const ConfigVersion = "1"

// Config represents the bulletproof configuration
type Config struct {
	OpenclawPath string             `yaml:"openclaw_path,omitempty"`
	Destination  *DestinationConfig `yaml:"destination,omitempty"`
	Schedule     ScheduleConfig     `yaml:"schedule"`
	Options      BackupOptions      `yaml:"options"`
}

// DestinationConfig specifies the backup destination
type DestinationConfig struct {
	Type string `yaml:"type"` // 'git', 'local', or 'sync'
	Path string `yaml:"path"`
}

// ScheduleConfig controls automatic backup scheduling
type ScheduleConfig struct {
	Enabled bool   `yaml:"enabled"`
	Time    string `yaml:"time"` // HH:MM format
}

// BackupOptions controls backup behavior
type BackupOptions struct {
	IncludeAuth bool     `yaml:"include_auth"`
	Exclude     []string `yaml:"exclude"`
}

// IsGit returns true if the destination is a git repository
func (d *DestinationConfig) IsGit() bool {
	return d.Type == "git"
}

// IsLocal returns true if the destination is a local directory
func (d *DestinationConfig) IsLocal() bool {
	return d.Type == "local"
}

// IsSync returns true if the destination is a sync directory
func (d *DestinationConfig) IsSync() bool {
	return d.Type == "sync"
}

// Hour returns the hour component of the schedule time
func (s *ScheduleConfig) Hour() (int, error) {
	parts := strings.Split(s.Time, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid time format: %s", s.Time)
	}
	return strconv.Atoi(parts[0])
}

// Minute returns the minute component of the schedule time
func (s *ScheduleConfig) Minute() (int, error) {
	parts := strings.Split(s.Time, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid time format: %s", s.Time)
	}
	return strconv.Atoi(parts[1])
}

// ConfigPath returns the path to the config file
func ConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".config", "bulletproof", "config.yaml"), nil
}

// ConfigDir returns the path to the config directory
func ConfigDir() (string, error) {
	configPath, err := ConfigPath()
	if err != nil {
		return "", err
	}
	return filepath.Dir(configPath), nil
}

// Exists checks if the config file exists
func Exists() (bool, error) {
	configPath, err := ConfigPath()
	if err != nil {
		return false, err
	}

	_, err = os.Stat(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to stat config file: %w", err)
	}

	return true, nil
}

// Load loads the configuration from the config file
func Load() (*Config, error) {
	configPath, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	// Return empty config if file doesn't exist
	exists, err := Exists()
	if err != nil {
		return nil, err
	}
	if !exists {
		return &Config{
			Schedule: ScheduleConfig{
				Enabled: false,
				Time:    "03:00",
			},
			Options: BackupOptions{
				IncludeAuth: false,
				Exclude:     []string{"*.log", "node_modules/", ".git/"},
			},
		}, nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Set defaults if not specified
	if config.Schedule.Time == "" {
		config.Schedule.Time = "03:00"
	}
	if config.Options.Exclude == nil {
		config.Options.Exclude = []string{"*.log", "node_modules/", ".git/"}
	}

	return &config, nil
}

// Save saves the configuration to the config file
func (c *Config) Save() error {
	configPath, err := ConfigPath()
	if err != nil {
		return err
	}

	// Create config directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Build YAML with comments
	var builder strings.Builder
	builder.WriteString("# Bulletproof configuration\n")
	builder.WriteString("# https://github.com/bulletproof-bot/backup\n")
	builder.WriteString("version: \"" + ConfigVersion + "\"\n")
	builder.WriteString("\n")

	if c.OpenclawPath != "" {
		builder.WriteString("# Path to OpenClaw installation\n")
		builder.WriteString("openclaw_path: \"" + c.OpenclawPath + "\"\n")
		builder.WriteString("\n")
	}

	if c.Destination != nil {
		builder.WriteString("# Backup destination\n")
		builder.WriteString("destination:\n")
		builder.WriteString("  type: \"" + c.Destination.Type + "\"\n")
		builder.WriteString("  path: \"" + c.Destination.Path + "\"\n")
		builder.WriteString("\n")
	}

	builder.WriteString("# Backup schedule\n")
	builder.WriteString("schedule:\n")
	builder.WriteString("  enabled: " + strconv.FormatBool(c.Schedule.Enabled) + "\n")
	builder.WriteString("  time: \"" + c.Schedule.Time + "\"\n")
	builder.WriteString("\n")

	builder.WriteString("# Backup options\n")
	builder.WriteString("options:\n")
	builder.WriteString("  include_auth: " + strconv.FormatBool(c.Options.IncludeAuth) + "\n")
	if len(c.Options.Exclude) > 0 {
		builder.WriteString("  exclude:\n")
		for _, pattern := range c.Options.Exclude {
			builder.WriteString("    - \"" + pattern + "\"\n")
		}
	}

	// Write to file
	if err := os.WriteFile(configPath, []byte(builder.String()), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// CopyWith returns a new Config with updated values
func (c *Config) CopyWith(openclawPath *string, destination *DestinationConfig, schedule *ScheduleConfig, options *BackupOptions) *Config {
	newConfig := &Config{
		OpenclawPath: c.OpenclawPath,
		Destination:  c.Destination,
		Schedule:     c.Schedule,
		Options:      c.Options,
	}

	if openclawPath != nil {
		newConfig.OpenclawPath = *openclawPath
	}
	if destination != nil {
		newConfig.Destination = destination
	}
	if schedule != nil {
		newConfig.Schedule = *schedule
	}
	if options != nil {
		newConfig.Options = *options
	}

	return newConfig
}

// String returns a string representation of the config
func (c *Config) String() string {
	return fmt.Sprintf("Config(\n  openclawPath: %s,\n  destination: %v,\n  schedule: %v,\n  options: %v,\n)",
		c.OpenclawPath, c.Destination, c.Schedule, c.Options)
}
