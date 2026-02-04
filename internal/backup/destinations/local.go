package destinations

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bulletproof-bot/backup/internal/types"
)

// LocalDestination stores backups as folders on the local filesystem.
// It can operate in two modes:
// - timestamped: Each backup creates a new folder (default)
// - overwrite: Overwrites the same folder (for sync services)
type LocalDestination struct {
	BasePath    string
	Timestamped bool
}

// NewLocalDestination creates a new local destination
func NewLocalDestination(basePath string, timestamped bool) *LocalDestination {
	return &LocalDestination{
		BasePath:    basePath,
		Timestamped: timestamped,
	}
}

func (d *LocalDestination) snapshotPath(id string) string {
	return filepath.Join(d.BasePath, id)
}

func (d *LocalDestination) metadataPath() string {
	return filepath.Join(d.BasePath, ".bulletproof")
}

// Validate ensures the destination is properly configured
func (d *LocalDestination) Validate() error {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(d.BasePath, 0755); err != nil {
		return fmt.Errorf("failed to create base directory: %w", err)
	}
	return nil
}

// Save saves a backup to the destination
func (d *LocalDestination) Save(sourcePath string, snapshot *types.Snapshot, message string) error {
	if err := d.Validate(); err != nil {
		return err
	}

	targetPath := d.BasePath
	if d.Timestamped {
		targetPath = d.snapshotPath(snapshot.ID)
	}

	if d.Timestamped {
		// Create new snapshot folder
		if err := os.MkdirAll(targetPath, 0755); err != nil {
			return fmt.Errorf("failed to create snapshot directory: %w", err)
		}
	} else {
		// Clear existing files for sync mode
		if err := d.clearExistingFiles(targetPath); err != nil {
			return fmt.Errorf("failed to clear existing files: %w", err)
		}
	}

	// Copy files
	fmt.Printf("  Copying %d files...\n", len(snapshot.Files))
	for filePath := range snapshot.Files {
		sourceFile := filepath.Join(sourcePath, filePath)
		destFile := filepath.Join(targetPath, filePath)

		if err := d.copyFile(sourceFile, destFile); err != nil {
			return fmt.Errorf("failed to copy file %s: %w", filePath, err)
		}
	}

	// Create .bulletproof directory within snapshot for self-contained structure
	if d.Timestamped {
		bulletproofDir := filepath.Join(targetPath, ".bulletproof")
		if err := os.MkdirAll(bulletproofDir, 0755); err != nil {
			return fmt.Errorf("failed to create .bulletproof directory: %w", err)
		}

		// Save snapshot.json in the snapshot's .bulletproof directory
		snapshotFile := filepath.Join(bulletproofDir, "snapshot.json")
		snapshotJSON, err := json.MarshalIndent(snapshot, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal snapshot: %w", err)
		}
		if err := os.WriteFile(snapshotFile, snapshotJSON, 0644); err != nil {
			return fmt.Errorf("failed to write snapshot file: %w", err)
		}

		// Copy config file to snapshot's .bulletproof directory for platform migration
		// Config path is stored in the engine, we need to pass it through
		// For now, we'll add this in the engine layer
	}

	// Also save metadata in central location for quick lookups
	metaDir := d.metadataPath()
	if err := os.MkdirAll(metaDir, 0755); err != nil {
		return fmt.Errorf("failed to create metadata directory: %w", err)
	}

	// Save snapshot info in central metadata
	snapshotFile := filepath.Join(metaDir, snapshot.ID+".json")
	snapshotJSON, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}
	if err := os.WriteFile(snapshotFile, snapshotJSON, 0644); err != nil {
		return fmt.Errorf("failed to write snapshot file: %w", err)
	}

	// Update latest pointer
	latestFile := filepath.Join(metaDir, "latest")
	if err := os.WriteFile(latestFile, []byte(snapshot.ID), 0644); err != nil {
		return fmt.Errorf("failed to write latest file: %w", err)
	}

	// Update index
	if err := d.updateIndex(snapshot, message); err != nil {
		return fmt.Errorf("failed to update index: %w", err)
	}

	fmt.Printf("  Backup saved to: %s\n", targetPath)
	return nil
}

func (d *LocalDestination) clearExistingFiles(targetPath string) error {
	entries, err := os.ReadDir(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.Name() == ".bulletproof" {
			continue
		}

		path := filepath.Join(targetPath, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("failed to remove %s: %w", path, err)
		}
	}

	return nil
}

