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
}
