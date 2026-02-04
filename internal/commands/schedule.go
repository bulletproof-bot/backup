package commands

import (
	"fmt"

	"github.com/bulletproof-bot/backup/internal/config"
	"github.com/bulletproof-bot/backup/internal/platform"
	"github.com/spf13/cobra"
)

// NewScheduleCommand creates the schedule command
func NewScheduleCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schedule",
		Short: "Manage automatic backup scheduling",
		Long: `Enable, disable, or check the status of automatic backup scheduling.

Scheduled backups run daily at a specified time using platform-specific services:
  - Linux: systemd timer or cron
  - macOS: launchd
  - Windows: Task Scheduler`,
	}

	cmd.AddCommand(NewScheduleEnableCommand())
	cmd.AddCommand(NewScheduleDisableCommand())
	cmd.AddCommand(NewScheduleStatusCommand())

	return cmd
}

// NewScheduleEnableCommand creates the schedule enable command
func NewScheduleEnableCommand() *cobra.Command {
	var timeStr string

	cmd := &cobra.Command{
		Use:   "enable",
		Short: "Enable automatic backup scheduling",
		Long:  "Enable automatic backup scheduling at the specified time (HH:MM format).",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScheduleEnable(timeStr)
		},
	}

	cmd.Flags().StringVarP(&timeStr, "time", "t", "03:00", "Daily backup time (HH:MM format)")

	return cmd
}

// NewScheduleDisableCommand creates the schedule disable command
func NewScheduleDisableCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "disable",
		Short: "Disable automatic backup scheduling",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScheduleDisable()
		},
	}
}

// NewScheduleStatusCommand creates the schedule status command
func NewScheduleStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show schedule status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScheduleStatus()
		},
	}
}

func runScheduleEnable(timeStr string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Validate time format
	if !isValidTime(timeStr) {
		return fmt.Errorf("invalid time format: %s (expected HH:MM)", timeStr)
	}

	// Set up platform-specific scheduled service
	if err := platform.SetupAutoBackup(timeStr); err != nil {
		return fmt.Errorf("failed to set up automatic backups: %w", err)
	}

	// Update config
	cfg.Schedule.Enabled = true
	cfg.Schedule.Time = timeStr

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✅ Automatic backups scheduled for %s daily\n", timeStr)

	return nil
}

func runScheduleDisable() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Remove platform-specific scheduled service
	if err := platform.RemoveAutoBackup(); err != nil {
		return fmt.Errorf("failed to remove automatic backups: %w", err)
	}

	// Update config
	cfg.Schedule.Enabled = false

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println("✅ Automatic backup scheduling disabled")

	return nil
}

func runScheduleStatus() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if cfg.Schedule.Enabled {
		fmt.Printf("Status: ✅ Enabled (daily at %s)\n", cfg.Schedule.Time)
	} else {
		fmt.Println("Status: ❌ Disabled")
	}

	fmt.Println("\nTo change schedule settings:")
	fmt.Println("  bulletproof schedule enable --time HH:MM")
	fmt.Println("  bulletproof schedule disable")

	return nil
}

// isValidTime validates HH:MM format
func isValidTime(timeStr string) bool {
	if len(timeStr) != 5 {
		return false
	}
	if timeStr[2] != ':' {
		return false
	}

	hour := timeStr[0:2]
	minute := timeStr[3:5]

	// Check hours (00-23)
	if hour < "00" || hour > "23" {
		return false
	}

	// Check minutes (00-59)
	if minute < "00" || minute > "59" {
		return false
	}

	return true
}
