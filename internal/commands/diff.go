package commands

import (
	"fmt"
	"path/filepath"

	"github.com/bulletproof-bot/backup/internal/backup"
	"github.com/bulletproof-bot/backup/internal/config"
	"github.com/bulletproof-bot/backup/internal/types"
	"github.com/spf13/cobra"
)

// NewDiffCommand creates the diff command
func NewDiffCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "diff [snapshot1] [snapshot2] [pattern]",
		Short: "Show changes between snapshots",
		Long: `Show changes between snapshots or current state.

Usage:
  bulletproof diff                    # Compare current state to last backup
  bulletproof diff 5                  # Compare current state to snapshot 5
  bulletproof diff 10 5               # Compare snapshot 10 to snapshot 5
  bulletproof diff 10 5 SOUL.md       # Compare specific file between snapshots
  bulletproof diff 10 5 'skills/*.js' # Compare files matching pattern

Snapshot IDs:
  0           Current filesystem state
  1, 2, 3...  Short IDs (1=latest, 2=second-latest, etc.)
  yyyyMMdd-HHmmss  Full timestamp IDs also accepted`,
		RunE: runDiff,
	}
}

func runDiff(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Create backup engine
	engine, err := backup.NewBackupEngine(cfg)
	if err != nil {
		return err
	}

	// Parse arguments based on count
	var diff *types.SnapshotDiff
	var from, to *types.Snapshot
	var pattern string

	switch len(args) {
	case 0:
		// No args: current vs last backup
		diff, from, to, err = diffCurrentVsLast(engine)
		if err != nil {
			return err
		}

	case 1:
		// 1 arg: current (ID 0) vs specified snapshot
		diff, from, to, err = diffCurrentVsSnapshotWithSnapshots(engine, args[0])
		if err != nil {
			return err
		}

	case 2:
		// 2 args: snapshot1 vs snapshot2
		diff, from, to, err = diffSnapshotVsSnapshotWithSnapshots(engine, args[0], args[1])
		if err != nil {
			return err
		}

	case 3:
		// 3 args: snapshot1 vs snapshot2 with pattern filter
		diff, from, to, err = diffSnapshotVsSnapshotWithSnapshots(engine, args[0], args[1])
		if err != nil {
			return err
		}
		pattern = args[2]

	default:
		return fmt.Errorf("too many arguments (expected 0-3, got %d)", len(args))
	}

	if diff == nil {
		fmt.Println("No differences found.")
		return nil
	}

	// Apply pattern filter if specified
	if pattern != "" {
		diff = filterDiffByPattern(diff, pattern)
	}

	// Display diff in unified format
	diff.PrintUnified(from, to)

	return nil
}

// diffCurrentVsLast compares current state to last backup
func diffCurrentVsLast(engine *backup.BackupEngine) (*types.SnapshotDiff, *types.Snapshot, *types.Snapshot, error) {
	openclawPath, err := engine.OpenclawPath()
	if err != nil {
		return nil, nil, nil, err
	}

	current, err := types.FromDirectory(
		openclawPath,
		engine.Config().Options.Exclude,
		"",
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to scan current state: %w", err)
	}

	last, err := engine.Destination().GetLastSnapshot()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get last snapshot: %w", err)
	}

	if last == nil {
		fmt.Println("No previous backup found.")
		return nil, nil, nil, nil
	}

	return current.Diff(last), current, last, nil
}

// diffCurrentVsSnapshotWithSnapshots compares current filesystem state to a specific snapshot
func diffCurrentVsSnapshotWithSnapshots(engine *backup.BackupEngine, snapshotID string) (*types.SnapshotDiff, *types.Snapshot, *types.Snapshot, error) {
	// Resolve short ID to full ID
	resolvedID, err := engine.ResolveSnapshotID(snapshotID)
	if err != nil {
		return nil, nil, nil, err
	}

	// Get OpenClaw path
	openclawPath, err := engine.OpenclawPath()
	if err != nil {
		return nil, nil, nil, err
	}

	// Create snapshot of current state
	current, err := types.FromDirectory(
		openclawPath,
		engine.Config().Options.Exclude,
		"",
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to scan current state: %w", err)
	}

	// Get the target snapshot
	snapshot, err := engine.GetSnapshot(resolvedID)
	if err != nil {
		return nil, nil, nil, err
	}

	return current.Diff(snapshot), current, snapshot, nil
}

// diffSnapshotVsSnapshotWithSnapshots compares two snapshots
func diffSnapshotVsSnapshotWithSnapshots(engine *backup.BackupEngine, id1, id2 string) (*types.SnapshotDiff, *types.Snapshot, *types.Snapshot, error) {
	// Resolve short IDs to full IDs
	resolvedID1, err := engine.ResolveSnapshotID(id1)
	if err != nil {
		return nil, nil, nil, err
	}

	resolvedID2, err := engine.ResolveSnapshotID(id2)
	if err != nil {
		return nil, nil, nil, err
	}

	// Special case: ID 0 means current state
	var snapshot1 *types.Snapshot
	if resolvedID1 == "0" {
		openclawPath, err := engine.OpenclawPath()
		if err != nil {
			return nil, nil, nil, err
		}
		snapshot1, err = types.FromDirectory(
			openclawPath,
			engine.Config().Options.Exclude,
			"",
		)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to scan current state: %w", err)
		}
	} else {
		snapshot1, err = engine.GetSnapshot(resolvedID1)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	// Get second snapshot (cannot be ID 0)
	if resolvedID2 == "0" {
		return nil, nil, nil, fmt.Errorf("second snapshot ID cannot be 0 (current state)")
	}

	snapshot2, err := engine.GetSnapshot(resolvedID2)
	if err != nil {
		return nil, nil, nil, err
	}

	return snapshot1.Diff(snapshot2), snapshot1, snapshot2, nil
}

// filterDiffByPattern filters diff results to only include files matching pattern
func filterDiffByPattern(diff *types.SnapshotDiff, pattern string) *types.SnapshotDiff {
	filtered := &types.SnapshotDiff{
		From:     diff.From,
		To:       diff.To,
		Added:    []string{},
		Removed:  []string{},
		Modified: []string{},
	}

	// Filter added files
	for _, path := range diff.Added {
		if matchesPattern(path, pattern) {
			filtered.Added = append(filtered.Added, path)
		}
	}

	// Filter removed files
	for _, path := range diff.Removed {
		if matchesPattern(path, pattern) {
			filtered.Removed = append(filtered.Removed, path)
		}
	}

	// Filter modified files
	for _, path := range diff.Modified {
		if matchesPattern(path, pattern) {
			filtered.Modified = append(filtered.Modified, path)
		}
	}

	return filtered
}

// matchesPattern checks if a file path matches the given pattern (glob or exact match)
func matchesPattern(path, pattern string) bool {
	// Try exact match first
	if path == pattern {
		return true
	}

	// Try glob match
	matched, err := filepath.Match(pattern, filepath.Base(path))
	if err == nil && matched {
		return true
	}

	// Try full path glob match
	matched, err = filepath.Match(pattern, path)
	if err == nil && matched {
		return true
	}

	// Try substring match (e.g., "SOUL.md" matches "workspace/SOUL.md")
	if filepath.Base(path) == pattern {
		return true
	}

	return false
}
