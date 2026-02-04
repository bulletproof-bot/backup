package backup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bulletproof-bot/backup/internal/config"
)

// TestEdgeCase_EmptyAgent tests backing up an empty OpenClaw installation
func TestEdgeCase_EmptyAgent(t *testing.T) {
	helper := newTestDataHelper(t)

	// Create minimal OpenClaw installation
	agentDir := filepath.Join(helper.baseDir, "empty-agent")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Only create openclaw.json (minimal requirement)
	helper.writeFile(filepath.Join(agentDir, "openclaw.json"), "{}")

	backupDir := helper.createBackupDestination("empty")

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

	// Backup empty agent
	result, err := engine.Backup(false, "Empty agent backup", false, false)
	helper.assertNoError(err, "Empty agent backup should succeed")

	if result.Snapshot == nil {
		t.Fatal("Snapshot should not be nil for empty agent")
	}

	// Verify at least openclaw.json was backed up
	snapshotPath := filepath.Join(backupDir, result.Snapshot.ID)
	helper.assertFileExists(filepath.Join(snapshotPath, "openclaw.json"))
}

// TestEdgeCase_FilesWithSpecialCharacters tests files with special characters in names
func TestEdgeCase_FilesWithSpecialCharacters(t *testing.T) {
	helper := newTestDataHelper(t)

	agentDir := helper.createOpenClawAgent("special-chars-agent")
	backupDir := helper.createBackupDestination("special-chars")

	// Create files with special characters
	specialFiles := []string{
		"file with spaces.txt",
		"file-with-dashes.txt",
		"file_with_underscores.txt",
		"file.multiple.dots.txt",
		"file(with)parens.txt",
		"file[with]brackets.txt",
	}

	for _, filename := range specialFiles {
		helper.writeFile(filepath.Join(agentDir, "workspace", filename), "test content")
	}

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

	// Backup
	result, err := engine.Backup(false, "Files with special characters", false, false)
	helper.assertNoError(err, "Backup with special characters failed")

	// Verify all files were backed up
	snapshotPath := filepath.Join(backupDir, result.Snapshot.ID)
	for _, filename := range specialFiles {
		helper.assertFileExists(filepath.Join(snapshotPath, "workspace", filename))
	}

	// Restore and verify
	helper.removeSkill(agentDir, "analysis.js") // Remove a file to verify restore
	err = engine.RestoreToTarget(result.Snapshot.ID, "", false, false, true)
	helper.assertNoError(err, "Restore failed")

	// Verify all special character files still exist
	for _, filename := range specialFiles {
		helper.assertFileExists(filepath.Join(agentDir, "workspace", filename))
	}
}

// TestEdgeCase_UnicodeFilenames tests files with Unicode characters
func TestEdgeCase_UnicodeFilenames(t *testing.T) {
	helper := newTestDataHelper(t)

	agentDir := helper.createOpenClawAgent("unicode-agent")
	backupDir := helper.createBackupDestination("unicode")

	// Create files with Unicode characters
	unicodeFiles := []string{
		"文件.txt",     // Chinese
		"файл.txt",   // Cyrillic
		"αρχείο.txt", // Greek
		"ファイル.txt",   // Japanese
		"emoji😀.txt", // Emoji
	}

	for _, filename := range unicodeFiles {
		helper.writeFile(filepath.Join(agentDir, "workspace", filename), "unicode content")
	}

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

	result, err := engine.Backup(false, "Unicode filenames", false, false)
	helper.assertNoError(err, "Backup with Unicode filenames failed")

	// Verify files were backed up
	snapshotPath := filepath.Join(backupDir, result.Snapshot.ID)
	for _, filename := range unicodeFiles {
		helper.assertFileExists(filepath.Join(snapshotPath, "workspace", filename))
	}
}

