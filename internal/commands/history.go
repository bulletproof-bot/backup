package commands

import (
	"fmt"

	"github.com/bulletproof-bot/backup/internal/backup"
	"github.com/bulletproof-bot/backup/internal/config"
	"github.com/spf13/cobra"
)

// NewHistoryCommand creates the history command
func NewHistoryCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "history",
		Short: "List all backup snapshots",
		Long:  "List all available backup snapshots with timestamps and file counts.",
		RunE:  runHistory,
	}
}

func runHistory(cmd *cobra.Command, args []string) error {
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

	// Get backups
	backups, err := engine.ListBackups()
	if err != nil {
		return err
	}

	if len(backups) == 0 {
		fmt.Println("No backups found.")
		return nil
	}

	fmt.Println("Available backups:")
	for _, b := range backups {
		fmt.Printf("  %s\n", b.String())
	}

	return nil
}
