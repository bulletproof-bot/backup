package main

import (
	"fmt"
	"os"

	"github.com/bulletproof-bot/backup/internal/commands"
	"github.com/bulletproof-bot/backup/internal/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "bulletproof",
	Short: "Back up your OpenClaw agent. Track changes. Rollback anytime.",
	Long: `Bulletproof backs up your AI agent with snapshot-based versioning.

Track changes over time, see what changed, and rollback when things go wrong.

Your agent changes over time — skills get added, personality drifts, memories
accumulate. This tool captures snapshots so you can see what changed and restore
your agent to any previous state.`,
}

func main() {
	// Add all commands
	rootCmd.AddCommand(commands.NewInitCommand())
	rootCmd.AddCommand(commands.NewBackupCommand())
	rootCmd.AddCommand(commands.NewRestoreCommand())
	rootCmd.AddCommand(commands.NewDiffCommand())
	rootCmd.AddCommand(commands.NewSnapshotsCommand())
	rootCmd.AddCommand(commands.NewPruneCommand())
	rootCmd.AddCommand(commands.NewConfigCommand())
	rootCmd.AddCommand(commands.NewVersionCommand())
	rootCmd.AddCommand(commands.NewSkillCommand())
	rootCmd.AddCommand(commands.NewAnalyticsCommand())
	rootCmd.AddCommand(commands.NewScheduleCommand())

	// Execute
	if err := rootCmd.Execute(); err != nil {
		// Check for debug mode
		if os.Getenv("BULLETPROOF_DEBUG") == "1" {
			fmt.Fprintf(os.Stderr, "Error: %+v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}

	// Check for updates after successful command (async, non-blocking)
	go version.PrintUpdateNotice()
}
