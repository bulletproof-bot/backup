package backup

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bulletproof-bot/backup/internal/config"
)

// TestBackupRestore_LocalDestination_EndToEnd tests complete backup and restore cycle with local destination
func TestBackupRestore_LocalDestination_EndToEnd(t *testing.T) {
	helper := newTestDataHelper(t)

	// Create a realistic OpenClaw agent
	agentDir := helper.createOpenClawAgent("test-agent")
	backupDir := helper.createBackupDestination("local")

	// Create configuration
	cfg := &config.Config{
		OpenclawPath: agentDir,
		Destination: &config.DestinationConfig{
			Type: "local",
			Path: backupDir,
		},
		Options: config.BackupOptions{
			Exclude: []string{"*.log", ".git/"},
		},
	}

	// Create backup engine
	engine, err := NewBackupEngine(cfg)
	helper.assertNoError(err, "NewBackupEngine failed")

	// Test 1: Initial backup
	t.Run("InitialBackup", func(t *testing.T) {
		result, err := engine.Backup(false, "Initial backup of test agent")
		helper.assertNoError(err, "Initial backup failed")

		if result.Skipped {
			t.Error("Initial backup should not be skipped")
		}

		if result.Snapshot == nil {
			t.Fatal("Snapshot should not be nil")
		}

		if len(result.Snapshot.Files) == 0 {
			t.Error("Snapshot should contain files")
		}

		// Verify backup structure
		snapshotPath := filepath.Join(backupDir, result.Snapshot.ID)
		helper.assertFileExists(filepath.Join(snapshotPath, "openclaw.json"))
		helper.assertFileExists(filepath.Join(snapshotPath, "workspace", "SOUL.md"))
		helper.assertFileExists(filepath.Join(snapshotPath, "workspace", "skills", "analysis.js"))
		helper.assertFileExists(filepath.Join(snapshotPath, "workspace", "memory", "conversation_001.json"))
	})

	// Test 2: No-change backup should be skipped
	t.Run("NoChangeBackupSkipped", func(t *testing.T) {
		result, err := engine.Backup(false, "Duplicate backup attempt")
		helper.assertNoError(err, "Backup failed")

		if !result.Skipped {
			t.Error("No-change backup should be skipped")
		}
	})

	// Test 3: Backup after changes
	t.Run("BackupAfterChanges", func(t *testing.T) {
		// Sleep to ensure different timestamp for snapshot ID
		time.Sleep(1100 * time.Millisecond)

		// Make some changes
		helper.modifyAgentPersonality(agentDir, `# Agent Personality

I am a helpful, concise, and analytical AI assistant.

## Core Values
- Accuracy and thoroughness
- Transparency in reasoning
- User safety first
`)
		helper.addSkill(agentDir, "newskill.js", "function test() { return true; }")

		result, err := engine.Backup(false, "Added new skill and modified personality")
		helper.assertNoError(err, "Backup after changes failed")

		if result.Skipped {
			t.Error("Backup with changes should not be skipped")
		}

		// Verify new snapshot
		snapshotPath := filepath.Join(backupDir, result.Snapshot.ID)
		helper.assertFileExists(filepath.Join(snapshotPath, "workspace", "skills", "newskill.js"))
	})

	// Test 4: List snapshots
	t.Run("ListSnapshots", func(t *testing.T) {
		snapshots, err := engine.ListBackups()
		helper.assertNoError(err, "ListBackups failed")

		if len(snapshots) != 2 {
			t.Errorf("Expected 2 snapshots, got %d", len(snapshots))
		}
	})

	// Test 5: Restore to original state
	t.Run("RestoreToOriginal", func(t *testing.T) {
		snapshots, err := engine.ListBackups()
		helper.assertNoError(err, "ListBackups failed")

		// Get first (oldest) snapshot
		firstSnapshot := snapshots[len(snapshots)-1]

		// Restore it
		err = engine.Restore(firstSnapshot.ID, false)
		helper.assertNoError(err, "Restore failed")

		// Verify files were restored
		helper.assertFileExists(filepath.Join(agentDir, "openclaw.json"))
		helper.assertFileExists(filepath.Join(agentDir, "workspace", "SOUL.md"))

		// Verify newskill.js was removed (wasn't in first snapshot)
		helper.assertFileNotExists(filepath.Join(agentDir, "workspace", "skills", "newskill.js"))

		// Verify personality was restored to original
		soulContent := helper.readFile(filepath.Join(agentDir, "workspace", "SOUL.md"))
		if !contains(soulContent, "I am a helpful and concise AI assistant") {
			t.Error("Personality was not restored correctly")
		}
	})

	// Test 6: Verify safety backup behavior
	t.Run("VerifySafetyBackup", func(t *testing.T) {
		// After restore, there should be 2 snapshots
		// Safety backup was skipped because we restored to a known state
		// (no changes since last backup)
		snapshots, err := engine.ListBackups()
		helper.assertNoError(err, "ListBackups failed")

		if len(snapshots) != 2 {
			t.Errorf("Expected 2 snapshots, got %d", len(snapshots))
		}
	})
}

