package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bulletproof-bot/backup/internal/errors"
	"gopkg.in/yaml.v3"
)

const ConfigVersion = "1"

// Config represents the bulletproof configuration
type Config struct {
	OpenclawPath string             `yaml:"openclaw_path,omitempty"`
	Sources      []string           `yaml:"sources,omitempty"`
	Destination  *DestinationConfig `yaml:"destination,omitempty"`
	Schedule     ScheduleConfig     `yaml:"schedule"`
	Options      BackupOptions      `yaml:"options"`
	Scripts      ScriptsConfig      `yaml:"scripts,omitempty"`
	Analytics    AnalyticsConfig    `yaml:"analytics,omitempty"`
	Retention    RetentionPolicy    `yaml:"retention,omitempty"`
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

// ScriptConfig represents a single script configuration
type ScriptConfig struct {
	Name    string `yaml:"name"`
	Command string `yaml:"command"`
	Timeout int    `yaml:"timeout"` // seconds, 0 = default (60s)
}

// ScriptsConfig controls script execution
type ScriptsConfig struct {
	PreBackup   []ScriptConfig `yaml:"pre_backup,omitempty"`
	PostRestore []ScriptConfig `yaml:"post_restore,omitempty"`
}

// AnalyticsConfig controls anonymous usage analytics
type AnalyticsConfig struct {
	Enabled     bool   `yaml:"enabled"`
	UserID      string `yaml:"user_id,omitempty"`
	NoticeShown bool   `yaml:"notice_shown"`
}

// RetentionPolicy controls snapshot retention and pruning
type RetentionPolicy struct {
	Enabled     bool `yaml:"enabled"`
	KeepLast    int  `yaml:"keep_last,omitempty"`    // Keep last N snapshots
	KeepDaily   int  `yaml:"keep_daily,omitempty"`   // Keep one snapshot per day for N days
	KeepWeekly  int  `yaml:"keep_weekly,omitempty"`  // Keep one snapshot per week for N weeks
	KeepMonthly int  `yaml:"keep_monthly,omitempty"` // Keep one snapshot per month for N months
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
			Analytics: AnalyticsConfig{
				Enabled:     true,
				NoticeShown: false,
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

	// Set analytics defaults - enabled by default for new configs
	if config.Analytics.UserID == "" && config.Analytics.Enabled {
		// UserID will be generated on first use
		config.Analytics.Enabled = true
	}

	return &config, nil
}

// saveConfig is the serialization wrapper that adds a version field
type saveConfig struct {
	Version      string             `yaml:"version"`
	OpenclawPath string             `yaml:"openclaw_path,omitempty"`
	Sources      []string           `yaml:"sources,omitempty"`
	Destination  *DestinationConfig `yaml:"destination,omitempty"`
	Schedule     ScheduleConfig     `yaml:"schedule"`
	Options      BackupOptions      `yaml:"options"`
	Scripts      *ScriptsConfig     `yaml:"scripts,omitempty"`
	Analytics    AnalyticsConfig    `yaml:"analytics"`
	Retention    *RetentionPolicy   `yaml:"retention,omitempty"`
}

// Save saves the configuration to the config file using yaml.v3 marshaling
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

	sc := saveConfig{
		Version:      ConfigVersion,
		OpenclawPath: c.OpenclawPath,
		Sources:      c.Sources,
		Destination:  c.Destination,
		Schedule:     c.Schedule,
		Options:      c.Options,
		Analytics:    c.Analytics,
	}

	// Only include scripts section if any scripts are configured
	if len(c.Scripts.PreBackup) > 0 || len(c.Scripts.PostRestore) > 0 {
		sc.Scripts = &c.Scripts
	}

	// Only include retention section if any retention settings are configured
	if c.Retention.Enabled || c.Retention.KeepLast > 0 || c.Retention.KeepDaily > 0 || c.Retention.KeepWeekly > 0 || c.Retention.KeepMonthly > 0 {
		sc.Retention = &c.Retention
	}

	// Marshal to yaml.Node for comment support
	var node yaml.Node
	if err := node.Encode(sc); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	// Add section comments
	node.HeadComment = "Bulletproof configuration\nhttps://github.com/bulletproof-bot/backup"
	addConfigComments(&node)

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&node); err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := enc.Close(); err != nil {
		return fmt.Errorf("failed to finalize config: %w", err)
	}

	if err := os.WriteFile(configPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// addConfigComments annotates YAML mapping keys with descriptive comments
func addConfigComments(node *yaml.Node) {
	if node.Kind != yaml.MappingNode {
		return
	}

	comments := map[string]string{
		"openclaw_path": "Path to OpenClaw installation (deprecated, use sources instead)",
		"sources":       "Source paths to back up (supports glob patterns)",
		"destination":   "Backup destination",
		"schedule":      "Backup schedule",
		"options":       "Backup options",
		"scripts":       "Script execution",
		"analytics":     "Anonymous usage analytics",
		"retention":     "Snapshot retention policy",
	}

	for i := 0; i < len(node.Content)-1; i += 2 {
		key := node.Content[i]
		if comment, ok := comments[key.Value]; ok {
			key.HeadComment = comment
		}
	}
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

// GetSources returns all source paths to back up
// Returns Sources if configured, otherwise returns OpenclawPath for backward compatibility
func (c *Config) GetSources() []string {
	if len(c.Sources) > 0 {
		return c.Sources
	}
	if c.OpenclawPath != "" {
		return []string{c.OpenclawPath}
	}
	return nil
}

// Validate performs comprehensive validation of the configuration
func (c *Config) Validate() error {
	// Validate destination
	if c.Destination == nil {
		return errors.NewActionableError(
			"validate configuration",
			fmt.Errorf("no destination configured"),
			[]string{"Bulletproof has not been initialized yet", "Config file is incomplete"},
			"bulletproof init",
			"",
		)
	}

	// Check if destination path exists
	if c.Destination.Path == "" {
		return fmt.Errorf("destination path is empty")
	}

	// For local and sync destinations, check if path is writable
	if c.Destination.Type == "local" || c.Destination.Type == "sync" {
		// Check if destination exists
		info, err := os.Stat(c.Destination.Path)
		if err != nil {
			if os.IsNotExist(err) {
				// Try to create it
				if err := os.MkdirAll(c.Destination.Path, 0755); err != nil {
					return errors.BackupDestinationError(
						"create backup destination",
						c.Destination.Path,
						err,
					)
				}
			} else {
				return fmt.Errorf("failed to check destination path: %w", err)
			}
		} else if !info.IsDir() {
			return errors.BackupDestinationError(
				"validate backup destination",
				c.Destination.Path,
				fmt.Errorf("path is not a directory"),
			)
		}

		// Check write permissions by creating a test file
		testFile := filepath.Join(c.Destination.Path, ".bulletproof_test")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			return errors.PermissionDenied(
				"write to backup destination",
				c.Destination.Path,
				err,
			)
		}
		os.Remove(testFile)
	}

	// Validate sources
	sources := c.GetSources()
	if len(sources) == 0 {
		return errors.OpenClawNotFound()
	}

	// Expand glob patterns and validate each source
	for _, source := range sources {
		expandedPaths, err := expandGlobPattern(source)
		if err != nil {
			return fmt.Errorf("failed to expand source pattern %s: %w", source, err)
		}

		if len(expandedPaths) == 0 {
			return errors.NewActionableError(
				"expand source pattern",
				fmt.Errorf("pattern matches no paths: %s", source),
				[]string{
					"Glob pattern is too specific",
					"Source directories don't exist yet",
					"Typo in path pattern",
				},
				fmt.Sprintf("Check if directories exist:\nls -d %s", source),
				"bulletproof config show",
			)
		}

		for _, path := range expandedPaths {
			// Check if path exists
			info, err := os.Stat(path)
			if err != nil {
				if os.IsNotExist(err) {
					return errors.NewActionableError(
						"validate source path",
						fmt.Errorf("path does not exist: %s", path),
						[]string{
							"Source directory hasn't been created yet",
							"Path was moved or deleted",
							"Typo in configuration",
						},
						fmt.Sprintf("Create the directory:\nmkdir -p %s\n\nOr update config:\nbulletproof config set openclaw_path /correct/path", path),
						"bulletproof config show",
					)
				}
				return fmt.Errorf("failed to check source path %s: %w", path, err)
			}

			// Check if it's a directory
			if !info.IsDir() {
				return errors.NewActionableError(
					"validate source path",
					fmt.Errorf("path is not a directory: %s", path),
					[]string{
						"Path points to a file instead of directory",
						"Configuration error",
					},
					fmt.Sprintf("Use the parent directory:\nbulletproof config set openclaw_path %s", filepath.Dir(path)),
					"ls -ld "+path,
				)
			}

			// Check read permissions
			if _, err := os.ReadDir(path); err != nil {
				return errors.PermissionDenied("read source directory", path, err)
			}
		}
	}

	// Validate script files exist and are executable
	for _, script := range c.Scripts.PreBackup {
		if err := validateScript(script); err != nil {
			return fmt.Errorf("pre-backup script %s: %w", script.Name, err)
		}
	}
	for _, script := range c.Scripts.PostRestore {
		if err := validateScript(script); err != nil {
			return fmt.Errorf("post-restore script %s: %w", script.Name, err)
		}
	}

	// Validate retention policy
	if c.Retention.Enabled {
		if c.Retention.KeepLast < 0 || c.Retention.KeepDaily < 0 || c.Retention.KeepWeekly < 0 || c.Retention.KeepMonthly < 0 {
			return fmt.Errorf("retention policy values cannot be negative")
		}
		if c.Retention.KeepLast == 0 && c.Retention.KeepDaily == 0 && c.Retention.KeepWeekly == 0 && c.Retention.KeepMonthly == 0 {
			return fmt.Errorf("retention policy enabled but no retention rules configured")
		}
	}

	return nil
}

// validateScript checks if a script command is valid
func validateScript(script ScriptConfig) error {
	if script.Name == "" {
		return fmt.Errorf("script name is empty")
	}
	if script.Command == "" {
		return fmt.Errorf("script command is empty")
	}

	// If the command is a path to a file, check if it exists and is executable
	// Otherwise assume it's a shell command
	parts := strings.Fields(script.Command)
	if len(parts) == 0 {
		return fmt.Errorf("script command is empty after parsing")
	}

	cmdPath := parts[0]
	if strings.Contains(cmdPath, "/") || strings.Contains(cmdPath, "\\") {
		// It's a file path, check if it exists
		info, err := os.Stat(cmdPath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("script file does not exist: %s", cmdPath)
			}
			return fmt.Errorf("failed to check script file %s: %w", cmdPath, err)
		}

		// Check if it's executable (Unix systems)
		if info.Mode()&0111 == 0 {
			return fmt.Errorf("script file is not executable: %s (hint: chmod +x %s)", cmdPath, cmdPath)
		}
	}

	return nil
}

// expandGlobPattern expands glob patterns like ~/.openclaw/* or ~/graph-exports/*
func expandGlobPattern(pattern string) ([]string, error) {
	// Expand ~ to home directory
	if strings.HasPrefix(pattern, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		pattern = filepath.Join(homeDir, pattern[1:])
	}

	// If pattern contains glob characters, expand it
	if strings.ContainsAny(pattern, "*?[]") {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern: %w", err)
		}
		return matches, nil
	}

	// Not a glob pattern, return as-is
	return []string{pattern}, nil
}