// TestEdgeCase_DeepDirectoryNesting tests very deep directory structures
func TestEdgeCase_DeepDirectoryNesting(t *testing.T) {
	helper := newTestDataHelper(t)

	agentDir := helper.createOpenClawAgent("deep-agent")
	backupDir := helper.createBackupDestination("deep")

	// Create deep nested directory (20 levels)
	deepPath := filepath.Join(agentDir, "workspace")
	for i := 0; i < 20; i++ {
		deepPath = filepath.Join(deepPath, "level"+string(rune('0'+i%10)))
	}

	if err := os.MkdirAll(deepPath, 0755); err != nil {
		t.Fatal(err)
	}

	// Create file at deepest level
	helper.writeFile(filepath.Join(deepPath, "deep.txt"), "deeply nested file")

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

	result, err := engine.Backup(false, "Deep directory nesting", false, false)
	helper.assertNoError(err, "Backup with deep nesting failed")

	// Verify deep file was backed up
	snapshotPath := filepath.Join(backupDir, result.Snapshot.ID)
	files := helper.listFiles(snapshotPath)

	foundDeepFile := false
	for _, file := range files {
		if strings.Contains(file, "deep.txt") {
			foundDeepFile = true
			break
		}
	}

	if !foundDeepFile {
		t.Error("Deeply nested file was not backed up")
	}
}

// TestEdgeCase_SymlinksHandling tests behavior with symbolic links
func TestEdgeCase_SymlinksHandling(t *testing.T) {
	if os.Getenv("SKIP_SYMLINK_TESTS") != "" {
		t.Skip("Symlink tests skipped")
	}

	helper := newTestDataHelper(t)

	agentDir := helper.createOpenClawAgent("symlink-agent")
	backupDir := helper.createBackupDestination("symlink")

	// Create a target file
	targetFile := filepath.Join(agentDir, "workspace", "target.txt")
	helper.writeFile(targetFile, "target content")

	// Create a symlink to the file
	symlinkPath := filepath.Join(agentDir, "workspace", "link.txt")
	if err := os.Symlink(targetFile, symlinkPath); err != nil {
		t.Skipf("Cannot create symlinks on this system: %v", err)
	}

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

	result, err := engine.Backup(false, "Backup with symlinks", false, false)
	helper.assertNoError(err, "Backup with symlinks failed")

	// Verify symlink handling (implementation may choose to follow or copy symlinks)
	// This test documents current behavior
	snapshotPath := filepath.Join(backupDir, result.Snapshot.ID)
	t.Logf("Snapshot created with symlinks: %s", snapshotPath)

	// Either the symlink or its target should be backed up
	linkExists := helper.fileExists(filepath.Join(snapshotPath, "workspace", "link.txt"))
	targetExists := helper.fileExists(filepath.Join(snapshotPath, "workspace", "target.txt"))

	if !linkExists && !targetExists {
		t.Error("Either symlink or target should be backed up")
	}
}

// TestEdgeCase_ReadOnlyFiles tests backup of read-only files
func TestEdgeCase_ReadOnlyFiles(t *testing.T) {
	helper := newTestDataHelper(t)

	agentDir := helper.createOpenClawAgent("readonly-agent")
	backupDir := helper.createBackupDestination("readonly")

	// Create a read-only file
	readonlyFile := filepath.Join(agentDir, "workspace", "readonly.txt")
	helper.writeFile(readonlyFile, "readonly content")
	if err := os.Chmod(readonlyFile, 0444); err != nil {
		t.Fatal(err)
	}

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

	// Backup should succeed even with readonly files
	result, err := engine.Backup(false, "Backup with readonly files", false, false)
	helper.assertNoError(err, "Backup with readonly files should succeed")

	// Verify readonly file was backed up
	snapshotPath := filepath.Join(backupDir, result.Snapshot.ID)
	helper.assertFileExists(filepath.Join(snapshotPath, "workspace", "readonly.txt"))

	// Restore should succeed
	err = engine.RestoreToTarget(result.Snapshot.ID, "", false, false, true)
	helper.assertNoError(err, "Restore with readonly files should succeed")
}

// TestEdgeCase_ConcurrentBackups tests behavior when multiple backups run simultaneously
func TestEdgeCase_ConcurrentBackups(t *testing.T) {
	t.Skip("Concurrent backup handling not yet tested - would require locking mechanism")

	// This test would verify:
	// 1. Only one backup can run at a time
	// 2. Concurrent backup attempts wait or fail gracefully
	// 3. No data corruption from concurrent operations
}