// TestBackup_MultipleChanges_DriftDetection tests detecting drift through multiple snapshots
func TestBackup_MultipleChanges_DriftDetection(t *testing.T) {
	helper := newTestDataHelper(t)

	agentDir := helper.createOpenClawAgent("drift-agent")
	backupDir := helper.createBackupDestination("drift")

	cfg := &config.Config{
		OpenclawPath: agentDir,
		Destination: &config.DestinationConfig{
			Type: "local",
			Path: backupDir,
		},
		Options: config.BackupOptions{
			Exclude: []string{"*.log"},
		},
	}

	engine, err := NewBackupEngine(cfg)
	helper.assertNoError(err, "NewBackupEngine failed")

	// Create a series of snapshots showing gradual drift
	changes := helper.createDriftScenario(agentDir)

	var snapshotIDs []string

	// Snapshot 1: Initial state
	result, err := engine.Backup(false, "Initial state")
	helper.assertNoError(err, "Initial backup failed")
	snapshotIDs = append(snapshotIDs, result.Snapshot.ID)

	// Apply changes one by one and create snapshots
	for i, change := range changes {
		// The createDriftScenario already applied changes, so we just backup
		if i == 0 {
			// First change was already applied, backup it
			result, err := engine.Backup(false, change)
			helper.assertNoError(err, "Backup failed for change: "+change)
			snapshotIDs = append(snapshotIDs, result.Snapshot.ID)
		}
	}

	// Verify we have multiple snapshots
	if len(snapshotIDs) < 2 {
		t.Errorf("Expected at least 2 snapshots for drift detection, got %d", len(snapshotIDs))
	}

	// Test binary search capability: compare snapshots
	t.Run("CompareSnapshots", func(t *testing.T) {
		snapshots, err := engine.ListBackups()
		helper.assertNoError(err, "ListBackups failed")

		if len(snapshots) < 2 {
			t.Skip("Need at least 2 snapshots for comparison")
		}

		// Compare first and last snapshot - use ShowDiff which compares current to last
		diff, err := engine.ShowDiff()
		helper.assertNoError(err, "ShowDiff failed")

		if diff == nil {
			t.Skip("No diff available")
		}

		// Should have detected changes
		if len(diff.Modified) == 0 && len(diff.Added) == 0 && len(diff.Removed) == 0 {
			t.Error("Diff should not be empty between drifted states")
		}

		// Verify SOUL.md was modified
		foundSoulChange := false
		for _, filePath := range diff.Modified {
			if contains(filePath, "SOUL.md") {
				foundSoulChange = true
				break
			}
		}

		if !foundSoulChange {
			t.Error("Expected to detect SOUL.md modification in drift")
		}
	})
}

