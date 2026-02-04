package backup

import (
	"testing"
	"time"

	"github.com/bulletproof-bot/backup/internal/config"
	"github.com/bulletproof-bot/backup/internal/types"
)

func TestCalculatePruneTargets_KeepLast(t *testing.T) {
	now := time.Now()
	snapshots := []*types.SnapshotInfo{
		{ID: "20240101-120000-000", Timestamp: now.AddDate(0, 0, -5), FileCount: 10},
		{ID: "20240102-120000-000", Timestamp: now.AddDate(0, 0, -4), FileCount: 10},
		{ID: "20240103-120000-000", Timestamp: now.AddDate(0, 0, -3), FileCount: 10},
		{ID: "20240104-120000-000", Timestamp: now.AddDate(0, 0, -2), FileCount: 10},
		{ID: "20240105-120000-000", Timestamp: now.AddDate(0, 0, -1), FileCount: 10},
	}

	policy := config.RetentionPolicy{
		Enabled:  true,
		KeepLast: 3,
	}

	result, err := CalculatePruneTargets(snapshots, policy)
	if err != nil {
		t.Fatalf("CalculatePruneTargets failed: %v", err)
	}

	if len(result.SnapshotsToKeep) != 3 {
		t.Errorf("Expected 3 snapshots to keep, got %d", len(result.SnapshotsToKeep))
	}

	if len(result.SnapshotsToDelete) != 2 {
		t.Errorf("Expected 2 snapshots to delete, got %d", len(result.SnapshotsToDelete))
	}

	// Verify newest snapshots are kept
	if result.SnapshotsToKeep[0].ID != "20240105-120000-000" {
		t.Errorf("Expected newest snapshot to be kept first")
	}
}

func TestCalculatePruneTargets_KeepDaily(t *testing.T) {
	now := time.Now()

	// Create snapshots: 2 per day for 5 days
	snapshots := []*types.SnapshotInfo{
		{ID: "20240101-120000-000", Timestamp: now.AddDate(0, 0, -5), FileCount: 10},
		{ID: "20240101-180000-000", Timestamp: now.AddDate(0, 0, -5).Add(6 * time.Hour), FileCount: 10},
		{ID: "20240102-120000-000", Timestamp: now.AddDate(0, 0, -4), FileCount: 10},
		{ID: "20240102-180000-000", Timestamp: now.AddDate(0, 0, -4).Add(6 * time.Hour), FileCount: 10},
		{ID: "20240103-120000-000", Timestamp: now.AddDate(0, 0, -3), FileCount: 10},
		{ID: "20240103-180000-000", Timestamp: now.AddDate(0, 0, -3).Add(6 * time.Hour), FileCount: 10},
		{ID: "20240104-120000-000", Timestamp: now.AddDate(0, 0, -2), FileCount: 10},
		{ID: "20240104-180000-000", Timestamp: now.AddDate(0, 0, -2).Add(6 * time.Hour), FileCount: 10},
		{ID: "20240105-120000-000", Timestamp: now.AddDate(0, 0, -1), FileCount: 10},
		{ID: "20240105-180000-000", Timestamp: now.AddDate(0, 0, -1).Add(6 * time.Hour), FileCount: 10},
	}

	policy := config.RetentionPolicy{
		Enabled:   true,
		KeepDaily: 5,
	}

	result, err := CalculatePruneTargets(snapshots, policy)
	if err != nil {
		t.Fatalf("CalculatePruneTargets failed: %v", err)
	}

	// Should keep one snapshot per day (5 days)
	if len(result.SnapshotsToKeep) != 5 {
		t.Errorf("Expected 5 snapshots to keep (one per day), got %d", len(result.SnapshotsToKeep))
	}

	if len(result.SnapshotsToDelete) != 5 {
		t.Errorf("Expected 5 snapshots to delete (duplicates), got %d", len(result.SnapshotsToDelete))
	}
}

func TestCalculatePruneTargets_KeepWeekly(t *testing.T) {
	now := time.Now()

	// Create snapshots spanning 4 weeks
	snapshots := []*types.SnapshotInfo{
		{ID: "20240101-120000-000", Timestamp: now.AddDate(0, 0, -28), FileCount: 10},
		{ID: "20240108-120000-000", Timestamp: now.AddDate(0, 0, -21), FileCount: 10},
		{ID: "20240115-120000-000", Timestamp: now.AddDate(0, 0, -14), FileCount: 10},
		{ID: "20240122-120000-000", Timestamp: now.AddDate(0, 0, -7), FileCount: 10},
	}

	policy := config.RetentionPolicy{
		Enabled:    true,
		KeepWeekly: 4,
	}

	result, err := CalculatePruneTargets(snapshots, policy)
	if err != nil {
		t.Fatalf("CalculatePruneTargets failed: %v", err)
	}

	// Should keep all 4 snapshots (one per week)
	if len(result.SnapshotsToKeep) != 4 {
		t.Errorf("Expected 4 snapshots to keep (one per week), got %d", len(result.SnapshotsToKeep))
	}

	if len(result.SnapshotsToDelete) != 0 {
		t.Errorf("Expected 0 snapshots to delete, got %d", len(result.SnapshotsToDelete))
	}
}