// TestEdgeCase_DiskSpaceHandling tests behavior when disk is full
func TestEdgeCase_DiskSpaceHandling(t *testing.T) {
	t.Skip("Disk space testing requires special setup - manual test recommended")

	// This test would verify:
	// 1. Graceful failure when destination disk is full
	// 2. Partial backup is cleaned up on failure
	// 3. Clear error message about disk space
}

// TestEdgeCase_CorruptedSnapshot tests handling of corrupted snapshot data
func TestEdgeCase_CorruptedSnapshot(t *testing.T) {
	t.Skip("TODO: Requires self-contained snapshot structure with .bulletproof/ directory (Phase 2 feature)")
	helper := newTestDataHelper(t)

	agentDir := helper.createOpenClawAgent("corrupt-agent")
	backupDir := helper.createBackupDestination("corrupt")

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

	// Create a backup
	result, err := engine.Backup(false, "Original backup", false, false)
	helper.assertNoError(err, "Backup failed")

	// Corrupt the snapshot metadata
	metadataPath := filepath.Join(backupDir, result.Snapshot.ID, ".bulletproof", "snapshot.json")
	helper.writeFile(metadataPath, "corrupted data {{{")

	// Attempting to restore should fail gracefully
	err = engine.RestoreToTarget(result.Snapshot.ID, "", false, false, true)
	helper.assertError(err, "Restore of corrupted snapshot should fail")

	// Error message should be helpful
	if err != nil && !contains(err.Error(), "snapshot") {
		t.Logf("Error message should mention snapshot: %v", err)
	}
}

// TestEdgeCase_InvalidSnapshotID tests handling of invalid snapshot IDs
func TestEdgeCase_InvalidSnapshotID(t *testing.T) {
	helper := newTestDataHelper(t)

	agentDir := helper.createOpenClawAgent("invalid-agent")
	backupDir := helper.createBackupDestination("invalid")

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

	// Try to restore with non-existent snapshot ID
	err = engine.RestoreToTarget("nonexistent-snapshot-id", "", false, false, true)
	helper.assertError(err, "Restore with invalid ID should fail")

	// Try with malformed ID
	err = engine.RestoreToTarget("../../etc/passwd", "", false, false, true)
	helper.assertError(err, "Restore with path traversal should fail")
}

// TestEdgeCase_VeryLongFilenames tests files with maximum path lengths
func TestEdgeCase_VeryLongFilenames(t *testing.T) {
	helper := newTestDataHelper(t)

	agentDir := helper.createOpenClawAgent("long-agent")
	backupDir := helper.createBackupDestination("long")

	// Create file with long name (200 characters)
	longName := strings.Repeat("a", 200) + ".txt"
	longPath := filepath.Join(agentDir, "workspace", longName)

	// Some filesystems have limits, so this might fail
	err := os.WriteFile(longPath, []byte("content"), 0644)
	if err != nil {
		t.Skipf("Cannot create file with long name on this filesystem: %v", err)
	}

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

	result, err := engine.Backup(false, "Long filename backup", false, false)
	helper.assertNoError(err, "Backup with long filename failed")

	// Verify long filename was backed up
	snapshotPath := filepath.Join(backupDir, result.Snapshot.ID)
	helper.assertFileExists(filepath.Join(snapshotPath, "workspace", longName))
}

// TestEdgeCase_RapidSuccessiveBackups tests creating many backups quickly
func TestEdgeCase_RapidSuccessiveBackups(t *testing.T) {
	helper := newTestDataHelper(t)

	agentDir := helper.createOpenClawAgent("rapid-agent")
	backupDir := helper.createBackupDestination("rapid")

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

	// Create 10 backups rapidly
	snapshotIDs := make([]string, 0, 10)
	for i := 0; i < 10; i++ {
		// Make a small change each time
		helper.writeFile(filepath.Join(agentDir, "workspace", "file"+string(rune('0'+i))+".txt"), "content")

		result, err := engine.Backup(false, "Rapid backup "+string(rune('0'+i)), false, false)
		helper.assertNoError(err, "Rapid backup failed")

		if !result.Skipped {
			snapshotIDs = append(snapshotIDs, result.Snapshot.ID)
		}
	}

	// Verify all snapshots have unique IDs (timestamp-based should guarantee this)
	idMap := make(map[string]bool)
	for _, id := range snapshotIDs {
		if idMap[id] {
			t.Errorf("Duplicate snapshot ID detected: %s", id)
		}
		idMap[id] = true
	}

	if len(snapshotIDs) < 8 {
		t.Errorf("Expected at least 8 unique snapshots, got %d", len(snapshotIDs))
	}
}

