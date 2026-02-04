package destinations

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bulletproof-bot/backup/internal/types"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// GitDestination stores backups as commits in a git repository.
// Each backup is a commit with all files copied to the repo.
// Automatically handles initializing repo if needed, committing changes,
// and pushing to remote if configured.
type GitDestination struct {
	RepoPath  string
	isRemote  bool
	validated bool
	repo      *git.Repository
}

// NewGitDestination creates a new git destination
func NewGitDestination(repoPath string) *GitDestination {
	isRemote := strings.HasPrefix(repoPath, "git@") ||
		strings.HasPrefix(repoPath, "https://") ||
		strings.HasPrefix(repoPath, "ssh://")

	return &GitDestination{
		RepoPath: repoPath,
		isRemote: isRemote,
	}
}

func (d *GitDestination) localPath() string {
	if d.isRemote {
		// Clone to a local cache directory
		homeDir, _ := os.UserHomeDir()
		repoName := filepath.Base(strings.TrimSuffix(d.RepoPath, ".git"))
		return filepath.Join(homeDir, ".cache", "bulletproof", "repos", repoName)
	}
	return d.RepoPath
}

// Validate ensures the git repository is properly configured
func (d *GitDestination) Validate() error {
	if d.validated {
		return nil
	}

	localPath := d.localPath()

	if d.isRemote {
		// Clone or open the remote repo
		if err := d.ensureCloned(); err != nil {
			return err
		}
	} else {
		// Check if local path exists and is a git repo
		repo, err := git.PlainOpen(localPath)
		if err != nil {
			// Initialize new repo
			if err := d.initRepo(); err != nil {
				return err
			}
		} else {
			d.repo = repo
		}
	}

	d.validated = true
	return nil
}

func (d *GitDestination) ensureCloned() error {
	localPath := d.localPath()

	// Check if already cloned
	if repo, err := git.PlainOpen(localPath); err == nil {
		d.repo = repo
		// Pull latest
		fmt.Println("  Pulling latest from remote...")
		worktree, err := repo.Worktree()
		if err != nil {
			return fmt.Errorf("failed to get worktree: %w", err)
		}
		if err := worktree.Pull(&git.PullOptions{}); err != nil && err != git.NoErrAlreadyUpToDate {
			return fmt.Errorf("failed to pull: %w", err)
		}
		return nil
	}

	// Clone the repository
	fmt.Println("  Cloning repository...")
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	repo, err := git.PlainClone(localPath, false, &git.CloneOptions{
		URL: d.RepoPath,
	})
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	d.repo = repo
	return nil
}

func (d *GitDestination) initRepo() error {
	localPath := d.localPath()

	// Create directory if it doesn't exist
	if err := os.MkdirAll(localPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	fmt.Println("  Initializing git repository...")
	repo, err := git.PlainInit(localPath, false)
	if err != nil {
		return fmt.Errorf("failed to initialize repository: %w", err)
	}

	d.repo = repo

	// Create initial .gitignore
	gitignorePath := filepath.Join(localPath, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte(".DS_Store\n*.log\n"), 0644); err != nil {
		return fmt.Errorf("failed to create .gitignore: %w", err)
	}

	// Initial commit
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	if _, err := worktree.Add(".gitignore"); err != nil {
		return fmt.Errorf("failed to add .gitignore: %w", err)
	}

	if _, err := worktree.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Bulletproof Backup",
			Email: "backup@bulletproof.bot",
		},
	}); err != nil {
		return fmt.Errorf("failed to create initial commit: %w", err)
	}

	return nil
}

// Save saves a backup to the git repository
func (d *GitDestination) Save(sourcePath string, snapshot *types.Snapshot, message string) error {
	if err := d.Validate(); err != nil {
		return err
	}

	localPath := d.localPath()

	// Sync files
	fmt.Println("  Copying files to backup repository...")
	if err := d.syncFiles(sourcePath, localPath, snapshot); err != nil {
		return err
	}

	// Save snapshot metadata
	metaFile := filepath.Join(localPath, ".bulletproof", "snapshot.json")
	if err := os.MkdirAll(filepath.Dir(metaFile), 0755); err != nil {
		return fmt.Errorf("failed to create metadata directory: %w", err)
	}

	snapshotJSON, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	if err := os.WriteFile(metaFile, snapshotJSON, 0644); err != nil {
		return fmt.Errorf("failed to write snapshot file: %w", err)
	}

	// Stage all changes
	worktree, err := d.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	if _, err := worktree.Add("."); err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}

	// Check if there are changes to commit
	status, err := worktree.Status()
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	if status.IsClean() {
		fmt.Println("  No changes to commit.")
		return nil
	}

	// Commit
	commitHash, err := worktree.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Bulletproof Backup",
			Email: "backup@bulletproof.bot",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	// Tag with snapshot ID
	if _, err := d.repo.CreateTag(snapshot.ID, commitHash, &git.CreateTagOptions{
		Message: message,
	}); err != nil {
		return fmt.Errorf("failed to create tag: %w", err)
	}

	// Push if remote
	if d.isRemote {
		fmt.Println("  Pushing to remote...")
		refSpec := config.RefSpec("refs/tags/*:refs/tags/*")
		if err := d.repo.Push(&git.PushOptions{
			RefSpecs: []config.RefSpec{refSpec},
		}); err != nil && err != git.NoErrAlreadyUpToDate {
			return fmt.Errorf("failed to push: %w", err)
		}
	}

	return nil
}

