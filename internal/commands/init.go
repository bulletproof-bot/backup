package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/bulletproof-bot/backup/internal/config"
	"github.com/spf13/cobra"
)

// NewInitCommand creates the init command
func NewInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize bulletproof configuration",
		Long:  "Interactive setup wizard to configure OpenClaw path and backup destination.",
		RunE:  runInit,
	}
}

func runInit(cmd *cobra.Command, args []string) error {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("🚀 Bulletproof Setup")
	fmt.Println()

	// Detect OpenClaw path
	var openclawPath string
	detected := config.DetectInstallation()
	if detected != "" {
		fmt.Printf("Detected OpenClaw installation at: %s\n", detected)
		fmt.Print("Use this path? [Y/n]: ")
		scanner.Scan()
		response := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if response == "" || response == "y" || response == "yes" {
			openclawPath = detected
		}
	}

	if openclawPath == "" {
		fmt.Print("Enter OpenClaw installation path: ")
		scanner.Scan()
		openclawPath = strings.TrimSpace(scanner.Text())

		// Validate the path
		if err := config.Validate(openclawPath); err != nil {
			return fmt.Errorf("invalid OpenClaw path: %w", err)
		}
	}

	// Choose destination type
	fmt.Println()
	fmt.Println("Where should backups be stored?")
	fmt.Println("  1. Local directory")
	fmt.Println("  2. Git repository")
	fmt.Println("  3. Cloud sync folder (Dropbox/Google Drive)")
	fmt.Print("Choose [1-3]: ")
	scanner.Scan()
	choice := strings.TrimSpace(scanner.Text())

	var destType, destPath string
	switch choice {
	case "1":
		destType = "local"
		fmt.Print("Enter local directory path: ")
		scanner.Scan()
		destPath = strings.TrimSpace(scanner.Text())

	case "2":
		destType = "git"
		fmt.Println("Enter git repository:")
		fmt.Println("  - Local path: /path/to/repo")
		fmt.Println("  - Remote URL: https://github.com/user/repo.git")
		fmt.Print("Repository: ")
		scanner.Scan()
		destPath = strings.TrimSpace(scanner.Text())

	case "3":
		destType = "sync"
		fmt.Print("Enter cloud sync folder path: ")
		scanner.Scan()
		destPath = strings.TrimSpace(scanner.Text())

	default:
		return fmt.Errorf("invalid choice: %s", choice)
	}

	// Create config
	cfg := &config.Config{
		OpenclawPath: openclawPath,
		Destination: &config.DestinationConfig{
			Type: destType,
			Path: destPath,
		},
		Schedule: config.ScheduleConfig{
			Enabled: false,
			Time:    "03:00",
		},
		Options: config.BackupOptions{
			IncludeAuth: false,
			Exclude:     []string{"*.log", "node_modules/", ".git/"},
		},
	}

	// Save config
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	configPath, _ := config.ConfigPath()
	fmt.Println()
	fmt.Println("✅ Configuration saved to:", configPath)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  - Run 'bulletproof backup' to create your first backup")
	fmt.Println("  - Run 'bulletproof schedule enable' to enable automatic backups")

	return nil
}