// TestEdgeCase_BackupDuringRestore tests behavior when backup runs during restore
func TestEdgeCase_BackupDuringRestore(t *testing.T) {
	t.Skip("Concurrent operation testing not yet implemented - would require goroutines")

	// This test would verify:
	// 1. Operations are properly serialized
	// 2. No data corruption
	// 3. Clear error messages or waiting behavior
}

// TestEdgeCase_PermissionErrors tests handling of permission errors
func TestEdgeCase_PermissionErrors(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Running as root - cannot test permission errors")
	}

	helper := newTestDataHelper(t)

	agentDir := helper.createOpenClawAgent("perm-agent")
	backupDir := helper.createBackupDestination("perm")

	// Create a file we can't read
	unreadableFile := filepath.Join(agentDir, "workspace", "unreadable.txt")
	helper.writeFile(unreadableFile, "secret content")
	if err := os.Chmod(unreadableFile, 0000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(unreadableFile, 0644) // Cleanup

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

	// Backup should handle permission error gracefully
	_, err = engine.Backup(false, "Backup with permission errors", false, false)

	// Implementation may choose to:
	// 1. Skip unreadable files and continue
	// 2. Fail the entire backup
	// Either behavior is acceptable, but error should be clear
	if err != nil {
		t.Logf("Backup with permission errors: %v", err)
	}
}

// TestEdgeCase_ZeroByteFiles tests handling of empty files
func TestEdgeCase_ZeroByteFiles(t *testing.T) {
	helper := newTestDataHelper(t)

	agentDir := helper.createOpenClawAgent("zero-agent")
	backupDir := helper.createBackupDestination("zero")

	// Create empty files
	emptyFiles := []string{
		"empty1.txt",
		"empty2.json",
		"empty3.md",
	}

	for _, filename := range emptyFiles {
		helper.writeFile(filepath.Join(agentDir, "workspace", filename), "")
	}

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

	result, err := engine.Backup(false, "Backup with zero-byte files", false, false)
	helper.assertNoError(err, "Backup with empty files failed")

	// Verify empty files were backed up
	snapshotPath := filepath.Join(backupDir, result.Snapshot.ID)
	for _, filename := range emptyFiles {
		helper.assertFileExists(filepath.Join(snapshotPath, "workspace", filename))

		// Verify they're still empty
		content := helper.readFile(filepath.Join(snapshotPath, "workspace", filename))
		if len(content) != 0 {
			t.Errorf("Empty file %s has content after backup", filename)
		}
	}
}

// TestEdgeCase_BackupOfBackups tests backing up the backup directory itself
func TestEdgeCase_BackupOfBackups(t *testing.T) {
	helper := newTestDataHelper(t)

	agentDir := helper.createOpenClawAgent("recursive-agent")

	// Create backup dir INSIDE the agent directory (potential recursion)
	backupDir := filepath.Join(agentDir, "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		OpenclawPath: agentDir,
		Destination: &config.DestinationConfig{
			Type: "local",
			Path: backupDir,
		},
		Options: config.BackupOptions{
			Exclude: []string{"backups/"}, // Should exclude backup directory
		},
	}

	engine, err := NewBackupEngine(cfg)
	helper.assertNoError(err, "NewBackupEngine failed")

	// This should not recurse infinitely
	result, err := engine.Backup(false, "Test exclusion of backup directory", false, false)
	helper.assertNoError(err, "Backup should not recurse infinitely")

	// Verify snapshot was created but doesn't include backups directory
	snapshotPath := filepath.Join(backupDir, result.Snapshot.ID)
	helper.assertFileNotExists(filepath.Join(snapshotPath, "backups"))
}