func TestCalculatePruneTargets_KeepMonthly(t *testing.T) {
	now := time.Now()

	// Create snapshots spanning 3 months
	snapshots := []*types.SnapshotInfo{
		{ID: "20240101-120000-000", Timestamp: now.AddDate(0, -3, 0), FileCount: 10},
		{ID: "20240201-120000-000", Timestamp: now.AddDate(0, -2, 0), FileCount: 10},
		{ID: "20240301-120000-000", Timestamp: now.AddDate(0, -1, 0), FileCount: 10},
	}

	policy := config.RetentionPolicy{
		Enabled:     true,
		KeepMonthly: 3,
	}

	result, err := CalculatePruneTargets(snapshots, policy)
	if err != nil {
		t.Fatalf("CalculatePruneTargets failed: %v", err)
	}

	// Should keep all 3 snapshots (one per month)
	if len(result.SnapshotsToKeep) != 3 {
		t.Errorf("Expected 3 snapshots to keep (one per month), got %d", len(result.SnapshotsToKeep))
	}

	if len(result.SnapshotsToDelete) != 0 {
		t.Errorf("Expected 0 snapshots to delete, got %d", len(result.SnapshotsToDelete))
	}
}

func TestCalculatePruneTargets_CombinedPolicy(t *testing.T) {
	now := time.Now()

	// Create many snapshots over time
	snapshots := []*types.SnapshotInfo{}
	for i := 0; i < 30; i++ {
		snapshots = append(snapshots, &types.SnapshotInfo{
			ID:        types.GenerateID(now.AddDate(0, 0, -i)),
			Timestamp: now.AddDate(0, 0, -i),
			FileCount: 10,
		})
	}

	policy := config.RetentionPolicy{
		Enabled:    true,
		KeepLast:   5, // Keep last 5
		KeepDaily:  7, // Keep daily for 7 days
		KeepWeekly: 4, // Keep weekly for 4 weeks
	}

	result, err := CalculatePruneTargets(snapshots, policy)
	if err != nil {
		t.Fatalf("CalculatePruneTargets failed: %v", err)
	}

	// At least 7 should be kept (last 5 + daily for 7 days with overlap)
	if len(result.SnapshotsToKeep) < 7 {
		t.Errorf("Expected at least 7 snapshots to keep, got %d", len(result.SnapshotsToKeep))
	}

	// Some should be deleted
	if len(result.SnapshotsToDelete) == 0 {
		t.Errorf("Expected some snapshots to be deleted")
	}

	// Total should equal original count
	if len(result.SnapshotsToKeep)+len(result.SnapshotsToDelete) != 30 {
		t.Errorf("Total snapshots mismatch: %d + %d != 30",
			len(result.SnapshotsToKeep), len(result.SnapshotsToDelete))
	}
}

func TestCalculatePruneTargets_EmptyList(t *testing.T) {
	policy := config.RetentionPolicy{
		Enabled:  true,
		KeepLast: 3,
	}

	result, err := CalculatePruneTargets([]*types.SnapshotInfo{}, policy)
	if err != nil {
		t.Fatalf("CalculatePruneTargets failed: %v", err)
	}

	if len(result.SnapshotsToKeep) != 0 {
		t.Errorf("Expected 0 snapshots to keep, got %d", len(result.SnapshotsToKeep))
	}

	if len(result.SnapshotsToDelete) != 0 {
		t.Errorf("Expected 0 snapshots to delete, got %d", len(result.SnapshotsToDelete))
	}
}

func TestCalculatePruneTargets_PolicyDisabled(t *testing.T) {
	snapshots := []*types.SnapshotInfo{
		{ID: "20240101-120000-000", Timestamp: time.Now(), FileCount: 10},
	}

	policy := config.RetentionPolicy{
		Enabled: false,
	}

	_, err := CalculatePruneTargets(snapshots, policy)
	if err == nil {
		t.Errorf("Expected error when policy is disabled")
	}
}
