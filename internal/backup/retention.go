package backup

import (
	"fmt"
	"sort"
	"time"

	"github.com/bulletproof-bot/backup/internal/config"
	"github.com/bulletproof-bot/backup/internal/types"
)

// PruneResult contains the results of a prune operation
type PruneResult struct {
	SnapshotsToKeep   []*types.SnapshotInfo
	SnapshotsToDelete []*types.SnapshotInfo
	TotalSnapshots    int
}

// CalculatePruneTargets determines which snapshots to keep and which to delete based on retention policy
func CalculatePruneTargets(snapshots []*types.SnapshotInfo, policy config.RetentionPolicy) (*PruneResult, error) {
	if !policy.Enabled {
		return nil, fmt.Errorf("retention policy is not enabled")
	}

	if len(snapshots) == 0 {
		return &PruneResult{
			SnapshotsToKeep:   []*types.SnapshotInfo{},
			SnapshotsToDelete: []*types.SnapshotInfo{},
			TotalSnapshots:    0,
		}, nil
	}

	// Sort snapshots by timestamp (newest first)
	sortedSnapshots := make([]*types.SnapshotInfo, len(snapshots))
	copy(sortedSnapshots, snapshots)
	sort.Slice(sortedSnapshots, func(i, j int) bool {
		return sortedSnapshots[i].Timestamp.After(sortedSnapshots[j].Timestamp)
	})

	// Track which snapshots to keep (use map for efficient lookups)
	toKeep := make(map[string]bool)

	// Apply keep-last policy
	if policy.KeepLast > 0 {
		for i := 0; i < len(sortedSnapshots) && i < policy.KeepLast; i++ {
			toKeep[sortedSnapshots[i].ID] = true
		}
	}

	// Apply keep-daily policy
	if policy.KeepDaily > 0 {
		keepDailySnapshots(sortedSnapshots, policy.KeepDaily, toKeep)
	}

	// Apply keep-weekly policy
	if policy.KeepWeekly > 0 {
		keepWeeklySnapshots(sortedSnapshots, policy.KeepWeekly, toKeep)
	}

	// Apply keep-monthly policy
	if policy.KeepMonthly > 0 {
		keepMonthlySnapshots(sortedSnapshots, policy.KeepMonthly, toKeep)
	}

	// Build result lists
	result := &PruneResult{
		SnapshotsToKeep:   []*types.SnapshotInfo{},
		SnapshotsToDelete: []*types.SnapshotInfo{},
		TotalSnapshots:    len(snapshots),
	}

	for _, snapshot := range sortedSnapshots {
		if toKeep[snapshot.ID] {
			result.SnapshotsToKeep = append(result.SnapshotsToKeep, snapshot)
		} else {
			result.SnapshotsToDelete = append(result.SnapshotsToDelete, snapshot)
		}
	}

	return result, nil
}

// keepDailySnapshots keeps one snapshot per day for the specified number of days
func keepDailySnapshots(snapshots []*types.SnapshotInfo, days int, toKeep map[string]bool) {
	if days <= 0 {
		return
	}

	// Track which days we've seen
	seenDays := make(map[string]bool)
	cutoffDate := time.Now().AddDate(0, 0, -days)

	for _, snapshot := range snapshots {
		if snapshot.Timestamp.Before(cutoffDate) {
			continue
		}

		dayKey := snapshot.Timestamp.Format("2006-01-02")
		if !seenDays[dayKey] {
			seenDays[dayKey] = true
			toKeep[snapshot.ID] = true
		}
	}
}

// keepWeeklySnapshots keeps one snapshot per week for the specified number of weeks
func keepWeeklySnapshots(snapshots []*types.SnapshotInfo, weeks int, toKeep map[string]bool) {
	if weeks <= 0 {
		return
	}

	// Track which weeks we've seen (using ISO week year and week number)
	seenWeeks := make(map[string]bool)
	// Add a small buffer to include snapshots on the cutoff boundary
	cutoffDate := time.Now().AddDate(0, 0, -weeks*7).Add(-time.Hour)

	for _, snapshot := range snapshots {
		if snapshot.Timestamp.Before(cutoffDate) {
			continue
		}

		year, week := snapshot.Timestamp.ISOWeek()
		weekKey := fmt.Sprintf("%d-W%02d", year, week)
		if !seenWeeks[weekKey] {
			seenWeeks[weekKey] = true
			toKeep[snapshot.ID] = true
		}
	}
}

// keepMonthlySnapshots keeps one snapshot per month for the specified number of months
func keepMonthlySnapshots(snapshots []*types.SnapshotInfo, months int, toKeep map[string]bool) {
	if months <= 0 {
		return
	}

	// Track which months we've seen
	seenMonths := make(map[string]bool)
	// Add a small buffer to include snapshots on the cutoff boundary
	cutoffDate := time.Now().AddDate(0, -months, 0).Add(-time.Hour)

	for _, snapshot := range snapshots {
		if snapshot.Timestamp.Before(cutoffDate) {
			continue
		}

		monthKey := snapshot.Timestamp.Format("2006-01")
		if !seenMonths[monthKey] {
			seenMonths[monthKey] = true
			toKeep[snapshot.ID] = true
		}
	}
}

// Prune deletes snapshots according to the retention policy
func (e *BackupEngine) Prune(dryRun bool) (*PruneResult, error) {
	if !e.config.Retention.Enabled {
		return nil, fmt.Errorf("retention policy is not enabled in configuration")
	}

	// Get all snapshots
	snapshots, err := e.ListBackups()
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}

	// Calculate what to keep and what to delete
	result, err := CalculatePruneTargets(snapshots, e.config.Retention)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate prune targets: %w", err)
	}

	if dryRun {
		return result, nil
	}

	// Delete snapshots
	for _, snapshot := range result.SnapshotsToDelete {
		if err := e.destination.DeleteSnapshot(snapshot.ID); err != nil {
			return nil, fmt.Errorf("failed to delete snapshot %s: %w", snapshot.ID, err)
		}
	}

	return result, nil
}
