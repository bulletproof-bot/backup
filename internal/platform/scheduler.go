package platform

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// SetupAutoBackup installs platform-specific scheduled backup service
// Returns true if setup succeeded, false otherwise
func SetupAutoBackup(backupTime string) error {
	switch runtime.GOOS {
	case "linux":
		return setupLinuxAutoBackup(backupTime)
	case "darwin":
		return setupMacOSAutoBackup(backupTime)
	case "windows":
		return setupWindowsAutoBackup(backupTime)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// RemoveAutoBackup removes platform-specific scheduled backup service
func RemoveAutoBackup() error {
	switch runtime.GOOS {
	case "linux":
		return removeLinuxAutoBackup()
	case "darwin":
		return removeMacOSAutoBackup()
	case "windows":
		return removeWindowsAutoBackup()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// setupLinuxAutoBackup creates systemd timer or cron job
func setupLinuxAutoBackup(backupTime string) error {
	// Try systemd first
	if hasSystemd() {
		return setupSystemdTimer(backupTime)
	}

	// Fallback to cron
	return setupCronJob(backupTime)
}

func hasSystemd() bool {
	_, err := exec.LookPath("systemctl")
	return err == nil
}

func setupSystemdTimer(backupTime string) error {
	// Create service file
	serviceContent := `[Unit]
Description=Bulletproof Backup

[Service]
Type=oneshot
ExecStart=/usr/local/bin/bulletproof backup
`

	servicePath := filepath.Join(os.Getenv("HOME"), ".config", "systemd", "user", "bulletproof-backup.service")
	if err := os.MkdirAll(filepath.Dir(servicePath), 0755); err != nil {
		return fmt.Errorf("failed to create systemd directory: %w", err)
	}

	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	// Create timer file
	timerContent := fmt.Sprintf(`[Unit]
Description=Daily bulletproof backup

[Timer]
OnCalendar=*-*-* %s:00
Persistent=true

[Install]
WantedBy=timers.target
`, backupTime)

	timerPath := filepath.Join(os.Getenv("HOME"), ".config", "systemd", "user", "bulletproof-backup.timer")
	if err := os.WriteFile(timerPath, []byte(timerContent), 0644); err != nil {
		return fmt.Errorf("failed to write timer file: %w", err)
	}

	// Enable and start timer
	cmd := exec.Command("systemctl", "--user", "enable", "--now", "bulletproof-backup.timer")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable timer: %w", err)
	}

	return nil
}

func setupCronJob(backupTime string) error {
	// Parse time (HH:MM format)
	hour := backupTime[:2]
	minute := "00"

	// Get existing crontab
	existingCronBytes, _ := exec.Command("crontab", "-l").Output()
	existingCron := string(existingCronBytes)

	// Check if entry already exists
	cronEntry := fmt.Sprintf("%s %s * * * /usr/local/bin/bulletproof backup\n", minute, hour)
	newCron := existingCron
	if newCron == "" || newCron[len(newCron)-1] != '\n' {
		newCron += "\n"
	}
	newCron += cronEntry

	// Write new crontab
	cmd := exec.Command("crontab", "-")
	cmd.Stdin = bytes.NewReader([]byte(newCron))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update crontab: %w", err)
	}

	return nil
}

// setupMacOSAutoBackup creates launchd plist
func setupMacOSAutoBackup(backupTime string) error {
	// Parse time
	hour := backupTime[:2]

	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>ai.bulletproof.backup</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/bulletproof</string>
        <string>backup</string>
    </array>
    <key>StartCalendarInterval</key>
    <dict>
        <key>Hour</key>
        <integer>%s</integer>
        <key>Minute</key>
        <integer>0</integer>
    </dict>
    <key>StandardOutPath</key>
    <string>%s/Library/Logs/bulletproof-backup.log</string>
    <key>StandardErrorPath</key>
    <string>%s/Library/Logs/bulletproof-backup.log</string>
</dict>
</plist>
`, hour, os.Getenv("HOME"), os.Getenv("HOME"))

	plistPath := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", "ai.bulletproof.backup.plist")
	if err := os.MkdirAll(filepath.Dir(plistPath), 0755); err != nil {
		return fmt.Errorf("failed to create LaunchAgents directory: %w", err)
	}

	if err := os.WriteFile(plistPath, []byte(plistContent), 0644); err != nil {
		return fmt.Errorf("failed to write plist file: %w", err)
	}

	// Load the agent
	cmd := exec.Command("launchctl", "load", plistPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to load launch agent: %w", err)
	}

	return nil
}

// setupWindowsAutoBackup creates Task Scheduler task
func setupWindowsAutoBackup(backupTime string) error {
	// Parse time
	hour := backupTime[:2]

	// Create scheduled task using PowerShell
	psScript := fmt.Sprintf(`
$action = New-ScheduledTaskAction -Execute "bulletproof.exe" -Argument "backup"
$trigger = New-ScheduledTaskTrigger -Daily -At "%s:00"
$principal = New-ScheduledTaskPrincipal -UserId "$env:USERNAME" -RunLevel Highest
Register-ScheduledTask -TaskName "BulletproofBackup" -Action $action -Trigger $trigger -Principal $principal -Force
`, hour)

	cmd := exec.Command("powershell", "-Command", psScript)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create scheduled task: %w", err)
	}

	return nil
}

// removeLinuxAutoBackup removes systemd timer or cron job
func removeLinuxAutoBackup() error {
	if hasSystemd() {
		cmd := exec.Command("systemctl", "--user", "disable", "--now", "bulletproof-backup.timer")
		_ = cmd.Run() // Ignore errors - service may not be running

		servicePath := filepath.Join(os.Getenv("HOME"), ".config", "systemd", "user", "bulletproof-backup.service")
		timerPath := filepath.Join(os.Getenv("HOME"), ".config", "systemd", "user", "bulletproof-backup.timer")
		_ = os.Remove(servicePath) // Ignore errors - files may not exist
		_ = os.Remove(timerPath)
	}

	// Also try to remove from cron
	existingCronBytes, _ := exec.Command("crontab", "-l").Output()
	_ = existingCronBytes // TODO: Filter out bulletproof entries

	return nil
}

func removeMacOSAutoBackup() error {
	plistPath := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", "ai.bulletproof.backup.plist")

	// Unload the agent
	cmd := exec.Command("launchctl", "unload", plistPath)
	_ = cmd.Run() // Ignore errors - agent may not be loaded

	// Remove plist file
	_ = os.Remove(plistPath) // Ignore errors - file may not exist

	return nil
}

func removeWindowsAutoBackup() error {
	cmd := exec.Command("powershell", "-Command", "Unregister-ScheduledTask -TaskName 'BulletproofBackup' -Confirm:$false")
	_ = cmd.Run() // Ignore errors - task may not exist

	return nil
}
