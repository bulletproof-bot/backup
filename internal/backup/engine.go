package backup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bulletproof-bot/backup/internal/backup/destinations"
	"github.com/bulletproof-bot/backup/internal/backup/scripts"
	"github.com/bulletproof-bot/backup/internal/config"
	"github.com/bulletproof-bot/backup/internal/types"
	"github.com/bulletproof-bot/backup/internal/utils"
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

// getSourcePaths returns all source paths to back up, with glob expansion
func (e *BackupEngine) getSourcePaths() ([]string, error) {
	sources := e.config.GetSources()
	if len(sources) == 0 {
		// Try auto-detection as fallback
		detected := config.DetectInstallation()
		if detected != "" {
			return []string{detected}, nil
		}
		return nil, nil
	}

	// Expand glob patterns in sources
	expandedSources := []string{}
	for _, source := range sources {
		// Expand ~ to home directory
		if strings.HasPrefix(source, "~") {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to get home directory: %w", err)
			}
			source = filepath.Join(homeDir, source[1:])
		}

		// If pattern contains glob characters, expand it
		if strings.ContainsAny(source, "*?[]") {
			matches, err := filepath.Glob(source)
			if err != nil {
				return nil, fmt.Errorf("invalid glob pattern %s: %w", source, err)
			}
			if len(matches) == 0 {
				return nil, fmt.Errorf("glob pattern matches no paths: %s", source)
			}
			expandedSources = append(expandedSources, matches...)
		} else {
			expandedSources = append(expandedSources, source)
		}
	}

	// Validate that all paths exist and are directories
	for _, source := range expandedSources {
		info, err := os.Stat(source)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("source path does not exist: %s", source)
			}
			return nil, fmt.Errorf("failed to check source path %s: %w", source, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("source path is not a directory: %s", source)
		}
	}

	return expandedSources, nil
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
func (e *BackupEngine) Backup(dryRun bool, message string, noScripts bool, force bool) (*types.BackupResult, error) {
	// Get all source paths (supports multi-source backups)
	sources, err := e.getSourcePaths()
	if err != nil {
		return nil, err
	}

	if len(sources) == 0 {
		return nil, errors.New("no source paths configured. Run: bulletproof config set openclaw_path /path/to/.openclaw")
	}

	// Display sources being backed up
	if len(sources) == 1 {
		fmt.Printf("🔍 Scanning source at: %s\n", sources[0])
	} else {
		fmt.Printf("🔍 Scanning %d sources:\n", len(sources))
		for _, source := range sources {
			fmt.Printf("  • %s\n", source)
		}
	}

	// Generate snapshot ID early so it's available to pre-backup scripts
	snapshotTimestamp := time.Now()
	snapshotID := types.GenerateID(snapshotTimestamp)

	// Execute pre-backup scripts (unless disabled)
	var exportsDir string
	if !noScripts && len(e.config.Scripts.PreBackup) > 0 {
		fmt.Println("\n📜 Executing pre-backup scripts...")

		// Create _exports directory
		configDir, err := config.ConfigDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get config directory: %w", err)
		}
		exportsDir, err = scripts.CreateExportsDir(configDir)
		if err != nil {
			return nil, fmt.Errorf("failed to create exports directory: %w", err)
		}

		// Execute scripts (use first source as OpenClawPath for backward compatibility)
		executor := scripts.NewExecutor(
			convertScriptConfigs(e.config.Scripts.PreBackup),
			scripts.ExecutionContext{
				SnapshotID:   snapshotID,
				OpenClawPath: sources[0],
				BackupDir:    e.config.Destination.Path,
				ExportsDir:   exportsDir,
			},
		)

		if err := executor.Execute(); err != nil {
			return nil, fmt.Errorf("pre-backup script failed: %w", err)
		}

		fmt.Println("✅ Pre-backup scripts completed")
	}

	// Create snapshots for each source (use the same timestamp for consistency)
	var snapshot *types.Snapshot
	if len(sources) == 1 {
		// Single source - create snapshot directly
		snapshot, err = types.FromDirectoryWithTimestamp(
			sources[0],
			e.config.Options.Exclude,
			message,
			snapshotTimestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create snapshot: %w", err)
		}
	} else {
		// Multiple sources - create individual snapshots and merge
		snapshots := make([]*types.Snapshot, len(sources))
		for i, source := range sources {
			s, err := types.FromDirectoryWithTimestamp(
				source,
				e.config.Options.Exclude,
				"",
				snapshotTimestamp,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to create snapshot for %s: %w", source, err)
			}
			snapshots[i] = s
		}

		// Merge all snapshots into one
		snapshot, err = types.MergeWithSources(snapshots, sources, message, snapshotTimestamp)
		if err != nil {
			return nil, fmt.Errorf("failed to merge snapshots: %w", err)
		}
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

		if diff.IsEmpty() && !force {
			fmt.Println("✨ No changes detected. Backup skipped.")
			fmt.Println("💡 Use --force flag to create backup anyway")
			return &types.BackupResult{
				Snapshot: snapshot,
				Diff:     diff,
				Skipped:  true,
			}, nil
		}
		if diff.IsEmpty() && force {
			fmt.Println("⚠️  No changes detected, but --force specified. Creating backup anyway.")
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

	// Save based on number of sources
	if len(sources) == 1 {
		// Single source - use traditional Save method
		err = e.destination.Save(sources[0], snapshot, backupMessage)
		if err != nil {
			return nil, fmt.Errorf("failed to save backup: %w", err)
		}
	} else {
		// Multi-source - save each source separately
		if err := e.saveMultiSource(sources, snapshot, backupMessage); err != nil {
			return nil, fmt.Errorf("failed to save multi-source backup: %w", err)
		}
	}

	// Copy config to snapshot for self-contained backups
	if err := e.copyConfigToSnapshot(snapshot.ID); err != nil {
		// Non-fatal - log but continue
		fmt.Printf("⚠️  Warning: failed to copy config to snapshot: %v\n", err)
	}

	// Copy scripts to snapshot for self-contained backups
	if err := e.copyScriptsToSnapshot(snapshot.ID); err != nil {
		// Non-fatal - log but continue
		fmt.Printf("⚠️  Warning: failed to copy scripts to snapshot: %v\n", err)
	}

	// Copy exports directory to snapshot if scripts were executed
	if exportsDir != "" {
		snapshotPath, err := e.getSnapshotPath(snapshot.ID)
		if err == nil && snapshotPath != "" {
			if err := scripts.CopyExportsToSnapshot(exportsDir, snapshotPath); err != nil {
				fmt.Printf("⚠️  Warning: failed to copy exports to snapshot: %v\n", err)
			}
		}
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
	configPath, err := config.ConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}
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

// copyScriptsToSnapshot copies the scripts directory to the snapshot's .bulletproof/scripts directory
func (e *BackupEngine) copyScriptsToSnapshot(snapshotID string) error {
	// Determine scripts source path
	configDir, err := config.ConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}
	scriptsDir := filepath.Join(configDir, "scripts")

	// Check if scripts directory exists
	if _, err := os.Stat(scriptsDir); os.IsNotExist(err) {
		return nil // No scripts to copy
	}

	// Determine snapshot path based on destination type
	snapshotPath, err := e.getSnapshotPath(snapshotID)
	if err != nil {
		return err
	}
	if snapshotPath == "" {
		return nil // Destination type doesn't support self-contained structure
	}

	// Use the helper function from scripts package
	return scripts.CopyScriptsToSnapshot(scriptsDir, snapshotPath)
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

// RestoreToTarget restores from a specific backup to a target location
// If target is empty, restores to the configured OpenClaw path
func (e *BackupEngine) RestoreToTarget(snapshotID string, target string, dryRun bool, noScripts bool, force bool) error {
	// Resolve short IDs to full timestamp IDs
	resolvedID, err := e.ResolveSnapshotID(snapshotID)
	if err != nil {
		return err
	}

	// Special case: ID 0 means current state (nothing to restore)
	if resolvedID == "0" {
		return fmt.Errorf("cannot restore to ID 0 (current filesystem state)")
	}

	// Determine restore target
	var openclawPath string
	if target != "" {
		openclawPath = target
		fmt.Printf("🎯 Restoring to alternative location: %s\n", target)
	} else {
		openclawPath, err = e.OpenclawPath()
		if err != nil {
			return err
		}
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

	// Show changes and ask for confirmation (unless force is set)
	if !force {
		// Create current snapshot to diff against
		currentSnapshot, err := types.FromDirectory(openclawPath, e.config.Options.Exclude, "")
		if err != nil {
			return fmt.Errorf("failed to create current snapshot for comparison: %w", err)
		}

		// Calculate diff
		diff := snapshot.Diff(currentSnapshot)

		if !diff.IsEmpty() {
			fmt.Println("\n📋 Changes that will be applied:")
			if len(diff.Added) > 0 {
				fmt.Printf("  + %d files will be removed (currently exist, not in backup)\n", len(diff.Added))
			}
			if len(diff.Removed) > 0 {
				fmt.Printf("  + %d files will be added (in backup, don't exist currently)\n", len(diff.Removed))
			}
			if len(diff.Modified) > 0 {
				fmt.Printf("  ~ %d files will be modified\n", len(diff.Modified))
			}

			// Show sample files
			fmt.Println()
			sampleCount := 0
			maxSamples := 10

			if len(diff.Removed) > 0 {
				fmt.Println("Files to be added:")
				for _, filePath := range diff.Removed {
					if sampleCount >= maxSamples {
						fmt.Printf("  ... and %d more\n", len(diff.Removed)-maxSamples)
						break
					}
					fmt.Printf("  + %s\n", filePath)
					sampleCount++
				}
				fmt.Println()
			}

			sampleCount = 0
			if len(diff.Modified) > 0 {
				fmt.Println("Files to be modified:")
				for _, filePath := range diff.Modified {
					if sampleCount >= maxSamples {
						fmt.Printf("  ... and %d more\n", len(diff.Modified)-maxSamples)
						break
					}
					fmt.Printf("  ~ %s\n", filePath)
					sampleCount++
				}
				fmt.Println()
			}

			sampleCount = 0
			if len(diff.Added) > 0 {
				fmt.Println("Files to be removed:")
				for _, filePath := range diff.Added {
					if sampleCount >= maxSamples {
						fmt.Printf("  ... and %d more\n", len(diff.Added)-maxSamples)
						break
					}
					fmt.Printf("  - %s\n", filePath)
					sampleCount++
				}
				fmt.Println()
			}

			fmt.Print("⚠️  This will overwrite your current files. Are you sure? [y/N]: ")
			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				fmt.Println("❌ Restore cancelled.")
				fmt.Println("💡 Use --force flag to skip this confirmation prompt")
				return nil
			}
		} else {
			fmt.Println("\n✨ No changes detected - current state matches backup exactly.")
			fmt.Println("💡 Proceeding with restore anyway to ensure consistency.")
		}
	}

	// Create backup of current state before restore
	fmt.Println("\n⚠️  Creating safety backup before restore...")
	safetyBackup, err := e.Backup(false, "Pre-restore safety backup", noScripts, false)
	if err != nil {
		return fmt.Errorf("failed to create safety backup: %w", err)
	}

	if !safetyBackup.Skipped {
		fmt.Printf("📝 Safety backup created: %s\n", safetyBackup.Snapshot.ID)
	}

	// Perform restore
	fmt.Printf("\n🔄 Restoring from %s...\n", snapshotID)
	err = e.destination.Restore(resolvedID, openclawPath)
	if err != nil {
		return fmt.Errorf("failed to restore: %w", err)
	}

	fmt.Println("✅ Restore complete!")
	if !safetyBackup.Skipped {
		fmt.Printf("💡 If something went wrong, restore from: %s\n", safetyBackup.Snapshot.ID)
	}

	// Execute post-restore scripts (unless disabled)
	if !noScripts && len(e.config.Scripts.PostRestore) > 0 {
		// Show security warning unless force is enabled
		if !force {
			fmt.Println("\n⚠️  SECURITY WARNING")
			fmt.Println("╭─────────────────────────────────────────────────────────────╮")
			fmt.Println("│ This backup contains post-restore scripts that will execute │")
			fmt.Println("│ with your system permissions. Scripts from untrusted        │")
			fmt.Println("│ sources can:                                                │")
			fmt.Println("│   • Access your files and data                              │")
			fmt.Println("│   • Execute arbitrary commands                              │")
			fmt.Println("│   • Install backdoors or malware                            │")
			fmt.Println("│                                                              │")
			fmt.Println("│ Scripts to be executed:                                     │")
			for _, script := range e.config.Scripts.PostRestore {
				fmt.Printf("│   • %s: %s\n", script.Name, script.Command)
			}
			fmt.Println("│                                                              │")
			fmt.Println("│ Safety options:                                             │")
			fmt.Println("│   • Use --no-scripts to skip script execution               │")
			fmt.Println("│   • Review scripts in .bulletproof/scripts/ first           │")
			fmt.Println("│   • Only use --force for verified trusted backups           │")
			fmt.Println("╰─────────────────────────────────────────────────────────────╯")
			fmt.Print("\nDo you want to proceed with script execution? [y/N]: ")

			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				fmt.Println("❌ Script execution cancelled. Restore completed without scripts.")
				fmt.Println("💡 Use --no-scripts flag to skip scripts automatically")
				return nil
			}
		}

		fmt.Println("\n📜 Executing post-restore scripts...")

		// Create _exports directory
		configDir, err := config.ConfigDir()
		if err != nil {
			return fmt.Errorf("failed to get config directory: %w", err)
		}
		exportsDir, err := scripts.CreateExportsDir(configDir)
		if err != nil {
			return fmt.Errorf("failed to create exports directory: %w", err)
		}

		// Get snapshot directory path (where _exports is located)
		snapshotDir := filepath.Join(e.config.Destination.Path, resolvedID)

		// Execute scripts
		executor := scripts.NewExecutor(
			convertScriptConfigs(e.config.Scripts.PostRestore),
			scripts.ExecutionContext{
				SnapshotID:   resolvedID,
				OpenClawPath: openclawPath,
				BackupDir:    snapshotDir,
				ExportsDir:   exportsDir,
			},
		)

		if err := executor.Execute(); err != nil {
			return fmt.Errorf("post-restore script failed: %w", err)
		}

		fmt.Println("✅ Post-restore scripts completed")
	}

	return nil
}

// Restore restores from a specific backup to the configured OpenClaw path
func (e *BackupEngine) Restore(snapshotID string, dryRun bool, noScripts bool) error {
	return e.RestoreToTarget(snapshotID, "", dryRun, noScripts, false)
}

// convertScriptConfigs converts config.ScriptConfig to scripts.ScriptConfig
func convertScriptConfigs(configs []config.ScriptConfig) []scripts.ScriptConfig {
	result := make([]scripts.ScriptConfig, len(configs))
	for i, cfg := range configs {
		result[i] = scripts.ScriptConfig{
			Name:    cfg.Name,
			Command: cfg.Command,
			Timeout: cfg.Timeout,
		}
	}
	return result
}

// getSnapshotPath returns the filesystem path for a snapshot ID
func (e *BackupEngine) getSnapshotPath(snapshotID string) (string, error) {
	switch dest := e.destination.(type) {
	case *destinations.LocalDestination:
		if dest.Timestamped {
			return filepath.Join(dest.BasePath, snapshotID), nil
		}
		return "", nil
	case *destinations.GitDestination:
		// Git destinations store in repo root
		localPath := dest.RepoPath
		if strings.HasPrefix(dest.RepoPath, "git@") || strings.HasPrefix(dest.RepoPath, "https://") {
			homeDir, _ := os.UserHomeDir()
			repoName := filepath.Base(strings.TrimSuffix(dest.RepoPath, ".git"))
			localPath = filepath.Join(homeDir, ".cache", "bulletproof", "repos", repoName)
		}
		return localPath, nil
	default:
		return "", nil
	}
}

// saveMultiSource saves a multi-source backup by copying files from each source
// The snapshot contains files with prefixed paths (e.g., "source_0/file.txt")
func (e *BackupEngine) saveMultiSource(sources []string, snapshot *types.Snapshot, message string) error {
	// Get the destination path where we'll save files
	var destBasePath string
	switch dest := e.destination.(type) {
	case *destinations.LocalDestination:
		if dest.Timestamped {
			destBasePath = filepath.Join(dest.BasePath, snapshot.ID)
		} else {
			destBasePath = dest.BasePath
		}
	case *destinations.GitDestination:
		destBasePath = dest.RepoPath
		if strings.HasPrefix(dest.RepoPath, "git@") || strings.HasPrefix(dest.RepoPath, "https://") {
			homeDir, _ := os.UserHomeDir()
			repoName := filepath.Base(strings.TrimSuffix(dest.RepoPath, ".git"))
			destBasePath = filepath.Join(homeDir, ".cache", "bulletproof", "repos", repoName)
		}
	default:
		return fmt.Errorf("unsupported destination type for multi-source backup")
	}

	// Create destination directory
	if err := os.MkdirAll(destBasePath, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Validate no duplicate source basenames (would cause wrong file restoration)
	basenames := make(map[string]string)
	for _, src := range sources {
		base := filepath.Base(src)
		if existing, ok := basenames[base]; ok {
			return fmt.Errorf("duplicate source basenames: %s and %s both have basename %q - cannot determine correct source for files", existing, src, base)
		}
		basenames[base] = src
	}

	// Copy files from each source
	fmt.Printf("  Copying %d files from %d sources...\n", len(snapshot.Files), len(sources))
	for _, fileSnapshot := range snapshot.Files {
		// Extract source index from path prefix (e.g., "openclaw/file.txt" -> "openclaw")
		parts := strings.SplitN(fileSnapshot.Path, string(filepath.Separator), 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid file path format: %s", fileSnapshot.Path)
		}
		sourceBaseName := parts[0]
		relativeFilePath := parts[1]

		// Find the source path that matches this base name
		var sourcePath string
		for _, src := range sources {
			if filepath.Base(src) == sourceBaseName {
				sourcePath = src
				break
			}
		}
		if sourcePath == "" {
			return fmt.Errorf("could not find source for base name: %s", sourceBaseName)
		}

		// Copy the file
		sourceFile := filepath.Join(sourcePath, relativeFilePath)
		destFile := filepath.Join(destBasePath, fileSnapshot.Path)

		// Create parent directory
		if err := os.MkdirAll(filepath.Dir(destFile), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", destFile, err)
		}

		// Copy file
		if err := utils.CopyFile(sourceFile, destFile); err != nil {
			return fmt.Errorf("failed to copy file %s: %w", fileSnapshot.Path, err)
		}
	}

	// Save snapshot metadata
	if err := e.saveSnapshotMetadata(destBasePath, snapshot); err != nil {
		return fmt.Errorf("failed to save snapshot metadata: %w", err)
	}

	return nil
}

// saveSnapshotMetadata saves the snapshot.json file
func (e *BackupEngine) saveSnapshotMetadata(basePath string, snapshot *types.Snapshot) error {
	bulletproofDir := filepath.Join(basePath, ".bulletproof")
	if err := os.MkdirAll(bulletproofDir, 0755); err != nil {
		return fmt.Errorf("failed to create .bulletproof directory: %w", err)
	}

	snapshotFile := filepath.Join(bulletproofDir, "snapshot.json")
	snapshotJSON, err := snapshot.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	if err := os.WriteFile(snapshotFile, snapshotJSON, 0644); err != nil {
		return fmt.Errorf("failed to write snapshot file: %w", err)
	}

	return nil
}
