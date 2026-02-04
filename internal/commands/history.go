package commands

import (
	"fmt"

	"github.com/bulletproof-bot/backup/internal/backup"
	"github.com/bulletproof-bot/backup/internal/config"
	"github.com/bulletproof-bot/backup/internal/types"
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

	// Assign short IDs (1=latest, 2=second-latest, etc.)
	shortIDs := types.AssignShortIDs(backups)

	fmt.Println("Available backups (ID 0 = current filesystem state):")
	fmt.Println()

	// Display in order (newest first)
	for i, b := range backups {
		shortID := shortIDs[b.ID]
		// Format: ID [timestamp] - message (N files)
		msg := ""
		if b.Message != "" {
			msg = fmt.Sprintf(" - %s", b.Message)
		}
		fmt.Printf("  [%d] %s%s (%d files)\n", shortID, b.Timestamp.Format("2006-01-02 15:04:05"), msg, b.FileCount)

		// Add a blank line between entries for readability
		if i < len(backups)-1 {
			fmt.Println()
		}
	}

	return nil
}
