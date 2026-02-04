package backup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bulletproof-bot/backup/internal/backup/destinations"
	"github.com/bulletproof-bot/backup/internal/config"
	"github.com/bulletproof-bot/backup/internal/types"
)

// BackupEngine orchestrates backups and restores
type BackupEngine struct {
	config      *config.Config
	destination Destination
}

// NewBackupEngine creates a new backup engine
func NewBackupEngine(cfg *config.Config) (*BackupEngine, error) {
	if cfg.Destination == nil {
		return nil, errors.New("no destination configured. Run: bulletproof init")
	}

	destination, err := createDestination(cfg.Destination)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination: %w", err)
	}

	return &BackupEngine{
		config:      cfg,
		destination: destination,
	}, nil
}

func createDestination(destConfig *config.DestinationConfig) (Destination, error) {
	switch destConfig.Type {
	case "git":
		return destinations.NewGitDestination(destConfig.Path), nil
	case "local":
		return destinations.NewLocalDestination(destConfig.Path, true), nil
	case "sync":
		// Sync destinations work like local - just copy files
		// The sync client (Dropbox/GDrive) handles the rest
		return destinations.NewSyncDestination(destConfig.Path), nil
	default:
		return nil, fmt.Errorf("unknown destination type: %s", destConfig.Type)
	}
}

// OpenclawPath returns the OpenClaw root path
func (e *BackupEngine) OpenclawPath() (string, error) {
	if e.config.OpenclawPath != "" {
		return e.config.OpenclawPath, nil
	}

	detected := config.DetectInstallation()
	if detected != "" {
		return detected, nil
	}

	return "", errors.New("OpenClaw installation not found. Run: bulletproof config set openclaw_path /path/to/.openclaw")
}

// ResolveSnapshotID converts a short numeric ID (1, 2, 3) to a full timestamp ID
// Returns the ID unchanged if it's already a full timestamp ID
// ID "0" is a special case for current filesystem state
func (e *BackupEngine) ResolveSnapshotID(id string) (string, error) {
	// If it's already a full ID or "0", return as-is
	if types.IsFullID(id) || id == "0" {
		return id, nil
	}

	// Get all snapshots to resolve short IDs
	snapshots, err := e.ListBackups()
	if err != nil {
		return "", fmt.Errorf("failed to list backups: %w", err)
	}

	return types.ResolveID(id, snapshots)
}

// Config returns the backup engine's configuration
func (e *BackupEngine) Config() *config.Config {
	return e.config
}

// Destination returns the backup engine's destination
func (e *BackupEngine) Destination() Destination {
	return e.destination
}

// GetSnapshot retrieves a snapshot by ID (supports both short and full IDs)
func (e *BackupEngine) GetSnapshot(id string) (*types.Snapshot, error) {
	// Resolve short ID to full ID
	resolvedID, err := e.ResolveSnapshotID(id)
	if err != nil {
		return nil, err
	}

	// ID 0 is special - it means current state, not a stored snapshot
	if resolvedID == "0" {
		return nil, fmt.Errorf("ID 0 represents current filesystem state, not a stored snapshot")
	}

	return e.destination.GetSnapshot(resolvedID)
}

// Backup runs a backup operation
func (e *BackupEngine) Backup(dryRun bool, message string) (*types.BackupResult, error) {
	openclawPath, err := e.OpenclawPath()
	if err != nil {
		return nil, err
	}

	fmt.Printf("🔍 Scanning OpenClaw installation at: %s\n", openclawPath)

	// Create snapshot of current state
	snapshot, err := types.FromDirectory(
		openclawPath,
		e.config.Options.Exclude,
		message,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot: %w", err)
	}

	fmt.Printf("📦 Found %d files to back up\n", len(snapshot.Files))

	// Get last snapshot for comparison
	lastSnapshot, err := e.destination.GetLastSnapshot()
	if err != nil {
		return nil, fmt.Errorf("failed to get last snapshot: %w", err)
	}

	var diff *types.SnapshotDiff
	if lastSnapshot != nil {
		diff = snapshot.Diff(lastSnapshot)
		fmt.Printf("📊 Changes since last backup: %s\n", diff.String())

		if diff.IsEmpty() {
			fmt.Println("✨ No changes detected. Backup skipped.")
			return &types.BackupResult{
				Snapshot: snapshot,
				Diff:     diff,
				Skipped:  true,
			}, nil
		}
	} else {
		fmt.Println("📝 First backup - no previous snapshot found")
	}

	if dryRun {
		fmt.Println("\n🔍 Dry run - no changes made")
		if diff != nil {
			diff.PrintDetailed()
		}
		return &types.BackupResult{
			Snapshot: snapshot,
			Diff:     diff,
			DryRun:   true,
		}, nil
	}

	// Perform the backup
	fmt.Printf("\n💾 Backing up to: %s\n", e.config.Destination.Path)

	backupMessage := message
	if backupMessage == "" {
		backupMessage = "Backup " + snapshot.ID
	}

	err = e.destination.Save(openclawPath, snapshot, backupMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to save backup: %w", err)
	}

	// Copy config to snapshot for self-contained backups (Phase 2 feature)
	if err := e.copyConfigToSnapshot(snapshot.ID); err != nil {
		// Non-fatal - log but continue
		fmt.Printf("⚠️  Warning: failed to copy config to snapshot: %v\n", err)
	}

	fmt.Printf("✅ Backup complete: %s\n", snapshot.ID)

	return &types.BackupResult{
		Snapshot: snapshot,
		Diff:     diff,
	}, nil
}