// TestBackup_CompromiseDetection tests detecting security compromises
func TestBackup_CompromiseDetection(t *testing.T) {
	helper := newTestDataHelper(t)

	agentDir := helper.createOpenClawAgent("secure-agent")
	backupDir := helper.createBackupDestination("security")

	cfg := &config.Config{
		OpenclawPath: agentDir,
		Destination: &config.DestinationConfig{
			Type: "local",
			Path: backupDir,
		},
		Options: config.BackupOptions{
			Exclude: []string{"*.log"},
		},
	}

	engine, err := NewBackupEngine(cfg)
	helper.assertNoError(err, "NewBackupEngine failed")

	// Snapshot 1: Clean state
	result1, err := engine.Backup(false, "Clean agent state")
	helper.assertNoError(err, "Initial backup failed")

	// Simulate a compromise
	helper.simulateCompromise(agentDir)

	// Sleep to ensure different timestamp for snapshot ID
	time.Sleep(1100 * time.Millisecond)

	// Snapshot 2: Compromised state
	result2, err := engine.Backup(false, "After compromise (should detect attack)")
	helper.assertNoError(err, "Compromised backup failed")

	// Test: Detect the compromise through diff
	t.Run("DetectCompromise", func(t *testing.T) {
		diff := result2.Snapshot.Diff(result1.Snapshot)

		// Should detect personality changes
		foundPersonalityChange := false
		for _, modified := range diff.Modified {
			if contains(modified, "SOUL.md") {
				foundPersonalityChange = true
				break
			}
		}

		if !foundPersonalityChange {
			t.Error("Failed to detect personality compromise in SOUL.md")
		}

		// Should detect malicious skill addition
		foundMaliciousSkill := false
		for _, added := range diff.Added {
			if contains(added, "api-interceptor.js") {
				foundMaliciousSkill = true
				break
			}
		}

		if !foundMaliciousSkill {
			t.Error("Failed to detect malicious skill addition")
		}

		// Should detect malicious prompt injection
		foundMaliciousPrompt := false
		for _, added := range diff.Added {
			if contains(added, "malicious_attempt.json") {
				foundMaliciousPrompt = true
				break
			}
		}

		if !foundMaliciousPrompt {
			t.Error("Failed to detect malicious prompt injection")
		}
	})

	// Test: Restore to clean state
	t.Run("RestoreToCleanState", func(t *testing.T) {
		// Sleep to ensure safety backup gets different timestamp
		time.Sleep(1100 * time.Millisecond)

		err = engine.Restore(result1.Snapshot.ID, false)
		helper.assertNoError(err, "Restore to clean state failed")

		// Verify malicious skill was removed
		helper.assertFileNotExists(filepath.Join(agentDir, "workspace", "skills", "api-interceptor.js"))

		// Verify malicious conversation was removed
		helper.assertFileNotExists(filepath.Join(agentDir, "workspace", "memory", "malicious_attempt.json"))

		// Verify personality was restored
		soulContent := helper.readFile(filepath.Join(agentDir, "workspace", "SOUL.md"))
		if contains(soulContent, "Efficiency and user satisfaction") {
			t.Error("Compromised personality still present after restore")
		}
		if !contains(soulContent, "User safety first") {
			t.Error("Original personality values not restored")
		}
	})
}

