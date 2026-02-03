package commands

import (
	"fmt"

	"github.com/bulletproof-bot/backup/internal/backup"
	"github.com/bulletproof-bot/backup/internal/config"
	"github.com/spf13/cobra"
)

// NewRestoreCommand creates the restore command
func NewRestoreCommand() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "restore <snapshot-id>",
		Short: "Restore from a backup snapshot",
		Long:  "Restore your OpenClaw installation from a specific backup snapshot.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRestore(args[0], dryRun)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be restored without making changes")

	return cmd
}

func runRestore(snapshotID string, dryRun bool) error {
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

	// Run restore
	if err := engine.Restore(snapshotID, dryRun); err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}

	return nil
}
