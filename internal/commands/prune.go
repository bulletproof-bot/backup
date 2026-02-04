package commands

import (
	"fmt"

	"github.com/bulletproof-bot/backup/internal/backup"
	"github.com/bulletproof-bot/backup/internal/config"
	"github.com/bulletproof-bot/backup/internal/types"
	"github.com/spf13/cobra"
)

// NewPruneCommand creates the prune command
func NewPruneCommand() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Delete old snapshots according to retention policy",
		Long: `Delete old backup snapshots according to the configured retention policy.

The retention policy is configured in config.yaml and can include:
  - keep_last: Keep the last N snapshots
  - keep_daily: Keep one snapshot per day for N days
  - keep_weekly: Keep one snapshot per week for N weeks
  - keep_monthly: Keep one snapshot per month for N months

Use --dry-run to see what would be deleted without actually deleting anything.`,
		RunE: func(c *cobra.Command, args []string) error {
			return runPrune(dryRun)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be deleted without actually deleting")

	return cmd
}

func runPrune(dryRun bool) error {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Check if retention policy is enabled
	if !cfg.Retention.Enabled {
		fmt.Println("❌ Retention policy is not enabled in configuration.")
		fmt.Println()
		fmt.Println("To enable retention policy, edit your config file:")
		fmt.Printf("  %s\n", config.DefaultConfigPath())
		fmt.Println()
		fmt.Println("Add a retention section like this:")
		fmt.Println("  retention:")
		fmt.Println("    enabled: true")
		fmt.Println("    keep_last: 10        # Keep last 10 snapshots")
		fmt.Println("    keep_daily: 7        # Keep daily snapshots for 7 days")
		fmt.Println("    keep_weekly: 4       # Keep weekly snapshots for 4 weeks")
		fmt.Println("    keep_monthly: 6      # Keep monthly snapshots for 6 months")
		return nil
	}

	// Create backup engine
	engine, err := backup.NewBackupEngine(cfg)
	if err != nil {
		return err
	}

	// Run prune
	if dryRun {
		fmt.Println("🔍 Dry run - showing what would be deleted...")
		fmt.Println()
	} else {
		fmt.Println("🗑️  Pruning old snapshots...")
		fmt.Println()
	}

	result, err := engine.Prune(dryRun)
	if err != nil {
		return err
	}

	// Display retention policy
	fmt.Println("📋 Retention Policy:")
	if cfg.Retention.KeepLast > 0 {
		fmt.Printf("  • Keep last %d snapshots\n", cfg.Retention.KeepLast)
	}
	if cfg.Retention.KeepDaily > 0 {
		fmt.Printf("  • Keep daily snapshots for %d days\n", cfg.Retention.KeepDaily)
	}
	if cfg.Retention.KeepWeekly > 0 {
		fmt.Printf("  • Keep weekly snapshots for %d weeks\n", cfg.Retention.KeepWeekly)
	}
	if cfg.Retention.KeepMonthly > 0 {
		fmt.Printf("  • Keep monthly snapshots for %d months\n", cfg.Retention.KeepMonthly)
	}
	fmt.Println()

	// Display results
	fmt.Printf("📊 Summary:\n")
	fmt.Printf("  Total snapshots: %d\n", result.TotalSnapshots)
	fmt.Printf("  Snapshots to keep: %d\n", len(result.SnapshotsToKeep))
	fmt.Printf("  Snapshots to delete: %d\n", len(result.SnapshotsToDelete))
	fmt.Println()

	if len(result.SnapshotsToDelete) == 0 {
		fmt.Println("✨ No snapshots to delete - all snapshots match retention policy")
		return nil
	}

	// Show snapshots to delete
	if len(result.SnapshotsToDelete) > 0 {
		// Assign short IDs for display
		allSnapshots := append(result.SnapshotsToKeep, result.SnapshotsToDelete...)
		shortIDs := types.AssignShortIDs(allSnapshots)

		fmt.Println("📝 Snapshots to delete:")
		for _, snapshot := range result.SnapshotsToDelete {
			shortID := shortIDs[snapshot.ID]
			msg := ""
			if snapshot.Message != "" {
				msg = fmt.Sprintf(" - %s", snapshot.Message)
			}
			fmt.Printf("  [%d] %s%s (%d files)\n", shortID, snapshot.Timestamp.Format("2006-01-02 15:04:05"), msg, snapshot.FileCount)
		}
		fmt.Println()
	}

	if dryRun {
		fmt.Println("💡 Run without --dry-run to actually delete these snapshots")
	} else {
		fmt.Println("✅ Prune complete!")
	}

	return nil
}