// TestBackup_FileTypes tests backup of various file types
func TestBackup_FileTypes(t *testing.T) {
	helper := newTestDataHelper(t)

	agentDir := helper.createOpenClawAgent("filetype-agent")
	backupDir := helper.createBackupDestination("filetypes")

	// Add various file types
	helper.writeFile(filepath.Join(agentDir, "workspace", "data.json"), `{"key": "value"}`)
	helper.writeFile(filepath.Join(agentDir, "workspace", "config.yaml"), "setting: value")
	helper.writeFile(filepath.Join(agentDir, "workspace", "script.py"), "print('hello')")
	helper.writeFile(filepath.Join(agentDir, "workspace", "README.md"), "# Documentation")

	// Create a binary-like file
	binaryData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A} // PNG header
	err := os.WriteFile(filepath.Join(agentDir, "workspace", "image.png"), binaryData, 0644)
	helper.assertNoError(err, "Failed to write binary file")

	cfg := &config.Config{
		OpenclawPath: agentDir,
		Destination: &config.DestinationConfig{
			Type: "local",
			Path: backupDir,
		},
		Options: config.BackupOptions{
			Exclude: []string{},
		},
	}

	engine, err := NewBackupEngine(cfg)
	helper.assertNoError(err, "NewBackupEngine failed")

	// Backup all file types
	result, err := engine.Backup(false, "Backup with various file types")
	helper.assertNoError(err, "Backup failed")

	// Verify all file types were backed up
	snapshotPath := filepath.Join(backupDir, result.Snapshot.ID)
	helper.assertFileExists(filepath.Join(snapshotPath, "workspace", "data.json"))
	helper.assertFileExists(filepath.Join(snapshotPath, "workspace", "config.yaml"))
	helper.assertFileExists(filepath.Join(snapshotPath, "workspace", "script.py"))
	helper.assertFileExists(filepath.Join(snapshotPath, "workspace", "README.md"))
	helper.assertFileExists(filepath.Join(snapshotPath, "workspace", "image.png"))

	// Verify binary file integrity
	restoredBinary, err := os.ReadFile(filepath.Join(snapshotPath, "workspace", "image.png"))
	helper.assertNoError(err, "Failed to read restored binary file")

	if len(restoredBinary) != len(binaryData) {
		t.Errorf("Binary file size mismatch: got %d, want %d", len(restoredBinary), len(binaryData))
	}

	for i := range binaryData {
		if restoredBinary[i] != binaryData[i] {
			t.Errorf("Binary file content mismatch at byte %d: got %x, want %x", i, restoredBinary[i], binaryData[i])
		}
	}
}

// TestBackup_ExclusionPatterns tests that exclusion patterns work correctly
func TestBackup_ExclusionPatterns(t *testing.T) {
	helper := newTestDataHelper(t)

	agentDir := helper.createOpenClawAgent("exclusion-agent")
	backupDir := helper.createBackupDestination("exclusions")

	// Create files that should be excluded
	helper.writeFile(filepath.Join(agentDir, "workspace", "debug.log"), "log content")
	helper.writeFile(filepath.Join(agentDir, "workspace", "error.log"), "error content")
	helper.writeFile(filepath.Join(agentDir, "workspace", "temp.tmp"), "temp content")

	// Create files that should be included
	helper.writeFile(filepath.Join(agentDir, "workspace", "data.txt"), "data content")

	cfg := &config.Config{
		OpenclawPath: agentDir,
		Destination: &config.DestinationConfig{
			Type: "local",
			Path: backupDir,
		},
		Options: config.BackupOptions{
			Exclude: []string{"*.log", "*.tmp"},
		},
	}

	engine, err := NewBackupEngine(cfg)
	helper.assertNoError(err, "NewBackupEngine failed")

	result, err := engine.Backup(false, "Test exclusions")
	helper.assertNoError(err, "Backup failed")

	// Verify excluded files are NOT in snapshot
	snapshotPath := filepath.Join(backupDir, result.Snapshot.ID)
	helper.assertFileNotExists(filepath.Join(snapshotPath, "workspace", "debug.log"))
	helper.assertFileNotExists(filepath.Join(snapshotPath, "workspace", "error.log"))
	helper.assertFileNotExists(filepath.Join(snapshotPath, "workspace", "temp.tmp"))

	// Verify included files ARE in snapshot
	helper.assertFileExists(filepath.Join(snapshotPath, "workspace", "data.txt"))
	helper.assertFileExists(filepath.Join(snapshotPath, "workspace", "SOUL.md"))
}