func (d *LocalDestination) copyFile(src, dst string) error {
	// Open source file
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// Get source file info for permissions
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	// Create destination directory
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// If destination file exists and is readonly, make it writable first
	if destInfo, err := os.Stat(dst); err == nil {
		if destInfo.Mode().Perm()&0200 == 0 { // Check if not writable
			if err := os.Chmod(dst, 0644); err != nil {
				return fmt.Errorf("failed to make destination writable: %w", err)
			}
		}
	}

	// Create destination file
	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	// Copy contents
	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	// Restore original permissions
	if err := os.Chmod(dst, sourceInfo.Mode().Perm()); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	return nil
}

func (d *LocalDestination) updateIndex(snapshot *types.Snapshot, message string) error {
	indexFile := filepath.Join(d.metadataPath(), "index.json")

	var index []map[string]interface{}

	// Read existing index
	if data, err := os.ReadFile(indexFile); err == nil {
		if err := json.Unmarshal(data, &index); err != nil {
			// Ignore parse errors, start fresh
			index = []map[string]interface{}{}
		}
	}

	// Add new entry at the beginning
	newEntry := map[string]interface{}{
		"id":        snapshot.ID,
		"timestamp": snapshot.Timestamp,
		"message":   message,
		"fileCount": len(snapshot.Files),
	}
	index = append([]map[string]interface{}{newEntry}, index...)

	// Keep last 100 entries
	if len(index) > 100 {
		index = index[:100]
	}

	// Write index
	indexJSON, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	if err := os.WriteFile(indexFile, indexJSON, 0644); err != nil {
		return fmt.Errorf("failed to write index file: %w", err)
	}

	return nil
}

// GetLastSnapshot returns the most recent snapshot
func (d *LocalDestination) GetLastSnapshot() (*types.Snapshot, error) {
	latestFile := filepath.Join(d.metadataPath(), "latest")
	data, err := os.ReadFile(latestFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read latest file: %w", err)
	}

	latestID := strings.TrimSpace(string(data))
	return d.GetSnapshot(latestID)
}

// GetSnapshot returns a specific snapshot by ID
func (d *LocalDestination) GetSnapshot(id string) (*types.Snapshot, error) {
	snapshotFile := filepath.Join(d.metadataPath(), id+".json")
	data, err := os.ReadFile(snapshotFile)
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

// ListSnapshots returns all available snapshots
func (d *LocalDestination) ListSnapshots() ([]*types.SnapshotInfo, error) {
	indexFile := filepath.Join(d.metadataPath(), "index.json")
	data, err := os.ReadFile(indexFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []*types.SnapshotInfo{}, nil
		}
		return nil, fmt.Errorf("failed to read index file: %w", err)
	}

	var index []map[string]interface{}
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("failed to unmarshal index: %w", err)
	}

	snapshots := make([]*types.SnapshotInfo, 0, len(index))
	for _, entry := range index {
		id, _ := entry["id"].(string)
		timestamp, _ := entry["timestamp"].(string)
		message, _ := entry["message"].(string)
		fileCount, _ := entry["fileCount"].(float64)

		snapshots = append(snapshots, &types.SnapshotInfo{
			ID:        id,
			Timestamp: parseTimestamp(timestamp),
			Message:   message,
			FileCount: int(fileCount),
		})
	}

	return snapshots, nil
}

func parseTimestamp(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

// Restore restores files from a snapshot to the target path
func (d *LocalDestination) Restore(snapshotID string, targetPath string) error {
	snapshotPath := d.BasePath
	if d.Timestamped {
		snapshotPath = d.snapshotPath(snapshotID)
	}

	// Check if snapshot exists
	if _, err := os.Stat(snapshotPath); err != nil {
		return fmt.Errorf("snapshot not found: %s", snapshotID)
	}

	// First, collect all files that should exist after restore
	snapshotFiles := make(map[string]bool)
	err := filepath.Walk(snapshotPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relativePath, err := filepath.Rel(snapshotPath, path)
		if err != nil {
			return err
		}

		// Skip metadata
		if strings.HasPrefix(relativePath, ".bulletproof") {
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

	// Now copy all files from snapshot to target
	return filepath.Walk(snapshotPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Get relative path
		relativePath, err := filepath.Rel(snapshotPath, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Skip metadata
		if strings.HasPrefix(relativePath, ".bulletproof") {
			return nil
		}

		// Copy file
		targetFile := filepath.Join(targetPath, relativePath)
		if err := d.copyFile(path, targetFile); err != nil {
			return fmt.Errorf("failed to copy file %s: %w", relativePath, err)
		}

		return nil
	})
}
