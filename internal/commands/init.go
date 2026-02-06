package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bulletproof-bot/backup/internal/config"
	"github.com/bulletproof-bot/backup/internal/platform"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// NewInitCommand creates the init command
func NewInitCommand() *cobra.Command {
	var fromBackup string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize bulletproof configuration",
		Long:  "Interactive setup wizard to configure OpenClaw path and backup destination.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(fromBackup)
		},
	}

	cmd.Flags().StringVar(&fromBackup, "from-backup", "", "Initialize from existing backup path")

	return cmd
}

func runInit(fromBackup string) error {
	// If initializing from backup, load config from backup
	if fromBackup != "" {
		return runInitFromBackup(fromBackup)
	}

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
	}

	// Convert to absolute path (critical: avoid CWD dependency)
	absOpenclawPath, err := filepath.Abs(openclawPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	openclawPath = absOpenclawPath

	// Validate the path
	if err := config.Validate(openclawPath); err != nil {
		return fmt.Errorf("invalid OpenClaw path: %w", err)
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

	// Convert destination path to absolute (critical: avoid CWD dependency)
	// Only convert if it's not a URL (git remotes can be URLs)
	if !strings.HasPrefix(destPath, "http://") && !strings.HasPrefix(destPath, "https://") &&
		!strings.HasPrefix(destPath, "git@") && !strings.HasPrefix(destPath, "ssh://") {
		absDestPath, err := filepath.Abs(destPath)
		if err != nil {
			return fmt.Errorf("invalid destination path: %w", err)
		}
		destPath = absDestPath
	}

	// Create config with scheduling enabled by default
	cfg := &config.Config{
		OpenclawPath: openclawPath,
		Destination: &config.DestinationConfig{
			Type: destType,
			Path: destPath,
		},
		Schedule: config.ScheduleConfig{
			Enabled: true,
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

	// Automatically set up scheduled backups
	fmt.Println()
	fmt.Println("⏰ Setting up automatic daily backups at 03:00...")
	if err := platform.SetupAutoBackup("03:00"); err != nil {
		fmt.Printf("⚠️  Warning: Failed to set up automatic backups: %v\n", err)
		fmt.Println("   You can set this up later with: bulletproof schedule enable")
	} else {
		fmt.Println("✅ Automatic backups scheduled for 03:00 daily")
	}

	fmt.Println()
	fmt.Println("🎉 You're bulletproof! Your agent is now protected.")
	fmt.Println()
	fmt.Println("What's next:")
	fmt.Println("  - Your agent will back up automatically every day at 03:00")
	fmt.Println("  - Run 'bulletproof backup' to create a backup right now")
	fmt.Println("  - Run 'bulletproof schedule disable' to turn off automatic backups")
	fmt.Println("  - Run 'bulletproof schedule enable --time HH:MM' to change the backup time")

	return nil
}

func runInitFromBackup(backupPath string) error {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("🚀 Bulletproof Setup from Backup")
	fmt.Printf("📦 Loading configuration from: %s\n", backupPath)
	fmt.Println()

	// Read config from backup's .bulletproof/config.yaml
	configPath := filepath.Join(backupPath, ".bulletproof", "config.yaml")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config from backup: %w", err)
	}

	// Parse config
	var cfg config.Config
	if err := yaml.Unmarshal(configData, &cfg); err != nil {
		return fmt.Errorf("failed to parse config from backup: %w", err)
	}

	// Prompt for new OpenClaw path (may be different on new machine)
	fmt.Printf("Original OpenClaw path: %s\n", cfg.OpenclawPath)

	detected := config.DetectInstallation()
	if detected != "" && detected != cfg.OpenclawPath {
		fmt.Printf("Detected OpenClaw installation at: %s\n", detected)
		fmt.Print("Use detected path instead? [Y/n]: ")
		scanner.Scan()
		response := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if response == "" || response == "y" || response == "yes" {
			cfg.OpenclawPath = detected
		}
	} else {
		fmt.Print("Update OpenClaw path? [y/N]: ")
		scanner.Scan()
		response := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if response == "y" || response == "yes" {
			fmt.Print("Enter new OpenClaw path: ")
			scanner.Scan()
			newPath := strings.TrimSpace(scanner.Text())
			if newPath != "" {
				// Convert to absolute path (critical: avoid CWD dependency)
				absPath, err := filepath.Abs(newPath)
				if err != nil {
					return fmt.Errorf("invalid path: %w", err)
				}
				cfg.OpenclawPath = absPath
			}
		}
	}

	// Prompt for new backup destination (likely different on new machine)
	fmt.Println()
	fmt.Printf("Original backup destination: %s (%s)\n", cfg.Destination.Path, cfg.Destination.Type)
	fmt.Print("Update backup destination? [Y/n]: ")
	scanner.Scan()
	response := strings.ToLower(strings.TrimSpace(scanner.Text()))
	if response == "" || response == "y" || response == "yes" {
		fmt.Print("Enter new backup destination path: ")
		scanner.Scan()
		newDest := strings.TrimSpace(scanner.Text())
		if newDest != "" {
			// Convert to absolute path (critical: avoid CWD dependency)
			// Only convert if it's not a URL (git remotes can be URLs)
			if !strings.HasPrefix(newDest, "http://") && !strings.HasPrefix(newDest, "https://") &&
				!strings.HasPrefix(newDest, "git@") && !strings.HasPrefix(newDest, "ssh://") {
				absPath, err := filepath.Abs(newDest)
				if err != nil {
					return fmt.Errorf("invalid destination path: %w", err)
				}
				newDest = absPath
			}
			cfg.Destination.Path = newDest
		}
	}

	// Save config
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	savedConfigPath, _ := config.ConfigPath()
	fmt.Println()
	fmt.Println("✅ Configuration restored and saved to:", savedConfigPath)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  - Review config: bulletproof config show")
	fmt.Println("  - Create backup: bulletproof backup")

	return nil
}