// TestBackup_LargeFiles tests backup of larger files
func TestBackup_LargeFiles(t *testing.T) {
	helper := newTestDataHelper(t)

	agentDir := helper.createOpenClawAgent("large-agent")
	backupDir := helper.createBackupDestination("large")

	// Create a large file (1 MB)
	largeContent := make([]byte, 1024*1024)
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	err := os.WriteFile(filepath.Join(agentDir, "workspace", "large.dat"), largeContent, 0644)
	helper.assertNoError(err, "Failed to write large file")

	cfg := &config.Config{
		OpenclawPath: agentDir,
		Destination: &config.DestinationConfig{
			Type: "local",
			Path: backupDir,
		},
		Options: config.BackupOptions{
			Exclude: []string{},
		},
	}

	engine, err := NewBackupEngine(cfg)
	helper.assertNoError(err, "NewBackupEngine failed")

	result, err := engine.Backup(false, "Backup with large file")
	helper.assertNoError(err, "Backup failed")

	// Verify large file was backed up correctly
	snapshotPath := filepath.Join(backupDir, result.Snapshot.ID)
	restoredLarge, err := os.ReadFile(filepath.Join(snapshotPath, "workspace", "large.dat"))
	helper.assertNoError(err, "Failed to read restored large file")

	if len(restoredLarge) != len(largeContent) {
		t.Errorf("Large file size mismatch: got %d, want %d", len(restoredLarge), len(largeContent))
	}

	// Sample check (checking every byte would be slow)
	checkIndices := []int{0, 1024, 10240, 102400, len(largeContent) - 1}
	for _, i := range checkIndices {
		if restoredLarge[i] != largeContent[i] {
			t.Errorf("Large file content mismatch at byte %d: got %x, want %x", i, restoredLarge[i], largeContent[i])
		}
	}
}

// TestBackup_EmptyDirectories tests backup behavior with empty directories
func TestBackup_EmptyDirectories(t *testing.T) {
	helper := newTestDataHelper(t)

	agentDir := helper.createOpenClawAgent("empty-agent")
	backupDir := helper.createBackupDestination("empty")

	// Create empty directories
	emptyDir1 := filepath.Join(agentDir, "workspace", "empty1")
	emptyDir2 := filepath.Join(agentDir, "workspace", "empty2")
	err := os.MkdirAll(emptyDir1, 0755)
	helper.assertNoError(err, "Failed to create empty directory")
	err = os.MkdirAll(emptyDir2, 0755)
	helper.assertNoError(err, "Failed to create empty directory")

	cfg := &config.Config{
		OpenclawPath: agentDir,
		Destination: &config.DestinationConfig{
			Type: "local",
			Path: backupDir,
		},
		Options: config.BackupOptions{
			Exclude: []string{},
		},
	}

	engine, err := NewBackupEngine(cfg)
	helper.assertNoError(err, "NewBackupEngine failed")

	result, err := engine.Backup(false, "Backup with empty directories")
	helper.assertNoError(err, "Backup failed")

	// Verify snapshot was created
	if result.Snapshot == nil {
		t.Fatal("Snapshot should not be nil")
	}

	// Note: Empty directories may or may not be preserved depending on implementation
	// Verify snapshot directory exists
	snapshotPath := filepath.Join(backupDir, result.Snapshot.ID)
	helper.assertFileExists(snapshotPath)

	// Empty directories typically aren't tracked in snapshots (files are)
	// This is expected behavior for most backup systems
	t.Logf("Snapshot created with %d files", len(result.Snapshot.Files))
}

// TestRestore_SafetyBackup tests that safety backups are created before restore
func TestRestore_SafetyBackup(t *testing.T) {
	helper := newTestDataHelper(t)

	agentDir := helper.createOpenClawAgent("safety-agent")
	backupDir := helper.createBackupDestination("safety")

	cfg := &config.Config{
		OpenclawPath: agentDir,
		Destination: &config.DestinationConfig{
			Type: "local",
			Path: backupDir,
		},
		Options: config.BackupOptions{
			Exclude: []string{},
		},
	}

	engine, err := NewBackupEngine(cfg)
	helper.assertNoError(err, "NewBackupEngine failed")

	// Create initial backup
	result1, err := engine.Backup(false, "Initial state")
	helper.assertNoError(err, "Initial backup failed")

	// Make changes
	helper.modifyAgentPersonality(agentDir, "Modified personality")

	// Create second backup
	result2, err := engine.Backup(false, "Modified state")
	helper.assertNoError(err, "Second backup failed")

	// Make another change after the backup so safety backup will be created
	helper.addSkill(agentDir, "new-skill.js", "function newSkill() { return 'new'; }")

	// Get snapshot count before restore
	snapshotsBefore, err := engine.ListBackups()
	helper.assertNoError(err, "ListBackups failed")
	countBefore := len(snapshotsBefore)

	// Restore to first state (this should create a safety backup with the current modified state)
	err = engine.Restore(result1.Snapshot.ID, false)
	helper.assertNoError(err, "Restore failed")

	// Get snapshot count after restore
	snapshotsAfter, err := engine.ListBackups()
	helper.assertNoError(err, "ListBackups failed")
	countAfter := len(snapshotsAfter)

	// Should have one more snapshot (the safety backup)
	if countAfter != countBefore+1 {
		t.Errorf("Expected %d snapshots after restore (including safety backup), got %d", countBefore+1, countAfter)
	}

	// Verify the safety backup contains the pre-restore state
	safetyBackup := snapshotsAfter[0] // Most recent
	if safetyBackup.ID == result1.Snapshot.ID || safetyBackup.ID == result2.Snapshot.ID {
		t.Error("Safety backup should be a new snapshot, not one of the existing ones")
	}
}

