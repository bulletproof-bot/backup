package commands

import (
	"github.com/bulletproof-bot/backup/internal/backup"
	"github.com/bulletproof-bot/backup/internal/config"
	"github.com/spf13/cobra"
)

// NewBackupCommand creates the backup command
func NewBackupCommand() *cobra.Command {
	var dryRun bool
	var message string

	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Create a backup snapshot",
		Long:  "Create a backup snapshot of your OpenClaw installation.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBackup(dryRun, message)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be backed up without making changes")
	cmd.Flags().StringVarP(&message, "message", "m", "", "Backup message")

	return cmd
}

func runBackup(dryRun bool, message string) error {
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
	_, err = engine.Backup(dryRun, message)
	return err
}
