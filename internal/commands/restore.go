package commands

import (
	"fmt"

	"github.com/bulletproof-bot/backup/internal/analytics"
	"github.com/bulletproof-bot/backup/internal/backup"
	"github.com/bulletproof-bot/backup/internal/config"
	"github.com/spf13/cobra"
)

// NewRestoreCommand creates the restore command
func NewRestoreCommand() *cobra.Command {
	var dryRun bool
	var noScripts bool
	var force bool
	var target string

	cmd := &cobra.Command{
		Use:   "restore <snapshot-id>",
		Short: "Restore from a backup snapshot",
		Long:  "Restore your OpenClaw installation from a specific backup snapshot.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRestore(args[0], dryRun, noScripts, force, target)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be restored without making changes")
	cmd.Flags().BoolVar(&noScripts, "no-scripts", false, "Skip post-restore script execution")
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompts")
	cmd.Flags().StringVar(&target, "target", "", "Restore to alternative location instead of OpenClaw path")

	return cmd
}

func runRestore(snapshotID string, dryRun bool, noScripts bool, force bool, target string) error {
	// Track analytics
	flags := make(map[string]string)
	if dryRun {
		flags["dry-run"] = "true"
	}
	if noScripts {
		flags["no-scripts"] = "true"
	}
	if force {
		flags["force"] = "true"
	}
	if target != "" {
		flags["target"] = "true"
	}
	analytics.TrackCommand("restore", flags)

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

	// Run restore (force flag controls script execution warnings)
	if err := engine.RestoreToTarget(snapshotID, target, dryRun, noScripts, force); err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}

	return nil
}
