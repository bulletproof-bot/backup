package backup

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bulletproof-bot/backup/internal/config"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// TestBackupRestore_GitDestination_EndToEnd tests complete backup and restore cycle with git destination
func TestBackupRestore_GitDestination_EndToEnd(t *testing.T) {
	helper := newTestDataHelper(t)

	// Create a realistic OpenClaw agent
	agentDir := helper.createOpenClawAgent("git-agent")
	backupDir := helper.createBackupDestination("git-backups")

	// Initialize git repository at backup destination
	_, err := gogit.PlainInit(backupDir, false)
	helper.assertNoError(err, "Failed to initialize git repository")

	// Create configuration for git destination
	cfg := &config.Config{
		OpenclawPath: agentDir,
		Destination: &config.DestinationConfig{
			Type: "git",
			Path: backupDir,
		},
		Options: config.BackupOptions{
			Exclude: []string{"*.log", ".git/"},
		},
	}

	// Create backup engine
	engine, err := NewBackupEngine(cfg)
	helper.assertNoError(err, "NewBackupEngine failed")

	// Test 1: Initial backup creates git commit and tag
	var firstSnapshotID string
	t.Run("InitialGitBackup", func(t *testing.T) {
		result, err := engine.Backup(false, "Initial git backup")
		helper.assertNoError(err, "Initial git backup failed")

		if result.Skipped {
			t.Error("Initial backup should not be skipped")
		}

		if result.Snapshot == nil {
			t.Fatal("Snapshot should not be nil")
		}

		firstSnapshotID = result.Snapshot.ID

		// Verify git repository state
		repo, err := gogit.PlainOpen(backupDir)
		helper.assertNoError(err, "Failed to open git repository")

		// Verify tag was created
		tags, err := repo.Tags()
		helper.assertNoError(err, "Failed to get tags")

		tagFound := false
		err = tags.ForEach(func(ref *plumbing.Reference) error {
			if ref.Name().Short() == result.Snapshot.ID {
				tagFound = true
			}
			return nil
		})
		helper.assertNoError(err, "Failed to iterate tags")

		if !tagFound {
			t.Errorf("Expected git tag %s to be created", result.Snapshot.ID)
		}

		// Verify files exist in git repo
		helper.assertFileExists(filepath.Join(backupDir, "openclaw.json"))
		helper.assertFileExists(filepath.Join(backupDir, "workspace", "SOUL.md"))
		helper.assertFileExists(filepath.Join(backupDir, "workspace", "skills", "analysis.js"))
	})

	// Test 2: Second backup creates new commit
	t.Run("SecondGitBackup", func(t *testing.T) {
		// Sleep to ensure different timestamp for snapshot ID
		time.Sleep(1100 * time.Millisecond)

		// Make changes
		helper.modifyAgentPersonality(agentDir, `# Agent Personality

I am a helpful, analytical, and thorough AI assistant.

## Core Values
- Precision and accuracy
- Detailed explanations
- User safety paramount
`)
		helper.addSkill(agentDir, "validation.js", "function validate() { return true; }")

		result, err := engine.Backup(false, "Added validation skill")
		helper.assertNoError(err, "Second git backup failed")

		if result.Skipped {
			t.Error("Backup with changes should not be skipped")
		}

		// Verify new tag was created
		repo, err := gogit.PlainOpen(backupDir)
		helper.assertNoError(err, "Failed to open git repository")

		tag, err := repo.Tag(result.Snapshot.ID)
		helper.assertNoError(err, "Tag should exist for snapshot ID")

		if tag == nil {
			t.Error("Tag should not be nil")
		}

		// Verify new file exists
		helper.assertFileExists(filepath.Join(backupDir, "workspace", "skills", "validation.js"))
	})

	// Test 3: Restore from git tag
	t.Run("RestoreFromGitTag", func(t *testing.T) {
		// Sleep to ensure safety backup gets different timestamp
		time.Sleep(1100 * time.Millisecond)

		// Restore to first snapshot
		err = engine.Restore(firstSnapshotID, false)
		helper.assertNoError(err, "Git restore failed")

		// Verify validation.js was removed (wasn't in first snapshot)
		helper.assertFileNotExists(filepath.Join(agentDir, "workspace", "skills", "validation.js"))

		// Verify personality was restored
		soulContent := helper.readFile(filepath.Join(agentDir, "workspace", "SOUL.md"))
		if contains(soulContent, "analytical") || contains(soulContent, "thorough") {
			t.Error("Modified personality still present after restore")
		}
		if !contains(soulContent, "helpful and concise") {
			t.Error("Original personality not restored")
		}
	})

	// Test 4: Verify git history is preserved
	t.Run("GitHistoryPreserved", func(t *testing.T) {
		repo, err := gogit.PlainOpen(backupDir)
		helper.assertNoError(err, "Failed to open git repository")

		// Get commit log
		ref, err := repo.Head()
		helper.assertNoError(err, "Failed to get HEAD")

		commits, err := repo.Log(&gogit.LogOptions{From: ref.Hash()})
		helper.assertNoError(err, "Failed to get commit log")

		commitCount := 0
		err = commits.ForEach(func(c *object.Commit) error {
			commitCount++
			return nil
		})
		helper.assertNoError(err, "Failed to iterate commits")

		// Should have at least 3 commits (initial + 2 backups + safety backup from restore)
		if commitCount < 3 {
			t.Errorf("Expected at least 3 commits in git history, got %d", commitCount)
		}
	})
}

