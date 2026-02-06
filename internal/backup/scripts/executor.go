package scripts

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/bulletproof-bot/backup/internal/utils"
)

// ScriptConfig represents a single script configuration
type ScriptConfig struct {
	Name    string `yaml:"name"`
	Command string `yaml:"command"`
	Timeout int    `yaml:"timeout"` // seconds, 0 = default (60s)
}

// ExecutionContext provides environment information to scripts
type ExecutionContext struct {
	SnapshotID   string
	OpenClawPath string
	BackupDir    string
	ExportsDir   string
}

// Executor runs pre-backup and post-restore scripts
type Executor struct {
	scripts []ScriptConfig
	ctx     ExecutionContext
}

// NewExecutor creates a script executor
func NewExecutor(scripts []ScriptConfig, ctx ExecutionContext) *Executor {
	return &Executor{
		scripts: scripts,
		ctx:     ctx,
	}
}

// Execute runs all configured scripts with timeout handling
func (e *Executor) Execute() error {
	if len(e.scripts) == 0 {
		return nil
	}

	for _, script := range e.scripts {
		if err := e.executeScript(script); err != nil {
			return fmt.Errorf("script '%s' failed: %w", script.Name, err)
		}
	}

	return nil
}

// executeScript runs a single script with environment variable substitution
func (e *Executor) executeScript(script ScriptConfig) error {
	// Substitute environment variables
	command := e.substituteVariables(script.Command)

	// Parse command (shell and arguments)
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	// Determine timeout
	timeout := time.Duration(script.Timeout) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second // default
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Execute command
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("SNAPSHOT_ID=%s", e.ctx.SnapshotID),
		fmt.Sprintf("OPENCLAW_PATH=%s", e.ctx.OpenClawPath),
		fmt.Sprintf("BACKUP_DIR=%s", e.ctx.BackupDir),
		fmt.Sprintf("EXPORTS_DIR=%s", e.ctx.ExportsDir),
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("timeout after %v", timeout)
	}

	// Check for execution error
	if err != nil {
		return fmt.Errorf("%w\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}

	return nil
}

// substituteVariables replaces environment variable placeholders
func (e *Executor) substituteVariables(command string) string {
	replacements := map[string]string{
		"$SNAPSHOT_ID":   e.ctx.SnapshotID,
		"$OPENCLAW_PATH": e.ctx.OpenClawPath,
		"$BACKUP_DIR":    e.ctx.BackupDir,
		"$EXPORTS_DIR":   e.ctx.ExportsDir,
	}

	result := command
	for placeholder, value := range replacements {
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}

// CreateExportsDir creates the _exports directory for script outputs
func CreateExportsDir(basePath string) (string, error) {
	exportsDir := filepath.Join(basePath, "_exports")
	if err := os.MkdirAll(exportsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create exports directory: %w", err)
	}
	return exportsDir, nil
}

// CopyExportsToSnapshot copies _exports directory to snapshot
func CopyExportsToSnapshot(exportsDir, snapshotPath string) error {
	// Check if exports directory exists and has content
	entries, err := os.ReadDir(exportsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No exports to copy
		}
		return fmt.Errorf("failed to read exports directory: %w", err)
	}

	if len(entries) == 0 {
		return nil // Empty exports directory
	}

	// Create _exports in snapshot
	targetDir := filepath.Join(snapshotPath, "_exports")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create exports directory in snapshot: %w", err)
	}

	// Copy all files
	for _, entry := range entries {
		srcPath := filepath.Join(exportsDir, entry.Name())
		dstPath := filepath.Join(targetDir, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return fmt.Errorf("failed to copy exports subdirectory: %w", err)
			}
		} else {
			if err := utils.CopyFile(srcPath, dstPath); err != nil {
				return fmt.Errorf("failed to copy exports file: %w", err)
			}
		}
	}

	return nil
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := utils.CopyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// CopyConfigToSnapshot copies the bulletproof config to snapshot's .bulletproof directory
func CopyConfigToSnapshot(configPath, snapshotPath string) error {
	// Check if config file exists
	if _, err := os.Stat(configPath); err != nil {
		if os.IsNotExist(err) {
			return nil // No config to copy
		}
		return fmt.Errorf("failed to stat config file: %w", err)
	}

	// Create .bulletproof directory in snapshot
	bulletproofDir := filepath.Join(snapshotPath, ".bulletproof")
	if err := os.MkdirAll(bulletproofDir, 0755); err != nil {
		return fmt.Errorf("failed to create .bulletproof directory: %w", err)
	}

	// Copy config.yaml
	dstPath := filepath.Join(bulletproofDir, "config.yaml")
	if err := utils.CopyFile(configPath, dstPath); err != nil {
		return fmt.Errorf("failed to copy config file: %w", err)
	}

	return nil
}

// CopyScriptsToSnapshot copies scripts directory to snapshot's .bulletproof/scripts directory
func CopyScriptsToSnapshot(scriptsDir, snapshotPath string) error {
	// Check if scripts directory exists
	entries, err := os.ReadDir(scriptsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No scripts to copy
		}
		return fmt.Errorf("failed to read scripts directory: %w", err)
	}

	if len(entries) == 0 {
		return nil // Empty scripts directory
	}

	// Create .bulletproof/scripts directory in snapshot
	targetDir := filepath.Join(snapshotPath, ".bulletproof", "scripts")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create scripts directory in snapshot: %w", err)
	}

	// Copy all scripts recursively
	if err := copyDir(scriptsDir, targetDir); err != nil {
		return fmt.Errorf("failed to copy scripts: %w", err)
	}

	return nil
}
