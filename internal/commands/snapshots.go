package commands

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"

	"github.com/bulletproof-bot/backup/internal/backup"
	"github.com/bulletproof-bot/backup/internal/config"
	"github.com/bulletproof-bot/backup/internal/types"
	"github.com/spf13/cobra"
)

// NewSnapshotsCommand creates the snapshots command
func NewSnapshotsCommand() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "snapshots",
		Short: "List all backup snapshots",
		Long:  "List all available backup snapshots with timestamps and file counts.",
		RunE: func(c *cobra.Command, args []string) error {
			return runSnapshots(format, args)
		},
	}

	cmd.Flags().StringVar(&format, "format", "text", "Output format: text, json, or csv")

	return cmd
}

func runSnapshots(format string, args []string) error {
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
		if format == "text" {
			fmt.Println("No backups found.")
		} else if format == "json" {
			fmt.Println("[]")
		} else if format == "csv" {
			// Output CSV header even if empty
			w := csv.NewWriter(os.Stdout)
			w.Write([]string{"short_id", "full_id", "timestamp", "message", "file_count"})
			w.Flush()
		}
		return nil
	}

	// Assign short IDs (1=latest, 2=second-latest, etc.)
	shortIDs := types.AssignShortIDs(backups)

	// Output based on format
	switch format {
	case "json":
		return outputJSON(backups, shortIDs)
	case "csv":
		return outputCSV(backups, shortIDs)
	case "text":
		fallthrough
	default:
		return outputText(backups, shortIDs)
	}
}

func outputText(backups []*types.SnapshotInfo, shortIDs map[string]int) error {
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

func outputJSON(backups []*types.SnapshotInfo, shortIDs map[string]int) error {
	type snapshotJSON struct {
		ShortID   int    `json:"short_id"`
		FullID    string `json:"full_id"`
		Timestamp string `json:"timestamp"`
		Message   string `json:"message,omitempty"`
		FileCount int    `json:"file_count"`
	}

	snapshots := make([]snapshotJSON, len(backups))
	for i, b := range backups {
		snapshots[i] = snapshotJSON{
			ShortID:   shortIDs[b.ID],
			FullID:    b.ID,
			Timestamp: b.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
			Message:   b.Message,
			FileCount: b.FileCount,
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(snapshots)
}

func outputCSV(backups []*types.SnapshotInfo, shortIDs map[string]int) error {
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

	// Write header
	if err := w.Write([]string{"short_id", "full_id", "timestamp", "message", "file_count"}); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data
	for _, b := range backups {
		shortID := fmt.Sprintf("%d", shortIDs[b.ID])
		fileCount := fmt.Sprintf("%d", b.FileCount)
		timestamp := b.Timestamp.Format("2006-01-02T15:04:05Z07:00")

		if err := w.Write([]string{shortID, b.ID, timestamp, b.Message, fileCount}); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}
