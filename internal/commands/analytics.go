package commands

import (
	"fmt"

	"github.com/bulletproof-bot/backup/internal/config"
	"github.com/spf13/cobra"
)

// NewAnalyticsCommand creates the analytics command
func NewAnalyticsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analytics",
		Short: "Manage anonymous usage analytics",
		Long: `Enable, disable, or check the status of anonymous usage analytics.

Analytics help improve Bulletproof by tracking command usage, OS/version info,
and flag usage. We never track file paths, snapshot contents, or any PII.`,
	}

	cmd.AddCommand(NewAnalyticsEnableCommand())
	cmd.AddCommand(NewAnalyticsDisableCommand())
	cmd.AddCommand(NewAnalyticsStatusCommand())

	return cmd
}

// NewAnalyticsEnableCommand creates the analytics enable command
func NewAnalyticsEnableCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "enable",
		Short: "Enable anonymous usage analytics",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAnalyticsEnable()
		},
	}
}

// NewAnalyticsDisableCommand creates the analytics disable command
func NewAnalyticsDisableCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "disable",
		Short: "Disable anonymous usage analytics",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAnalyticsDisable()
		},
	}
}

// NewAnalyticsStatusCommand creates the analytics status command
func NewAnalyticsStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show analytics status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAnalyticsStatus()
		},
	}
}

func runAnalyticsEnable() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	cfg.Analytics.Enabled = true
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println("✅ Analytics enabled")
	fmt.Println("\nWe collect anonymous usage data to improve Bulletproof:")
	fmt.Println("  • Command usage")
	fmt.Println("  • Operating system and version")
	fmt.Println("  • Tool version")
	fmt.Println("  • Flag usage")
	fmt.Println("\nWe never track file paths, snapshot contents, or personally identifiable information.")

	return nil
}

func runAnalyticsDisable() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	cfg.Analytics.Enabled = false
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println("✅ Analytics disabled")
	fmt.Println("\nNo analytics data will be collected.")

	return nil
}

func runAnalyticsStatus() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if cfg.Analytics.Enabled {
		fmt.Println("Status: ✅ Enabled")
		if cfg.Analytics.UserID != "" {
			fmt.Printf("User ID: %s\n", cfg.Analytics.UserID)
		}
	} else {
		fmt.Println("Status: ❌ Disabled")
	}

	fmt.Println("\nTo change analytics settings:")
	fmt.Println("  bulletproof analytics enable")
	fmt.Println("  bulletproof analytics disable")

	return nil
}
