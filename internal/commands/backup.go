package commands

import (
	"github.com/bulletproof-bot/backup/internal/analytics"
	"github.com/bulletproof-bot/backup/internal/backup"
	"github.com/bulletproof-bot/backup/internal/config"
	"github.com/spf13/cobra"
)

// NewBackupCommand creates the backup command
func NewBackupCommand() *cobra.Command {
	var dryRun bool
	var message string
	var noScripts bool
	var force bool

	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Create a backup snapshot",
		Long:  "Create a backup snapshot of your OpenClaw installation.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBackup(dryRun, message, noScripts, force)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be backed up without making changes")
	cmd.Flags().StringVarP(&message, "message", "m", "", "Backup message")
	cmd.Flags().BoolVar(&noScripts, "no-scripts", false, "Skip pre-backup script execution")
	cmd.Flags().BoolVar(&force, "force", false, "Force backup even if no changes detected")

	return cmd
}

func runBackup(dryRun bool, message string, noScripts bool, force bool) error {
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
	analytics.TrackCommand("backup", flags)

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

	// Run backup
	_, err = engine.Backup(dryRun, message, noScripts, force)
	return err
}
