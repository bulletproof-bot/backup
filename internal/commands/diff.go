package commands

import (
	"github.com/bulletproof-bot/backup/internal/backup"
	"github.com/bulletproof-bot/backup/internal/config"
	"github.com/spf13/cobra"
)

// NewDiffCommand creates the diff command
func NewDiffCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "diff",
		Short: "Show changes since last backup",
		Long:  "Show what has changed in your OpenClaw installation since the last backup.",
		RunE:  runDiff,
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

	// Show diff
	diff, err := engine.ShowDiff()
	if err != nil {
		return err
	}

	if diff != nil {
		diff.PrintDetailed()
	}

	return nil
}
