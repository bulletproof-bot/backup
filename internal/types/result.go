package types

import (
	"fmt"
	"time"
)

// BackupResult represents the result of a backup operation
type BackupResult struct {
	Snapshot *Snapshot
	Diff     *SnapshotDiff
	Skipped  bool
	DryRun   bool
}

// SnapshotInfo provides basic information about a snapshot (for listing)
type SnapshotInfo struct {
	ID        string
	Timestamp time.Time
	Message   string
	FileCount int
}

// String returns a string representation of snapshot info
func (si *SnapshotInfo) String() string {
	msg := ""
	if si.Message != "" {
		msg = " - " + si.Message
	}
	return fmt.Sprintf("%s (%d files)%s", si.ID, si.FileCount, msg)
}