// copyConfigToSnapshot copies the config file to the snapshot's .bulletproof directory
func (e *BackupEngine) copyConfigToSnapshot(snapshotID string) error {
	// Determine config source path
	configPath := config.DefaultConfigPath()
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	// Determine snapshot path based on destination type
	var snapshotPath string
	switch dest := e.destination.(type) {
	case *destinations.LocalDestination:
		if dest.Timestamped {
			snapshotPath = filepath.Join(dest.BasePath, snapshotID)
		} else {
			// Sync destinations don't have timestamped folders
			return nil
		}
	case *destinations.GitDestination:
		// Git destinations store in repo root, use .bulletproof there
		localPath := dest.RepoPath
		if strings.HasPrefix(dest.RepoPath, "git@") || strings.HasPrefix(dest.RepoPath, "https://") {
			homeDir, _ := os.UserHomeDir()
			repoName := filepath.Base(strings.TrimSuffix(dest.RepoPath, ".git"))
			localPath = filepath.Join(homeDir, ".cache", "bulletproof", "repos", repoName)
		}
		snapshotPath = localPath
	default:
		return nil
	}

	// Write config to .bulletproof/config.yaml in snapshot
	bulletproofDir := filepath.Join(snapshotPath, ".bulletproof")
	configFile := filepath.Join(bulletproofDir, "config.yaml")
	if err := os.WriteFile(configFile, configData, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// ListBackups returns all available backups
func (e *BackupEngine) ListBackups() ([]*types.SnapshotInfo, error) {
	return e.destination.ListSnapshots()
}

// ShowDiff shows the diff between current state and last backup
func (e *BackupEngine) ShowDiff() (*types.SnapshotDiff, error) {
	openclawPath, err := e.OpenclawPath()
	if err != nil {
		return nil, err
	}

	current, err := types.FromDirectory(
		openclawPath,
		e.config.Options.Exclude,
		"",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create current snapshot: %w", err)
	}

	last, err := e.destination.GetLastSnapshot()
	if err != nil {
		return nil, fmt.Errorf("failed to get last snapshot: %w", err)
	}

	if last == nil {
		fmt.Println("No previous backup found.")
		return nil, nil
	}

	return current.Diff(last), nil
}

// Restore restores from a specific backup
func (e *BackupEngine) Restore(snapshotID string, dryRun bool) error {
	// Resolve short IDs to full timestamp IDs
	resolvedID, err := e.ResolveSnapshotID(snapshotID)
	if err != nil {
		return err
	}

	// Special case: ID 0 means current state (nothing to restore)
	if resolvedID == "0" {
		return fmt.Errorf("cannot restore to ID 0 (current filesystem state)")
	}

	openclawPath, err := e.OpenclawPath()
	if err != nil {
		return err
	}

	// Show both short and full ID if they differ
	if snapshotID != resolvedID {
		fmt.Printf("🔍 Looking for backup: %s (ID %s)\n", resolvedID, snapshotID)
	} else {
		fmt.Printf("🔍 Looking for backup: %s\n", resolvedID)
	}

	snapshot, err := e.destination.GetSnapshot(resolvedID)
	if err != nil {
		return fmt.Errorf("failed to get snapshot: %w", err)
	}

	if snapshot == nil {
		return fmt.Errorf("backup not found: %s", snapshotID)
	}

	fmt.Printf("📦 Found backup with %d files\n", len(snapshot.Files))

	if dryRun {
		fmt.Println("\n🔍 Dry run - would restore these files:")
		count := 0
		for file := range snapshot.Files {
			if count < 20 {
				fmt.Printf("  %s\n", file)
			}
			count++
		}
		if count > 20 {
			fmt.Printf("  ... and %d more\n", count-20)
		}
		return nil
	}

	// Create backup of current state before restore
	fmt.Println("\n⚠️  Creating safety backup before restore...")
	safetyBackup, err := e.Backup(false, "Pre-restore safety backup")
	if err != nil {
		return fmt.Errorf("failed to create safety backup: %w", err)
	}

	if !safetyBackup.Skipped {
		fmt.Printf("📝 Safety backup created: %s\n", safetyBackup.Snapshot.ID)
	}

	// Perform restore
	fmt.Printf("\n🔄 Restoring from %s...\n", snapshotID)
	err = e.destination.Restore(snapshotID, openclawPath)
	if err != nil {
		return fmt.Errorf("failed to restore: %w", err)
	}

	fmt.Println("✅ Restore complete!")
	if !safetyBackup.Skipped {
		fmt.Printf("💡 If something went wrong, restore from: %s\n", safetyBackup.Snapshot.ID)
	}

	return nil
}
