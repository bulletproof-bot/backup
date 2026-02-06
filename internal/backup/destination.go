package backup

import "github.com/bulletproof-bot/backup/internal/types"

// Destination is an abstract interface for backup destinations
type Destination interface {
	Save(sourcePath string, snapshot *types.Snapshot, message string) error
	GetLastSnapshot() (*types.Snapshot, error)
	GetSnapshot(id string) (*types.Snapshot, error)
	ListSnapshots() ([]*types.SnapshotInfo, error)
	Restore(snapshotID string, targetPath string) error
	Validate() error
	// GetSnapshotPath returns the filesystem path where a snapshot's files are stored
	// Returns empty string if not applicable (e.g., git destination)
	GetSnapshotPath(id string) string
	// DeleteSnapshot deletes a snapshot by ID
	DeleteSnapshot(id string) error
}