// TestGitBackup_TagNaming tests that git tags match snapshot IDs
func TestGitBackup_TagNaming(t *testing.T) {
	helper := newTestDataHelper(t)

	agentDir := helper.createOpenClawAgent("tag-agent")
	backupDir := helper.createBackupDestination("git-tags")

	// Initialize git repository
	_, err := gogit.PlainInit(backupDir, false)
	helper.assertNoError(err, "Failed to initialize git repository")

	cfg := &config.Config{
		OpenclawPath: agentDir,
		Destination: &config.DestinationConfig{
			Type: "git",
			Path: backupDir,
		},
		Options: config.BackupOptions{
			Exclude: []string{},
		},
	}

	engine, err := NewBackupEngine(cfg)
	helper.assertNoError(err, "NewBackupEngine failed")

	// Create several backups
	snapshotIDs := make([]string, 0)

	for i := 0; i < 3; i++ {
		// Make a change
		helper.writeFile(filepath.Join(agentDir, "workspace", "file"+string(rune('A'+i))+".txt"), "content")

		result, err := engine.Backup(false, "Backup "+string(rune('A'+i)))
		helper.assertNoError(err, "Backup failed")

		snapshotIDs = append(snapshotIDs, result.Snapshot.ID)
	}

	// Verify all tags exist
	repo, err := gogit.PlainOpen(backupDir)
	helper.assertNoError(err, "Failed to open git repository")

	for _, snapshotID := range snapshotIDs {
		tag, err := repo.Tag(snapshotID)
		helper.assertNoError(err, "Tag should exist for snapshot "+snapshotID)

		if tag == nil {
			t.Errorf("Tag should not be nil for snapshot %s", snapshotID)
		}
	}
}

// TestGitBackup_DeduplicationBehavior tests git's internal deduplication
func TestGitBackup_DeduplicationBehavior(t *testing.T) {
	helper := newTestDataHelper(t)

	agentDir := helper.createOpenClawAgent("dedup-agent")
	backupDir := helper.createBackupDestination("git-dedup")

	// Initialize git repository
	_, err := gogit.PlainInit(backupDir, false)
	helper.assertNoError(err, "Failed to initialize git repository")

	cfg := &config.Config{
		OpenclawPath: agentDir,
		Destination: &config.DestinationConfig{
			Type: "git",
			Path: backupDir,
		},
		Options: config.BackupOptions{
			Exclude: []string{},
		},
	}

	engine, err := NewBackupEngine(cfg)
	helper.assertNoError(err, "NewBackupEngine failed")

	// Backup 1: Initial state
	result1, err := engine.Backup(false, "Initial")
	helper.assertNoError(err, "Backup 1 failed")

	// Get .git directory size after first backup
	gitDir1 := filepath.Join(backupDir, ".git")
	size1 := helper.getDirSize(gitDir1)

	// Backup 2: Change only one file
	helper.modifySkill(agentDir, "analysis.js", "modified content")
	result2, err := engine.Backup(false, "Modified one file")
	helper.assertNoError(err, "Backup 2 failed")

	// Get .git directory size after second backup
	size2 := helper.getDirSize(gitDir1)

	// Git's deduplication means size increase should be minimal
	// (much less than the full agent directory size)
	agentSize := helper.getDirSize(agentDir)
	sizeIncrease := size2 - size1

	// Size increase should be less than 10% of full agent size
	// (accounting for git metadata overhead)
	if sizeIncrease > agentSize/10 {
		t.Logf("Warning: Git may not be deduplicating efficiently")
		t.Logf("Agent size: %d bytes", agentSize)
		t.Logf("Git size increase: %d bytes", sizeIncrease)
		t.Logf("Expected much less than agent size due to deduplication")
	}

	// Verify both snapshots exist
	if result1.Snapshot == nil || result2.Snapshot == nil {
		t.Fatal("Snapshots should not be nil")
	}
}