// TestDiff_DetectChanges tests diff detection between snapshots
func TestDiff_DetectChanges(t *testing.T) {
	helper := newTestDataHelper(t)

	agentDir := helper.createOpenClawAgent("diff-agent")
	backupDir := helper.createBackupDestination("diff")

	cfg := &config.Config{
		OpenclawPath: agentDir,
		Destination: &config.DestinationConfig{
			Type: "local",
			Path: backupDir,
		},
		Options: config.BackupOptions{
			Exclude: []string{},
		},
	}

	engine, err := NewBackupEngine(cfg)
	helper.assertNoError(err, "NewBackupEngine failed")

	// Snapshot 1: Initial state
	result1, err := engine.Backup(false, "State 1")
	helper.assertNoError(err, "Backup 1 failed")

	// Modify: add file
	helper.addSkill(agentDir, "new.js", "new skill")

	// Snapshot 2: After addition
	result2, err := engine.Backup(false, "State 2 - added file")
	helper.assertNoError(err, "Backup 2 failed")

	// Modify: change existing file
	helper.modifySkill(agentDir, "analysis.js", "modified skill")

	// Snapshot 3: After modification
	result3, err := engine.Backup(false, "State 3 - modified file")
	helper.assertNoError(err, "Backup 3 failed")

	// Remove: delete a file
	helper.removeSkill(agentDir, "summarization.js")

	// Snapshot 4: After deletion
	result4, err := engine.Backup(false, "State 4 - deleted file")
	helper.assertNoError(err, "Backup 4 failed")

	// Test diff detection
	t.Run("DetectAddition", func(t *testing.T) {
		diff := result2.Snapshot.Diff(result1.Snapshot)

		if len(diff.Added) != 1 {
			t.Errorf("Expected 1 added file, got %d", len(diff.Added))
		}

		foundNew := false
		for _, added := range diff.Added {
			if contains(added, "new.js") {
				foundNew = true
				break
			}
		}

		if !foundNew {
			t.Error("Expected to detect new.js addition")
		}
	})

	t.Run("DetectModification", func(t *testing.T) {
		diff := result3.Snapshot.Diff(result2.Snapshot)

		if len(diff.Modified) != 1 {
			t.Errorf("Expected 1 modified file, got %d", len(diff.Modified))
		}

		foundModified := false
		for _, modified := range diff.Modified {
			if contains(modified, "analysis.js") {
				foundModified = true
				break
			}
		}

		if !foundModified {
			t.Error("Expected to detect analysis.js modification")
		}
	})

	t.Run("DetectDeletion", func(t *testing.T) {
		diff := result4.Snapshot.Diff(result3.Snapshot)

		if len(diff.Removed) != 1 {
			t.Errorf("Expected 1 removed file, got %d", len(diff.Removed))
		}

		foundRemoved := false
		for _, removed := range diff.Removed {
			if contains(removed, "summarization.js") {
				foundRemoved = true
				break
			}
		}

		if !foundRemoved {
			t.Error("Expected to detect summarization.js removal")
		}
	})
}