func (d *GitDestination) syncFiles(sourcePath, destPath string, snapshot *types.Snapshot) error {
	// Clear existing files (except .git and .bulletproof)
	entries, err := os.ReadDir(destPath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if name == ".git" || name == ".bulletproof" {
			continue
		}

		path := filepath.Join(destPath, name)
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("failed to remove %s: %w", path, err)
		}
	}

	// Copy all files from snapshot
	for filePath := range snapshot.Files {
		sourceFile := filepath.Join(sourcePath, filePath)
		destFile := filepath.Join(destPath, filePath)

		if err := copyFileGit(sourceFile, destFile); err != nil {
			return fmt.Errorf("failed to copy file %s: %w", filePath, err)
		}
	}

	return nil
}

func copyFileGit(src, dst string) error {
	// Read source file
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Create destination directory
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Write destination file
	if err := os.WriteFile(dst, data, 0644); err != nil {
		return fmt.Errorf("failed to write destination file: %w", err)
	}

	return nil
}

// GetLastSnapshot returns the most recent snapshot
func (d *GitDestination) GetLastSnapshot() (*types.Snapshot, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}

	localPath := d.localPath()
	metaFile := filepath.Join(localPath, ".bulletproof", "snapshot.json")

	data, err := os.ReadFile(metaFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read snapshot file: %w", err)
	}

	snapshot, err := types.FromJSON(data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot: %w", err)
	}

	return snapshot, nil
}

// GetSnapshot returns a specific snapshot by ID
func (d *GitDestination) GetSnapshot(id string) (*types.Snapshot, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}

	// Checkout the tag
	worktree, err := d.repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	tagRef, err := d.repo.Tag(id)
	if err != nil {
		return nil, fmt.Errorf("snapshot not found: %s", id)
	}

	if err := worktree.Checkout(&git.CheckoutOptions{
		Branch: tagRef.Name(),
	}); err != nil {
		return nil, fmt.Errorf("failed to checkout tag: %w", err)
	}

	return d.GetLastSnapshot()
}

// ListSnapshots returns all available snapshots
func (d *GitDestination) ListSnapshots() ([]*types.SnapshotInfo, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}

	// Get all tags
	tags, err := d.repo.Tags()
	if err != nil {
		return nil, fmt.Errorf("failed to get tags: %w", err)
	}

	snapshots := []*types.SnapshotInfo{}
	tags.ForEach(func(ref *plumbing.Reference) error {
		snapshots = append(snapshots, &types.SnapshotInfo{
			ID: ref.Name().Short(),
		})
		return nil
	})

	return snapshots, nil
}

// Restore restores files from a snapshot to the target path
func (d *GitDestination) Restore(snapshotID string, targetPath string) error {
	if err := d.Validate(); err != nil {
		return err
	}

	// Checkout the tag
	worktree, err := d.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	tagRef, err := d.repo.Tag(snapshotID)
	if err != nil {
		return fmt.Errorf("snapshot not found: %s", snapshotID)
	}

	if err := worktree.Checkout(&git.CheckoutOptions{
		Branch: tagRef.Name(),
	}); err != nil {
		return fmt.Errorf("failed to checkout tag: %w", err)
	}

	// First, collect all files that should exist after restore
	localPath := d.localPath()
	snapshotFiles := make(map[string]bool)
	err = filepath.Walk(localPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relativePath, err := filepath.Rel(localPath, path)
		if err != nil {
			return err
		}

		// Skip .git and .bulletproof
		if strings.HasPrefix(relativePath, ".git") || strings.HasPrefix(relativePath, ".bulletproof") {
			return nil
		}

		snapshotFiles[relativePath] = true
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to scan snapshot: %w", err)
	}

	// Remove files from target that don't exist in snapshot
	err = filepath.Walk(targetPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors on walk
		}

		if info.IsDir() {
			return nil
		}

		relativePath, err := filepath.Rel(targetPath, path)
		if err != nil {
			return nil
		}

		// Keep OpenClaw config files
		if relativePath == "openclaw.json" || strings.HasPrefix(relativePath, "workspace") {
			if !snapshotFiles[relativePath] {
				// File exists in target but not in snapshot - remove it
				if err := os.Remove(path); err != nil {
					return fmt.Errorf("failed to remove file %s: %w", relativePath, err)
				}
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to clean target directory: %w", err)
	}

	// Copy files from repo to target
	return filepath.Walk(localPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relativePath, err := filepath.Rel(localPath, path)
		if err != nil {
			return err
		}

		// Skip .git and .bulletproof
		if strings.HasPrefix(relativePath, ".git") || strings.HasPrefix(relativePath, ".bulletproof") {
			return nil
		}

		// Copy file
		destFile := filepath.Join(targetPath, relativePath)
		return copyFileGit(path, destFile)
	})
}