// TestGitBackup_CommitMessages tests that commit messages are meaningful
func TestGitBackup_CommitMessages(t *testing.T) {
	helper := newTestDataHelper(t)

	agentDir := helper.createOpenClawAgent("commit-agent")
	backupDir := helper.createBackupDestination("git-commits")

	// Initialize git repository
	_, err := gogit.PlainInit(backupDir, false)
	helper.assertNoError(err, "Failed to initialize git repository")

	cfg := &config.Config{
		OpenclawPath: agentDir,
		Destination: &config.DestinationConfig{
			Type: "git",
			Path: backupDir,
		},
		Options: config.BackupOptions{
			Exclude: []string{},
		},
	}

	engine, err := NewBackupEngine(cfg)
	helper.assertNoError(err, "NewBackupEngine failed")

	// Create backup with custom message
	customMessage := "Fixed critical security vulnerability in skill execution"
	result, err := engine.Backup(false, customMessage)
	helper.assertNoError(err, "Backup failed")

	// Open repository and check commit message
	repo, err := gogit.PlainOpen(backupDir)
	helper.assertNoError(err, "Failed to open git repository")

	ref, err := repo.Head()
	helper.assertNoError(err, "Failed to get HEAD")

	commit, err := repo.CommitObject(ref.Hash())
	helper.assertNoError(err, "Failed to get commit")

	// Verify commit message contains our custom message or snapshot ID
	commitMsg := commit.Message
	if !contains(commitMsg, customMessage) && !contains(commitMsg, result.Snapshot.ID) {
		t.Errorf("Commit message should contain custom message or snapshot ID\nGot: %s", commitMsg)
	}
}

// getDirSize calculates the total size of a directory
func (h *testDataHelper) getDirSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

// TestGitBackup_BranchManagement tests git branch handling
func TestGitBackup_BranchManagement(t *testing.T) {
	helper := newTestDataHelper(t)

	agentDir := helper.createOpenClawAgent("branch-agent")
	backupDir := helper.createBackupDestination("git-branches")

	// Initialize git repository
	repo, err := gogit.PlainInit(backupDir, false)
	helper.assertNoError(err, "Failed to initialize git repository")

	cfg := &config.Config{
		OpenclawPath: agentDir,
		Destination: &config.DestinationConfig{
			Type: "git",
			Path: backupDir,
		},
		Options: config.BackupOptions{
			Exclude: []string{},
		},
	}

	engine, err := NewBackupEngine(cfg)
	helper.assertNoError(err, "NewBackupEngine failed")

	// Create initial backup
	_, err = engine.Backup(false, "Initial backup on main branch")
	helper.assertNoError(err, "Backup failed")

	// Verify we're on expected branch (usually "main" or "master")
	ref, err := repo.Head()
	helper.assertNoError(err, "Failed to get HEAD")

	branchName := ref.Name().Short()
	if branchName != "main" && branchName != "master" {
		t.Logf("Note: Branch name is %s (expected main or master)", branchName)
	}
}

// TestGitBackup_RemotePushBehavior tests behavior when git remote is configured
// Note: This test doesn't actually push to a real remote, it just verifies
// the local git state is correct
func TestGitBackup_RemotePushBehavior(t *testing.T) {
	helper := newTestDataHelper(t)

	agentDir := helper.createOpenClawAgent("remote-agent")
	backupDir := helper.createBackupDestination("git-remote")

	// Initialize git repository
	repo, err := gogit.PlainInit(backupDir, false)
	helper.assertNoError(err, "Failed to initialize git repository")

	// Note: We don't actually configure a remote here since that would require
	// a real git server. This test just verifies the backup works with a git repo.

	cfg := &config.Config{
		OpenclawPath: agentDir,
		Destination: &config.DestinationConfig{
			Type: "git",
			Path: backupDir,
		},
		Options: config.BackupOptions{
			Exclude: []string{},
		},
	}

	engine, err := NewBackupEngine(cfg)
	helper.assertNoError(err, "NewBackupEngine failed")

	// Create backup
	result, err := engine.Backup(false, "Backup that would be pushed to remote")
	helper.assertNoError(err, "Backup failed")

	// Verify backup succeeded even without remote
	if result.Snapshot == nil {
		t.Fatal("Snapshot should not be nil")
	}

	// Verify tag exists locally
	tag, err := repo.Tag(result.Snapshot.ID)
	helper.assertNoError(err, "Tag should exist locally")

	if tag == nil {
		t.Error("Tag should not be nil")
	}
}
