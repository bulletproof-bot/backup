package commands

import (
	"fmt"

	"github.com/bulletproof-bot/backup/internal/version"
	"github.com/spf13/cobra"
)

// NewVersionCommand creates the version command
func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  "Display the current version of bulletproof, including build information.",
		Run:   runVersion,
	}
}

func runVersion(cmd *cobra.Command, args []string) {
	fmt.Println(version.Info())

	// Check for updates
	latestVersion, downloadURL, err := version.CheckForUpdate()
	if err != nil {
		// Silently ignore errors
		return
	}

	if latestVersion != "" {
		fmt.Printf("\n💡 New version available: %s\n", latestVersion)
		fmt.Printf("   Download: %s\n", downloadURL)
	} else if version.Version != "dev" {
		fmt.Println("✅ You're running the latest version")
	}
}
